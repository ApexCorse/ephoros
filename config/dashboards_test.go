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
		`r[\"topic\"] == \"data/powertrain/engine-speed\"`,
	} {
		if !strings.Contains(json, expected) {
			t.Errorf("generated dashboard does not contain %q", expected)
		}
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
