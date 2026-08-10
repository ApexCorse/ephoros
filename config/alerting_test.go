package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func float64Pointer(value float64) *float64 { return &value }
func intPointer(value int) *int             { return &value }

func TestBuildAlertProvisioning(t *testing.T) {
	tests := []struct {
		name               string
		signals            []AlertSignal
		wantRuleCount      int
		wantTopics         []string
		wantSeverities     []string
		wantNoDataStates   []string
		wantLookbacks      []int
		wantConditions     []string
		wantDashboardUID   string
		wantDashboardPanel int
	}{
		{
			name: "warning critical and stale bands",
			signals: []AlertSignal{{
				Topic: "data/powertrain/engine-speed", WarningLow: float64Pointer(600),
				WarningHigh: float64Pointer(6500), CriticalLow: float64Pointer(300),
				CriticalHigh: float64Pointer(7000), StaleAfterSeconds: intPointer(30),
				DashboardUID: "generated-telemetry", PanelID: 12,
			}},
			wantRuleCount: 3, wantTopics: []string{"data/powertrain/engine-speed", "data/powertrain/engine-speed", "data/powertrain/engine-speed"},
			wantSeverities: []string{"warning", "critical", "warning"}, wantNoDataStates: []string{"OK", "OK", "Alerting"},
			wantLookbacks: []int{30, 30, 30}, wantConditions: []string{"(($B <= 600 && $B > 300) || ($B >= 6500 && $B < 7000))", "($B <= 300 || $B >= 7000)", "is_number($B) == 0"},
			wantDashboardUID: "generated-telemetry", wantDashboardPanel: 12,
		},
		{
			name:          "threshold without stale policy uses default lookback",
			signals:       []AlertSignal{{Topic: "data/electrical/battery-temperature", WarningHigh: float64Pointer(80)}},
			wantRuleCount: 1, wantTopics: []string{"data/electrical/battery-temperature"}, wantSeverities: []string{"warning"},
			wantNoDataStates: []string{"NoData"}, wantLookbacks: []int{alertDefaultLookbackSecs}, wantConditions: []string{"$B >= 80"},
		},
		{
			name:          "signals without policies produce no rules",
			signals:       []AlertSignal{{Topic: "data/interior/ambient-light"}},
			wantRuleCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provisioning, err := buildAlertProvisioning(test.signals)
			require.NoError(t, err)
			require.Equal(t, 1, provisioning.APIVersion)
			require.Len(t, provisioning.Groups, 1)
			group := provisioning.Groups[0]
			assert.Equal(t, alertEvaluationInterval, group.Interval)
			assert.Equal(t, alertFolder, group.Folder)
			require.Len(t, group.Rules, test.wantRuleCount)

			for index, rule := range group.Rules {
				assert.Equal(t, test.wantTopics[index], rule.Labels["topic"])
				assert.Equal(t, test.wantSeverities[index], rule.Labels["severity"])
				assert.Equal(t, test.wantNoDataStates[index], rule.NoDataState)
				assert.LessOrEqual(t, len(rule.UID), 40)
				assertAlertData(t, rule, test.wantLookbacks[index], test.wantConditions[index])
			}
			if test.wantDashboardUID != "" {
				assert.Equal(t, test.wantDashboardUID, group.Rules[0].DashboardUID)
				assert.Equal(t, test.wantDashboardPanel, group.Rules[0].PanelID)
			}
		})
	}
}

func TestBuildAlertProvisioningIsDeterministic(t *testing.T) {
	tests := []struct {
		name   string
		first  []AlertSignal
		second []AlertSignal
	}{
		{
			name:   "input order does not affect output",
			first:  []AlertSignal{{Topic: "data/zeta/value", CriticalHigh: float64Pointer(2)}, {Topic: "data/alpha/value", WarningLow: float64Pointer(1)}},
			second: []AlertSignal{{Topic: "data/alpha/value", WarningLow: float64Pointer(1)}, {Topic: "data/zeta/value", CriticalHigh: float64Pointer(2)}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			left, err := buildAlertProvisioning(test.first)
			require.NoError(t, err)
			right, err := buildAlertProvisioning(test.second)
			require.NoError(t, err)
			assert.Equal(t, left, right)
			rules := left.Groups[0].Rules
			require.Len(t, rules, 2)
			assert.Equal(t, "data/alpha/value", rules[0].Labels["topic"])
			assert.Equal(t, "data/zeta/value", rules[1].Labels["topic"])
		})
	}
}

func TestWriteAlertProvisioning(t *testing.T) {
	tests := []struct {
		name        string
		path        func(string) string
		wantError   string
		wantContent []string
	}{
		{name: "rejects empty path", path: func(string) string { return " \t" }, wantError: "path cannot be empty"},
		{
			name:        "writes valid influx JSON in nested directory",
			path:        func(root string) string { return filepath.Join(root, "nested", "alerts.json") },
			wantContent: []string{"\"datasourceUid\": \"influxdb-datasource\"", "\"apiVersion\": 1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := test.path(t.TempDir())
			err := WriteAlertProvisioning(path, []AlertSignal{{Topic: "data/electrical/voltage", CriticalHigh: float64Pointer(100)}})
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			contents, err := os.ReadFile(path)
			require.NoError(t, err)
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(contents, &decoded))
			assert.NotEmpty(t, decoded)
			assert.Equal(t, byte('\n'), contents[len(contents)-1])
			assert.NotContains(t, string(contents), "mqtt-datasource")
			for _, expected := range test.wantContent {
				assert.Contains(t, string(contents), expected)
			}
		})
	}
}

func TestBuildAlertProvisioningRejectsInvalidSignals(t *testing.T) {
	tests := []struct {
		name    string
		signals []AlertSignal
		want    string
	}{
		{name: "empty topic", signals: []AlertSignal{{}}, want: "topic cannot be empty"},
		{name: "duplicate topic", signals: []AlertSignal{{Topic: "data/a/value", WarningHigh: float64Pointer(100)}, {Topic: "data/a/value", CriticalHigh: float64Pointer(100)}}, want: "duplicate alert topic"},
		{name: "UID without panel", signals: []AlertSignal{{Topic: "data/a/value", DashboardUID: "dashboard"}}, want: "must be provided together"},
		{name: "panel without UID", signals: []AlertSignal{{Topic: "data/a/value", PanelID: 1}}, want: "must be provided together"},
		{name: "negative panel ID", signals: []AlertSignal{{Topic: "data/a/value", DashboardUID: "dashboard", PanelID: -1}}, want: "panel ID must be positive"},
		{name: "zero stale interval", signals: []AlertSignal{{Topic: "data/a/value", StaleAfterSeconds: intPointer(0)}}, want: "must be positive"},
		{name: "negative stale interval", signals: []AlertSignal{{Topic: "data/a/value", StaleAfterSeconds: intPointer(-1)}}, want: "must be positive"},
		{name: "NaN threshold", signals: []AlertSignal{{Topic: "data/a/value", CriticalHigh: float64Pointer(math.NaN())}}, want: "must be finite"},
		{name: "infinite threshold", signals: []AlertSignal{{Topic: "data/a/value", WarningLow: float64Pointer(math.Inf(1))}}, want: "must be finite"},
		{name: "descending thresholds", signals: []AlertSignal{{Topic: "data/a/value", CriticalLow: float64Pointer(300), WarningLow: float64Pointer(200)}}, want: "critical low threshold must be less than warning low threshold"},
		{name: "equal thresholds", signals: []AlertSignal{{Topic: "data/a/value", WarningHigh: float64Pointer(100), CriticalHigh: float64Pointer(100)}}, want: "warning high threshold must be less than critical high threshold"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := buildAlertProvisioning(test.signals)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestAlertExpressions(t *testing.T) {
	tests := []struct {
		name         string
		signal       AlertSignal
		wantWarning  string
		wantCritical string
	}{
		{name: "none", signal: AlertSignal{}, wantWarning: "", wantCritical: ""},
		{name: "warning low only", signal: AlertSignal{WarningLow: float64Pointer(1.25)}, wantWarning: "$B <= 1.25"},
		{name: "critical high only", signal: AlertSignal{CriticalHigh: float64Pointer(2e6)}, wantCritical: "$B >= 2e+06"},
		{name: "both warning bounds", signal: AlertSignal{WarningLow: float64Pointer(-1), WarningHigh: float64Pointer(1)}, wantWarning: "($B <= -1 || $B >= 1)"},
		{name: "both critical bounds", signal: AlertSignal{CriticalLow: float64Pointer(-2), CriticalHigh: float64Pointer(2)}, wantCritical: "($B <= -2 || $B >= 2)"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.wantWarning, buildWarningExpression(test.signal))
			assert.Equal(t, test.wantCritical, buildCriticalExpression(test.signal))
		})
	}
}

func assertAlertData(t *testing.T, rule alertRule, wantLookback int, wantCondition string) {
	t.Helper()
	require.Len(t, rule.Data, 3)
	query := rule.Data[0]
	assert.Equal(t, alertDatasourceUID, query.DatasourceUID)
	assert.Equal(t, alertRelativeTimeRange{From: wantLookback}, query.RelativeTimeRange)
	model, ok := query.Model.(influxAlertModel)
	require.True(t, ok)
	for _, expected := range []string{`from(bucket: "telemetry")`, `r["_measurement"] == "can_signal"`, `r["_field"] == "value"`, `r["topic"] == "data/`, `|> last()`} {
		assert.Contains(t, model.Query, expected)
	}
	reduce, ok := rule.Data[1].Model.(expressionAlertModel)
	require.True(t, ok)
	assert.Equal(t, "reduce", reduce.Type)
	assert.Equal(t, "last", reduce.Reducer)
	assert.Equal(t, "A", reduce.Expression)
	condition, ok := rule.Data[2].Model.(expressionAlertModel)
	require.True(t, ok)
	assert.Equal(t, "math", condition.Type)
	assert.Equal(t, wantCondition, condition.Expression)
	assert.Equal(t, "C", rule.Condition)
}
