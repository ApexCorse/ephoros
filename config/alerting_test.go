package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBuildAlertProvisioningCreatesInfluxThresholdBands(t *testing.T) {
	warningLow := 600.0
	warningHigh := 6_500.0
	criticalLow := 300.0
	criticalHigh := 7_000.0
	staleSeconds := 30

	provisioning, err := buildAlertProvisioning([]AlertSignal{{
		Topic:             "data/powertrain/engine-speed",
		WarningLow:        &warningLow,
		WarningHigh:       &warningHigh,
		CriticalLow:       &criticalLow,
		CriticalHigh:      &criticalHigh,
		StaleAfterSeconds: &staleSeconds,
		DashboardUID:      "generated-telemetry",
		PanelID:           12,
	}})
	if err != nil {
		t.Fatalf("build alert provisioning: %v", err)
	}

	if provisioning.APIVersion != 1 {
		t.Fatalf("apiVersion = %d, want 1", provisioning.APIVersion)
	}
	if len(provisioning.Groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(provisioning.Groups))
	}
	group := provisioning.Groups[0]
	if group.Interval != "10s" {
		t.Errorf("evaluation interval = %q, want 10s", group.Interval)
	}
	if len(group.Rules) != 3 {
		t.Fatalf("rules = %d, want warning, critical, and stale", len(group.Rules))
	}

	warning := group.Rules[0]
	if warning.Labels["severity"] != "warning" || warning.Labels["topic"] != "data/powertrain/engine-speed" {
		t.Errorf("warning labels = %#v", warning.Labels)
	}
	if warning.DashboardUID != "generated-telemetry" || warning.PanelID != 12 {
		t.Errorf("warning dashboard link = %q/%d", warning.DashboardUID, warning.PanelID)
	}
	if warning.NoDataState != "OK" {
		t.Errorf("warning no-data state = %q, want OK when a stale rule exists", warning.NoDataState)
	}
	assertAlertData(t, warning, 30, "(($B <= 600 && $B > 300) || ($B >= 6500 && $B < 7000))")

	critical := group.Rules[1]
	if critical.Labels["severity"] != "critical" {
		t.Errorf("critical severity = %q", critical.Labels["severity"])
	}
	assertAlertData(t, critical, 30, "($B <= 300 || $B >= 7000)")

	stale := group.Rules[2]
	if stale.Labels["severity"] != "warning" {
		t.Errorf("stale severity = %q", stale.Labels["severity"])
	}
	if stale.NoDataState != "Alerting" {
		t.Errorf("stale no-data state = %q, want Alerting", stale.NoDataState)
	}
	assertAlertData(t, stale, 30, "is_number($B) == 0")
}

func TestBuildAlertProvisioningUsesNoDataStateWithoutStalePolicy(t *testing.T) {
	warningHigh := 80.0
	provisioning, err := buildAlertProvisioning([]AlertSignal{{
		Topic:       "data/electrical/battery-temperature",
		WarningHigh: &warningHigh,
	}})
	if err != nil {
		t.Fatalf("build alert provisioning: %v", err)
	}

	rule := provisioning.Groups[0].Rules[0]
	if rule.NoDataState != "NoData" {
		t.Errorf("no-data state = %q, want NoData", rule.NoDataState)
	}
	assertAlertData(t, rule, alertDefaultLookbackSecs, "$B >= 80")
}

func TestBuildAlertProvisioningIsDeterministic(t *testing.T) {
	low := 1.0
	high := 2.0
	first := []AlertSignal{
		{Topic: "data/zeta/value", CriticalHigh: &high},
		{Topic: "data/alpha/value", WarningLow: &low},
	}
	second := []AlertSignal{first[1], first[0]}

	left, err := buildAlertProvisioning(first)
	if err != nil {
		t.Fatalf("build first provisioning: %v", err)
	}
	right, err := buildAlertProvisioning(second)
	if err != nil {
		t.Fatalf("build second provisioning: %v", err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("provisioning differs when input order changes\nleft: %#v\nright: %#v", left, right)
	}

	rules := left.Groups[0].Rules
	if rules[0].Labels["topic"] != "data/alpha/value" || rules[1].Labels["topic"] != "data/zeta/value" {
		t.Errorf("rules are not sorted by topic: %#v", rules)
	}
	for _, rule := range rules {
		if len(rule.UID) > 40 {
			t.Errorf("rule UID %q exceeds Grafana's 40-character limit", rule.UID)
		}
	}
}

func TestWriteAlertProvisioningWritesValidJSON(t *testing.T) {
	criticalHigh := 100.0
	path := filepath.Join(t.TempDir(), "nested", "alerts.json")
	if err := WriteAlertProvisioning(path, []AlertSignal{{
		Topic:        "data/electrical/voltage",
		CriticalHigh: &criticalHigh,
	}}); err != nil {
		t.Fatalf("write alert provisioning: %v", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read alert provisioning: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(contents, &decoded); err != nil {
		t.Fatalf("generated provisioning is not JSON: %v", err)
	}
	if !strings.HasSuffix(string(contents), "\n") {
		t.Error("generated provisioning should end with a newline")
	}
	if strings.Contains(string(contents), "mqtt-datasource") {
		t.Error("generated alert provisioning must not query the MQTT datasource")
	}
	if !strings.Contains(string(contents), `"datasourceUid": "influxdb-datasource"`) {
		t.Error("generated alert provisioning does not query InfluxDB")
	}
}

func TestBuildAlertProvisioningRejectsInvalidSignals(t *testing.T) {
	zero := 0
	nan := math.NaN()
	criticalLow := 300.0
	warningLow := 200.0
	high := 100.0

	tests := []struct {
		name    string
		signals []AlertSignal
		want    string
	}{
		{name: "empty topic", signals: []AlertSignal{{}}, want: "topic cannot be empty"},
		{
			name: "duplicate topic",
			signals: []AlertSignal{
				{Topic: "data/a/value", WarningHigh: &high},
				{Topic: "data/a/value", CriticalHigh: &high},
			},
			want: "duplicate alert topic",
		},
		{
			name:    "partial dashboard link",
			signals: []AlertSignal{{Topic: "data/a/value", DashboardUID: "dashboard"}},
			want:    "must be provided together",
		},
		{
			name:    "non-positive stale interval",
			signals: []AlertSignal{{Topic: "data/a/value", StaleAfterSeconds: &zero}},
			want:    "must be positive",
		},
		{
			name:    "non-finite threshold",
			signals: []AlertSignal{{Topic: "data/a/value", CriticalHigh: &nan}},
			want:    "must be finite",
		},
		{
			name: "misordered thresholds",
			signals: []AlertSignal{{
				Topic:       "data/a/value",
				CriticalLow: &criticalLow,
				WarningLow:  &warningLow,
			}},
			want: "critical low threshold must be less than warning low threshold",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := buildAlertProvisioning(test.signals)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func assertAlertData(t *testing.T, rule alertRule, wantLookback int, wantCondition string) {
	t.Helper()
	if len(rule.Data) != 3 {
		t.Fatalf("data stages = %d, want 3", len(rule.Data))
	}

	query := rule.Data[0]
	if query.DatasourceUID != alertDatasourceUID {
		t.Errorf("query datasource = %q, want %q", query.DatasourceUID, alertDatasourceUID)
	}
	if query.RelativeTimeRange.From != wantLookback || query.RelativeTimeRange.To != 0 {
		t.Errorf("query time range = %#v, want from=%d to=0", query.RelativeTimeRange, wantLookback)
	}
	model, ok := query.Model.(influxAlertModel)
	if !ok {
		t.Fatalf("query model type = %T, want influxAlertModel", query.Model)
	}
	for _, expected := range []string{
		`from(bucket: "telemetry")`,
		`r["_measurement"] == "can_signal"`,
		`r["_field"] == "value"`,
		`r["topic"] == "data/`,
		`|> last()`,
	} {
		if !strings.Contains(model.Query, expected) {
			t.Errorf("Influx query does not contain %q:\n%s", expected, model.Query)
		}
	}

	reduce, ok := rule.Data[1].Model.(expressionAlertModel)
	if !ok || reduce.Type != "reduce" || reduce.Reducer != "last" || reduce.Expression != "A" {
		t.Errorf("reduce model = %#v", rule.Data[1].Model)
	}
	condition, ok := rule.Data[2].Model.(expressionAlertModel)
	if !ok || condition.Type != "math" || condition.Expression != wantCondition {
		t.Errorf("condition model = %#v, want expression %q", rule.Data[2].Model, wantCondition)
	}
	if rule.Condition != "C" {
		t.Errorf("rule condition = %q, want C", rule.Condition)
	}
}
