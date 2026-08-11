package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDashboardsWithSignalTopics(t *testing.T) {
	tests := []struct {
		name       string
		topics     []SignalTopic
		wantCount  int
		wantPieces []string
	}{
		{
			name: "stable overview and detail dashboards",
			topics: []SignalTopic{
				{Signal: "OilPressure", Topic: "data/powertrain/engine/oil-pressure"},
				{Signal: "StateOfCharge", Topic: "data/electrical/battery/state-of-charge"},
				{Signal: "EngineSpeed", Topic: "data/powertrain/engine-speed"},
			},
			wantCount: 4,
			wantPieces: []string{
				`"uid":"generated-telemetry"`, `"refresh":"1s"`, `"from":"now-15m"`, `"type":"alertlist"`,
				`"title":"Active Alerts"`, `"viewMode":"list"`, `"dashboardAlerts":true`,
				`"stateFilter":{"firing":true,"pending":true,"recovering":false,"noData":true,"normal":false,"error":true}`,
				`"title":"Electrical"`, `"title":"Powertrain"`, `"title":"Battery / State Of Charge (live)"`,
				`"title":"Engine / Oil Pressure (history)"`, `"title":"Engine Speed (live)"`,
				"grafana-mqtt-datasource", "influxdb-datasource", `"graphMode":"area"`, `"noValue":"No data"`,
				`"collapsed":false`, `"title":"Open detail"`, `from=${__from}\u0026to=${__to}`,
			},
		},
		{name: "empty topic list still creates overview", topics: nil, wantCount: 1, wantPieces: []string{`"uid":"generated-telemetry"`, `"title":"Active Alerts"`}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dashboards, err := createDashboardsWithSignalTopics(test.topics)
			require.NoError(t, err)
			require.Len(t, dashboards, test.wantCount)
			telemetryDashboard, ok := dashboards["telemetry"]
			require.True(t, ok)
			encoded, err := json.Marshal(telemetryDashboard)
			require.NoError(t, err)
			generated := string(encoded)
			for _, expected := range test.wantPieces {
				assert.Contains(t, generated, expected)
			}
			alertIndex := strings.Index(generated, `"title":"Active Alerts"`)
			require.NotEqual(t, -1, alertIndex)
			if electricalIndex := strings.Index(generated, `"title":"Electrical"`); electricalIndex >= 0 {
				assert.Less(t, alertIndex, electricalIndex)
			}

			for _, signalTopic := range test.topics {
				topic := signalTopic.Topic
				linkURL := strings.ReplaceAll(detailDashboardURL(topic), "&", `\u0026`)
				assert.Equal(t, 2, strings.Count(generated, linkURL))
				key := detailDashboardKey(topic)
				detailDashboard, exists := dashboards[key]
				require.True(t, exists)
				detailJSON, err := json.Marshal(detailDashboard)
				require.NoError(t, err)
				generatedDetail := string(detailJSON)
				for _, expected := range []string{`"uid":"` + key + `"`, `"refresh":"1s"`, `"from":"now-24h"`, topic, "grafana-mqtt-datasource", "influxdb-datasource"} {
					assert.Contains(t, generatedDetail, expected)
				}
			}
		})
	}
}

func TestParseSignalTopicHierarchy(t *testing.T) {
	tests := []struct {
		name      string
		topics    []SignalTopic
		want      []topicSection
		wantError string
	}{
		{
			name:   "groups and sorts sections and signals",
			topics: []SignalTopic{{Topic: "data/z_section/oil_temp"}, {Topic: "data/a-section/wheel-speed"}, {Topic: "data/a-section/brake/pressure"}},
			want: []topicSection{
				{name: "A Section", signals: []topicSignal{{label: "Brake / Pressure", topic: "data/a-section/brake/pressure"}, {label: "Wheel Speed", topic: "data/a-section/wheel-speed"}}},
				{name: "Z Section", signals: []topicSignal{{label: "Oil Temp", topic: "data/z_section/oil_temp"}}},
			},
		},
		{name: "rejects wrong prefix", topics: []SignalTopic{{Topic: "vehicle/powertrain/engine-speed"}}, wantError: `must start with "data/"`},
		{name: "rejects missing signal", topics: []SignalTopic{{Topic: "data/powertrain"}}, wantError: "section and signal"},
		{name: "rejects empty section", topics: []SignalTopic{{Topic: "data//engine-speed"}}, wantError: "levels cannot be empty"},
		{name: "rejects empty signal", topics: []SignalTopic{{Topic: "data/powertrain/"}}, wantError: "levels cannot be empty"},
		{name: "rejects duplicate", topics: []SignalTopic{{Topic: "data/powertrain/engine-speed"}, {Topic: "data/powertrain/engine-speed"}}, wantError: "duplicate topic"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseSignalTopicHierarchy(test.topics)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestHumanizeTopicSegment(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{name: "hyphens", input: "state-of-charge", want: "State Of Charge"},
		{name: "underscores", input: "oil_temp", want: "Oil Temp"},
		{name: "unicode", input: "énergie-level", want: "Énergie Level"},
		{name: "empty", input: "", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { assert.Equal(t, test.want, humanizeTopicSegment(test.input)) })
	}
}

func TestStableIdentifiersAndLinks(t *testing.T) {
	tests := []struct {
		name  string
		topic string
	}{
		{name: "powertrain signal", topic: "data/powertrain/engine-speed"},
		{name: "unicode signal", topic: "data/interior/température"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			live := stablePanelID(test.topic, 'l')
			history := stablePanelID(test.topic, 'h')
			assert.NotZero(t, live)
			assert.NotZero(t, history)
			assert.NotEqual(t, live, history)
			assert.Equal(t, live, stablePanelID(test.topic, 'l'))
			key := detailDashboardKey(test.topic)
			assert.LessOrEqual(t, len(key), 40)
			assert.Equal(t, key, detailDashboardKey(test.topic))
			assert.NotEqual(t, key, detailDashboardKey(test.topic+"-different"))
			assert.Equal(t, "/d/"+key+"?from=${__from}&to=${__to}", detailDashboardURL(test.topic))
		})
	}
}
