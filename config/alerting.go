package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	alertDatasourceUID       = "influxdb-datasource"
	alertDatasourceType      = "influxdb"
	alertExpressionUID       = "__expr__"
	alertEvaluationInterval  = "10s"
	alertFolder              = "Ephoros Telemetry"
	alertGroupName           = "Ephoros signal alerts"
	alertDefaultLookbackSecs = 300
)

// AlertSignal is the dashboard-independent input used to provision alerts for
// one telemetry topic. Nil threshold and staleness fields are not provisioned.
// DashboardUID and PanelID are optional, but must be provided together.
type AlertSignal struct {
	Topic string

	WarningLow   *float64
	WarningHigh  *float64
	CriticalLow  *float64
	CriticalHigh *float64

	StaleAfterSeconds *int
	DashboardUID      string
	PanelID           int
}

type alertProvisioning struct {
	APIVersion int          `json:"apiVersion"`
	Groups     []alertGroup `json:"groups"`
}

type alertGroup struct {
	OrgID    int         `json:"orgId"`
	Name     string      `json:"name"`
	Folder   string      `json:"folder"`
	Interval string      `json:"interval"`
	Rules    []alertRule `json:"rules"`
}

type alertRule struct {
	UID          string            `json:"uid"`
	Title        string            `json:"title"`
	Condition    string            `json:"condition"`
	Data         []alertQuery      `json:"data"`
	DashboardUID string            `json:"dashboardUid,omitempty"`
	PanelID      int               `json:"panelId,omitempty"`
	NoDataState  string            `json:"noDataState"`
	ExecErrState string            `json:"execErrState"`
	For          string            `json:"for"`
	Annotations  map[string]string `json:"annotations"`
	Labels       map[string]string `json:"labels"`
	IsPaused     bool              `json:"isPaused"`
}

type alertQuery struct {
	RefID             string                 `json:"refId"`
	QueryType         string                 `json:"queryType"`
	RelativeTimeRange alertRelativeTimeRange `json:"relativeTimeRange"`
	DatasourceUID     string                 `json:"datasourceUid"`
	Model             any                    `json:"model"`
}

type alertRelativeTimeRange struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type influxAlertModel struct {
	Datasource    alertDatasource `json:"datasource"`
	Query         string          `json:"query"`
	RawQuery      bool            `json:"rawQuery"`
	ResultFormat  string          `json:"resultFormat"`
	Hide          bool            `json:"hide"`
	IntervalMS    int             `json:"intervalMs"`
	MaxDataPoints int             `json:"maxDataPoints"`
	RefID         string          `json:"refId"`
}

type expressionAlertModel struct {
	Datasource    alertDatasource `json:"datasource"`
	Expression    string          `json:"expression"`
	Hide          bool            `json:"hide"`
	IntervalMS    int             `json:"intervalMs"`
	MaxDataPoints int             `json:"maxDataPoints"`
	Reducer       string          `json:"reducer,omitempty"`
	RefID         string          `json:"refId"`
	Type          string          `json:"type"`
}

type alertDatasource struct {
	Type string `json:"type"`
	UID  string `json:"uid"`
}

// WriteAlertProvisioning writes Grafana file-provisioning JSON to path. The
// caller should place that file under Grafana's provisioning/alerting folder.
func WriteAlertProvisioning(path string, signals []AlertSignal) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("alert provisioning path cannot be empty")
	}

	provisioning, err := buildAlertProvisioning(signals)
	if err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(provisioning, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal alert provisioning: %w", err)
	}
	encoded = append(encoded, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create alert provisioning directory: %w", err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		return fmt.Errorf("write alert provisioning: %w", err)
	}
	return nil
}

func buildAlertProvisioning(signals []AlertSignal) (alertProvisioning, error) {
	sortedSignals := append([]AlertSignal(nil), signals...)
	sort.Slice(sortedSignals, func(i, j int) bool {
		return sortedSignals[i].Topic < sortedSignals[j].Topic
	})

	rules := make([]alertRule, 0, len(sortedSignals)*3)
	seenTopics := make(map[string]struct{}, len(sortedSignals))
	for _, signal := range sortedSignals {
		if err := validateAlertSignal(signal); err != nil {
			return alertProvisioning{}, err
		}
		if _, exists := seenTopics[signal.Topic]; exists {
			return alertProvisioning{}, fmt.Errorf("duplicate alert topic %q", signal.Topic)
		}
		seenTopics[signal.Topic] = struct{}{}

		lookbackSeconds := alertDefaultLookbackSecs
		thresholdNoDataState := "NoData"
		if signal.StaleAfterSeconds != nil {
			lookbackSeconds = *signal.StaleAfterSeconds
			thresholdNoDataState = "OK"
		}

		criticalExpression := buildCriticalExpression(signal)
		if warningExpression := buildWarningExpression(signal); warningExpression != "" {
			rules = append(rules, newThresholdAlertRule(
				signal,
				"warning",
				warningExpression,
				lookbackSeconds,
				thresholdNoDataState,
			))
		}
		if criticalExpression != "" {
			rules = append(rules, newThresholdAlertRule(
				signal,
				"critical",
				criticalExpression,
				lookbackSeconds,
				thresholdNoDataState,
			))
		}
		if signal.StaleAfterSeconds != nil {
			rules = append(rules, newStaleAlertRule(signal))
		}
	}

	return alertProvisioning{
		APIVersion: 1,
		Groups: []alertGroup{{
			OrgID:    1,
			Name:     alertGroupName,
			Folder:   alertFolder,
			Interval: alertEvaluationInterval,
			Rules:    rules,
		}},
	}, nil
}

func validateAlertSignal(signal AlertSignal) error {
	if strings.TrimSpace(signal.Topic) == "" {
		return errors.New("alert topic cannot be empty")
	}
	if (signal.DashboardUID == "") != (signal.PanelID == 0) {
		return fmt.Errorf("dashboard UID and panel ID must be provided together for topic %q", signal.Topic)
	}
	if signal.PanelID < 0 {
		return fmt.Errorf("panel ID must be positive for topic %q", signal.Topic)
	}
	if signal.StaleAfterSeconds != nil && *signal.StaleAfterSeconds <= 0 {
		return fmt.Errorf("stale-after seconds must be positive for topic %q", signal.Topic)
	}

	orderedThresholds := []struct {
		name  string
		value *float64
	}{
		{"critical low", signal.CriticalLow},
		{"warning low", signal.WarningLow},
		{"warning high", signal.WarningHigh},
		{"critical high", signal.CriticalHigh},
	}
	var previous *struct {
		name  string
		value float64
	}
	for _, threshold := range orderedThresholds {
		if threshold.value == nil {
			continue
		}
		if math.IsNaN(*threshold.value) || math.IsInf(*threshold.value, 0) {
			return fmt.Errorf("%s threshold must be finite for topic %q", threshold.name, signal.Topic)
		}
		if previous != nil && previous.value > *threshold.value {
			return fmt.Errorf(
				"%s threshold must not exceed %s threshold for topic %q",
				previous.name,
				threshold.name,
				signal.Topic,
			)
		}
		previous = &struct {
			name  string
			value float64
		}{threshold.name, *threshold.value}
	}
	return nil
}

func newThresholdAlertRule(
	signal AlertSignal,
	severity string,
	expression string,
	lookbackSeconds int,
	noDataState string,
) alertRule {
	rule := newAlertRule(signal, severity, severity)
	rule.NoDataState = noDataState
	rule.Data = alertRuleData(signal.Topic, expression, lookbackSeconds)
	rule.Annotations = map[string]string{
		"summary":     fmt.Sprintf("%s telemetry is outside its %s operating band", signal.Topic, severity),
		"description": fmt.Sprintf("The latest stored value for %s breached its configured %s threshold.", signal.Topic, severity),
	}
	return rule
}

func newStaleAlertRule(signal AlertSignal) alertRule {
	rule := newAlertRule(signal, "stale", "warning")
	rule.NoDataState = "Alerting"
	rule.Data = alertRuleData(signal.Topic, "is_number($B) == 0", *signal.StaleAfterSeconds)
	rule.Annotations = map[string]string{
		"summary":     fmt.Sprintf("%s telemetry is stale", signal.Topic),
		"description": fmt.Sprintf("InfluxDB has no numeric sample for %s within the last %d seconds.", signal.Topic, *signal.StaleAfterSeconds),
	}
	return rule
}

func newAlertRule(signal AlertSignal, kind string, severity string) alertRule {
	return alertRule{
		UID:          alertRuleUID(signal.Topic, kind),
		Title:        fmt.Sprintf("%s %s", signal.Topic, kind),
		Condition:    "C",
		DashboardUID: signal.DashboardUID,
		PanelID:      signal.PanelID,
		NoDataState:  "NoData",
		ExecErrState: "Error",
		For:          "0s",
		Labels: map[string]string{
			"severity": severity,
			"topic":    signal.Topic,
		},
		IsPaused: false,
	}
}

func alertRuleData(topic string, condition string, lookbackSeconds int) []alertQuery {
	zeroRange := alertRelativeTimeRange{From: 0, To: 0}
	return []alertQuery{
		{
			RefID:             "A",
			QueryType:         "",
			RelativeTimeRange: alertRelativeTimeRange{From: lookbackSeconds, To: 0},
			DatasourceUID:     alertDatasourceUID,
			Model: influxAlertModel{
				Datasource:    alertDatasource{Type: alertDatasourceType, UID: alertDatasourceUID},
				Query:         alertInfluxQuery(topic),
				RawQuery:      true,
				ResultFormat:  "time_series",
				Hide:          false,
				IntervalMS:    1_000,
				MaxDataPoints: 43_200,
				RefID:         "A",
			},
		},
		{
			RefID:             "B",
			QueryType:         "",
			RelativeTimeRange: zeroRange,
			DatasourceUID:     alertExpressionUID,
			Model: expressionAlertModel{
				Datasource:    alertDatasource{Type: alertExpressionUID, UID: alertExpressionUID},
				Expression:    "A",
				Hide:          false,
				IntervalMS:    1_000,
				MaxDataPoints: 43_200,
				Reducer:       "last",
				RefID:         "B",
				Type:          "reduce",
			},
		},
		{
			RefID:             "C",
			QueryType:         "",
			RelativeTimeRange: zeroRange,
			DatasourceUID:     alertExpressionUID,
			Model: expressionAlertModel{
				Datasource:    alertDatasource{Type: alertExpressionUID, UID: alertExpressionUID},
				Expression:    condition,
				Hide:          false,
				IntervalMS:    1_000,
				MaxDataPoints: 43_200,
				RefID:         "C",
				Type:          "math",
			},
		},
	}
}

func alertInfluxQuery(topic string) string {
	return fmt.Sprintf(`from(bucket: %q)
  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)
  |> filter(fn: (r) => r["_measurement"] == "can_signal")
  |> filter(fn: (r) => r["_field"] == "value")
  |> filter(fn: (r) => r["topic"] == %q)
  |> last()`, influxDBBucket(), topic)
}

func buildCriticalExpression(signal AlertSignal) string {
	conditions := make([]string, 0, 2)
	if signal.CriticalLow != nil {
		conditions = append(conditions, "$B <= "+formatThreshold(*signal.CriticalLow))
	}
	if signal.CriticalHigh != nil {
		conditions = append(conditions, "$B >= "+formatThreshold(*signal.CriticalHigh))
	}
	return joinAlertConditions(conditions, " || ")
}

func buildWarningExpression(signal AlertSignal) string {
	conditions := make([]string, 0, 2)
	if signal.WarningLow != nil && (signal.CriticalLow == nil || *signal.WarningLow != *signal.CriticalLow) {
		low := "$B <= " + formatThreshold(*signal.WarningLow)
		if signal.CriticalLow != nil {
			low = "(" + low + " && $B > " + formatThreshold(*signal.CriticalLow) + ")"
		}
		conditions = append(conditions, low)
	}
	if signal.WarningHigh != nil && (signal.CriticalHigh == nil || *signal.WarningHigh != *signal.CriticalHigh) {
		high := "$B >= " + formatThreshold(*signal.WarningHigh)
		if signal.CriticalHigh != nil {
			high = "(" + high + " && $B < " + formatThreshold(*signal.CriticalHigh) + ")"
		}
		conditions = append(conditions, high)
	}
	return joinAlertConditions(conditions, " || ")
}

func joinAlertConditions(conditions []string, separator string) string {
	if len(conditions) == 0 {
		return ""
	}
	if len(conditions) == 1 {
		return conditions[0]
	}
	return "(" + strings.Join(conditions, separator) + ")"
}

func formatThreshold(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

func alertRuleUID(topic string, kind string) string {
	digest := sha256.Sum256([]byte(topic + "\x00" + kind))
	return "ephoros-" + kind + "-" + hex.EncodeToString(digest[:8])
}
