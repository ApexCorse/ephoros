package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ApexCorse/vera"
)

func TestCreateDashboardsWithSignalTopicsBuildsStableOverviewAndDetails(t *testing.T) {
	dashboards, err := createDashboardsWithSignalTopics([]vera.SignalTopic{
		{Signal: "OilPressure", Topic: "data/powertrain/engine/oil-pressure"},
		{Signal: "StateOfCharge", Topic: "data/electrical/battery/state-of-charge"},
		{Signal: "EngineSpeed", Topic: "data/powertrain/engine-speed"},
	})
	if err != nil {
		t.Fatalf("create dashboards: %v", err)
	}
	if len(dashboards) != 4 {
		t.Fatalf("dashboard count = %d, want 4", len(dashboards))
	}

	telemetryDashboard, ok := dashboards["telemetry"]
	if !ok {
		t.Fatal("telemetry dashboard is missing")
	}
	encoded, err := json.Marshal(telemetryDashboard)
	if err != nil {
		t.Fatalf("marshal dashboard: %v", err)
	}
	generated := string(encoded)

	for _, expected := range []string{
		`"uid":"generated-telemetry"`,
		`"refresh":"1s"`,
		`"from":"now-15m"`,
		`"type":"alertlist"`,
		`"title":"Active Alerts"`,
		`"viewMode":"list"`,
		`"groupMode":"default"`,
		`"sortOrder":3`,
		`"dashboardAlerts":true`,
		`"showInactiveAlerts":false`,
		`"stateFilter":{"firing":true,"pending":true,"recovering":false,"noData":true,"normal":false,"error":true}`,
		`"title":"Electrical"`,
		`"title":"Powertrain"`,
		`"title":"Battery / State Of Charge (live)"`,
		`"title":"Engine / Oil Pressure (history)"`,
		`"title":"Engine Speed (live)"`,
		"grafana-mqtt-datasource",
		"influxdb-datasource",
		"data/powertrain/engine/oil-pressure",
		`"graphMode":"area"`,
		`"noValue":"No data"`,
		`"collapsed":false`,
		`"title":"Open detail"`,
		`from=${__from}\u0026to=${__to}`,
	} {
		if !strings.Contains(generated, expected) {
			t.Errorf("generated dashboard does not contain %q", expected)
		}
	}
	if strings.Index(generated, `"title":"Active Alerts"`) > strings.Index(generated, `"title":"Electrical"`) {
		t.Error("alert list is not the first dashboard panel")
	}

	if strings.Index(generated, `"title":"Electrical"`) > strings.Index(generated, `"title":"Powertrain"`) {
		t.Error("sections are not in alphabetical order")
	}
	if strings.Index(generated, `"title":"Engine / Oil Pressure (live)"`) > strings.Index(generated, `"title":"Engine Speed (live)"`) {
		t.Error("signals are not in alphabetical order")
	}

	for _, topic := range []string{
		"data/powertrain/engine/oil-pressure",
		"data/electrical/battery/state-of-charge",
		"data/powertrain/engine-speed",
	} {
		linkURL := strings.ReplaceAll(detailDashboardURL(topic), "&", `\u0026`)
		if count := strings.Count(generated, linkURL); count != 2 {
			t.Errorf("overview links to detail dashboard for %q %d times, want 2", topic, count)
		}

		key := detailDashboardKey(topic)
		detailDashboard, ok := dashboards[key]
		if !ok {
			t.Errorf("detail dashboard for %q is missing", topic)
			continue
		}

		detailJSON, err := json.Marshal(detailDashboard)
		if err != nil {
			t.Errorf("marshal detail dashboard for %q: %v", topic, err)
			continue
		}
		generatedDetail := string(detailJSON)
		for _, expected := range []string{
			`"uid":"` + key + `"`,
			`"refresh":"1s"`,
			`"from":"now-24h"`,
			topic,
			"grafana-mqtt-datasource",
			"influxdb-datasource",
		} {
			if !strings.Contains(generatedDetail, expected) {
				t.Errorf("detail dashboard for %q does not contain %q", topic, expected)
			}
		}
	}
}

func TestParseSignalTopicHierarchyRejectsInvalidTopics(t *testing.T) {
	for _, topic := range []string{
		"vehicle/powertrain/engine-speed",
		"data/powertrain",
		"data//engine-speed",
		"data/powertrain/",
		"data/powertrain/engine-speed",
	} {
		t.Run(topic, func(t *testing.T) {
			topics := []vera.SignalTopic{{Topic: topic}}
			if topic == "data/powertrain/engine-speed" {
				topics = append(topics, vera.SignalTopic{Topic: topic})
			}
			_, err := parseSignalTopicHierarchy(topics)
			if err == nil {
				t.Errorf("parseSignalTopicHierarchy(%q) succeeded, want error", topic)
			}
		})
	}
}

func TestInfluxDBQueryUsesMQTTTopic(t *testing.T) {
	query, err := NewInfluxDBQueryBuilder("data/electrical/battery-voltage").Build()
	if err != nil {
		t.Fatalf("build query: %v", err)
	}

	influxQuery, ok := query.(InfluxDBQuery)
	if !ok {
		t.Fatalf("query type = %T, want InfluxDBQuery", query)
	}
	if !strings.Contains(influxQuery.Query, `r["topic"] == "data/electrical/battery-voltage"`) {
		t.Errorf("query does not filter by the MQTT topic: %s", influxQuery.Query)
	}
}

func TestStablePanelIDSeparatesPanelKinds(t *testing.T) {
	topic := "data/powertrain/engine-speed"
	live := stablePanelID(topic, 'l')
	history := stablePanelID(topic, 'h')
	if live == 0 || history == 0 {
		t.Fatal("panel IDs must be non-zero")
	}
	if live == history {
		t.Fatalf("live and history panel IDs collide: %d", live)
	}
	if stablePanelID(topic, 'l') != live {
		t.Fatal("panel ID is not deterministic")
	}
}

func TestDetailDashboardKeyAndLinkAreStable(t *testing.T) {
	topic := "data/powertrain/engine-speed"
	key := detailDashboardKey(topic)
	if len(key) > 40 {
		t.Fatalf("dashboard UID length = %d, want <= 40", len(key))
	}
	if key != detailDashboardKey(topic) {
		t.Fatal("detail dashboard key is not deterministic")
	}
	if key == detailDashboardKey("data/powertrain/oil-pressure") {
		t.Fatal("different topics must not share a detail dashboard key")
	}

	wantURL := "/d/" + key + "?from=${__from}&to=${__to}"
	if got := detailDashboardURL(topic); got != wantURL {
		t.Errorf("detail dashboard URL = %q, want %q", got, wantURL)
	}
}
