package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ApexCorse/vera"
)

func TestBuildDashboardForLevelPairsLiveAndHistoricalPanels(t *testing.T) {
	dashboard, err := buildDashboardForLevel("powertrain", map[string][]vera.SignalTopic{
		"engine": {{Signal: "EngineSpeed", Topic: "data/powertrain/engine-speed"}},
	})
	if err != nil {
		t.Fatalf("build dashboard: %v", err)
	}

	encoded, err := json.Marshal(dashboard)
	if err != nil {
		t.Fatalf("marshal dashboard: %v", err)
	}
	json := string(encoded)

	for _, expected := range []string{
		"data/powertrain/engine-speed",
		"grafana-mqtt-datasource",
		`"graphMode":"none"`,
		"influxdb-datasource",
		`r[\"name\"] == \"EngineSpeed\"`,
	} {
		if !strings.Contains(json, expected) {
			t.Errorf("generated dashboard does not contain %q", expected)
		}
	}
}

func TestInfluxDBQueryUsesSignalName(t *testing.T) {
	query, err := NewInfluxDBQueryBuilder("BatteryVoltage").Build()
	if err != nil {
		t.Fatalf("build query: %v", err)
	}

	influxQuery, ok := query.(InfluxDBQuery)
	if !ok {
		t.Fatalf("query type = %T, want InfluxDBQuery", query)
	}
	if !strings.Contains(influxQuery.Query, `r["name"] == "BatteryVoltage"`) {
		t.Errorf("query does not filter by the signal name: %s", influxQuery.Query)
	}
}
