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
				`"title":"Electrical"`, `"title":"Battery"`, `"title":"State Of Charge (live)"`,
				`"title":"Powertrain"`, `"title":"Engine"`, `"title":"Oil Pressure (history)"`, `"title":"Engine Speed (live)"`,
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
				{name: "A Section", signals: []topicSignal{{label: "Wheel Speed", detailLabel: "Wheel Speed", topic: "data/a-section/wheel-speed"}}, modules: []topicModule{{name: "Brake", signals: []topicSignal{{label: "Pressure", detailLabel: "Brake / Pressure", topic: "data/a-section/brake/pressure"}}}}},
				{name: "Z Section", signals: []topicSignal{{label: "Oil Temp", detailLabel: "Oil Temp", topic: "data/z_section/oil_temp"}}},
			},
		},
		{name: "rejects wrong prefix", topics: []SignalTopic{{Topic: "vehicle/powertrain/engine-speed"}}, wantError: `must start with "data/"`},
		{name: "rejects missing signal", topics: []SignalTopic{{Topic: "data/powertrain"}}, wantError: "section and signal"},
		{name: "rejects more than one module", topics: []SignalTopic{{Topic: "data/powertrain/engine/cooling/temperature"}}, wantError: "optionally preceded by one module"},
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

func TestBuildTelemetryDashboardUsesCompactModuleGrid(t *testing.T) {
	sections, err := parseSignalTopicHierarchy([]SignalTopic{
		{Topic: "data/battery/voltage"},
		{Topic: "data/battery/m1/s3"},
		{Topic: "data/battery/m1/s1"},
		{Topic: "data/battery/m1/s2"},
	})
	require.NoError(t, err)

	telemetry, err := buildTelemetryDashboard(sections)
	require.NoError(t, err)
	encoded, err := json.Marshal(telemetry)
	require.NoError(t, err)

	type gridPos struct {
		W int `json:"w"`
		X int `json:"x"`
	}
	type panel struct {
		Type    string  `json:"type"`
		Title   string  `json:"title"`
		GridPos gridPos `json:"gridPos"`
	}
	var generated struct {
		Panels []panel `json:"panels"`
	}
	require.NoError(t, json.Unmarshal(encoded, &generated))

	panelsByTitle := make(map[string]panel, len(generated.Panels))
	panelIndexes := make(map[string]int, len(generated.Panels))
	for index, panel := range generated.Panels {
		panelsByTitle[panel.Title] = panel
		panelIndexes[panel.Title] = index
	}
	assert.Equal(t, "row", panelsByTitle["Battery"].Type)
	assert.Equal(t, "row", panelsByTitle["M1"].Type)
	assert.Equal(t, gridPos{W: 12, X: 0}, panelsByTitle["Voltage (live)"].GridPos)
	assert.Equal(t, gridPos{W: 12, X: 12}, panelsByTitle["Voltage (history)"].GridPos)
	assert.Equal(t, gridPos{W: 6, X: 0}, panelsByTitle["S1 (live)"].GridPos)
	assert.Equal(t, gridPos{W: 6, X: 6}, panelsByTitle["S1 (history)"].GridPos)
	assert.Equal(t, gridPos{W: 6, X: 12}, panelsByTitle["S2 (live)"].GridPos)
	assert.Equal(t, gridPos{W: 6, X: 18}, panelsByTitle["S2 (history)"].GridPos)
	assert.Equal(t, gridPos{W: 6, X: 0}, panelsByTitle["S3 (live)"].GridPos)
	assert.Equal(t, gridPos{W: 6, X: 6}, panelsByTitle["S3 (history)"].GridPos)
	assert.Less(t, panelIndexes["Voltage (live)"], panelIndexes["M1"])
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
