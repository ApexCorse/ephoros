package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/ApexCorse/vera"
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/stat"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"
)

const (
	topicPrefix       = "data/"
	telemetryUID      = "generated-telemetry"
	telemetryTitle    = "Vehicle Telemetry"
	detailTitleSuffix = " Telemetry Detail"
	providers         = `apiVersion: 1

providers:
  - name: 'MQTT dashboards'
    orgId: 1
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: false
    options:
      path: /var/lib/grafana/dashboards`
)

var (
	dataSourceRef = dashboard.DataSourceRef{
		Uid:  cog.ToPtr("mqtt-datasource"),
		Type: cog.ToPtr("grafana-mqtt-datasource"),
	}
	influxDBDataSourceRef = dashboard.DataSourceRef{
		Uid:  cog.ToPtr("influxdb-datasource"),
		Type: cog.ToPtr("influxdb"),
	}
)

type topicSection struct {
	name    string
	signals []topicSignal
}

type topicSignal struct {
	label string
	topic string
}

// alertListOptions mirrors Grafana's native alertlist panel options. It is
// local because the Foundation SDK does not currently generate this panel.
type alertListOptions struct {
	ViewMode                 string               `json:"viewMode"`
	GroupMode                string               `json:"groupMode"`
	MaxItems                 int                  `json:"maxItems"`
	SortOrder                int                  `json:"sortOrder"`
	DashboardAlerts          bool                 `json:"dashboardAlerts"`
	AlertName                string               `json:"alertName"`
	AlertInstanceLabelFilter string               `json:"alertInstanceLabelFilter"`
	ShowInactiveAlerts       bool                 `json:"showInactiveAlerts"`
	StateFilter              alertListStateFilter `json:"stateFilter"`
}

type alertListStateFilter struct {
	Firing     bool `json:"firing"`
	Pending    bool `json:"pending"`
	Recovering bool `json:"recovering"`
	NoData     bool `json:"noData"`
	Normal     bool `json:"normal"`
	Error      bool `json:"error"`
}

// parseSignalTopicHierarchy groups topics into dashboard sections. A valid topic
// has a section and signal path after the required data/ prefix.
func parseSignalTopicHierarchy(signalTopics []vera.SignalTopic) ([]topicSection, error) {
	sectionsByName := make(map[string][]topicSignal)
	seenTopics := make(map[string]struct{}, len(signalTopics))

	for _, signalTopic := range signalTopics {
		topic := signalTopic.Topic
		if !strings.HasPrefix(topic, topicPrefix) {
			return nil, fmt.Errorf("each topic must start with %q: %s", topicPrefix, topic)
		}

		parts := strings.Split(strings.TrimPrefix(topic, topicPrefix), "/")
		if len(parts) < 2 {
			return nil, fmt.Errorf("topic must have a section and signal after %q: %s", topicPrefix, topic)
		}
		for _, part := range parts {
			if part == "" {
				return nil, fmt.Errorf("topic levels cannot be empty: %s", topic)
			}
		}
		if _, exists := seenTopics[topic]; exists {
			return nil, fmt.Errorf("duplicate topic: %s", topic)
		}
		seenTopics[topic] = struct{}{}

		section := parts[0]
		sectionsByName[section] = append(sectionsByName[section], topicSignal{
			label: humanizeTopicPath(parts[1:]),
			topic: topic,
		})
	}

	sections := make([]topicSection, 0, len(sectionsByName))
	for name, signals := range sectionsByName {
		sort.Slice(signals, func(i, j int) bool {
			if signals[i].label == signals[j].label {
				return signals[i].topic < signals[j].topic
			}
			return signals[i].label < signals[j].label
		})
		sections = append(sections, topicSection{name: humanizeTopicSegment(name), signals: signals})
	}
	sort.Slice(sections, func(i, j int) bool { return sections[i].name < sections[j].name })

	return sections, nil
}

func humanizeTopicPath(parts []string) string {
	labels := make([]string, len(parts))
	for i, part := range parts {
		labels[i] = humanizeTopicSegment(part)
	}
	return strings.Join(labels, " / ")
}

func humanizeTopicSegment(segment string) string {
	words := strings.FieldsFunc(segment, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, word := range words {
		words[i] = capitalize(word)
	}
	return strings.Join(words, " ")
}

func capitalize(value string) string {
	for index, r := range value {
		return string(unicode.ToUpper(r)) + value[index+len(string(r)):]
	}
	return value
}

// stablePanelID keeps provisioned alert links valid across regenerations. The
// kind byte separates the live and historical panels for the same topic.
func stablePanelID(topic string, kind byte) uint32 {
	digest := sha256.Sum256(append([]byte{kind, 0}, []byte(topic)...))
	id := binary.BigEndian.Uint32(digest[:4]) & 0x7fffffff
	if id == 0 {
		return 1
	}
	return id
}

// detailDashboardKey is used for both the provisioned filename and dashboard
// UID. It intentionally derives from the complete topic rather than its label,
// so renaming a display label never invalidates existing drill-down URLs. The
// 80-bit digest keeps the UID within Grafana's 40-character limit.
func detailDashboardKey(topic string) string {
	digest := sha256.Sum256([]byte(topic))
	return "generated-signal-" + hex.EncodeToString(digest[:10])
}

func detailDashboardURL(topic string) string {
	return "/d/" + detailDashboardKey(topic) + "?from=${__from}&to=${__to}"
}

func detailDashboardLink(topic string) *dashboard.DashboardLinkBuilder {
	return dashboard.NewDashboardLinkBuilder("Open detail").
		Type(dashboard.DashboardLinkTypeLink).
		Url(detailDashboardURL(topic)).
		KeepTime(true)
}

func alertListPanel() *dashboard.PanelBuilder {
	return dashboard.NewPanelBuilder().
		Type("alertlist").
		Id(stablePanelID(telemetryUID, 'a')).
		Title("Active Alerts").
		Description("Firing, pending, no-data, and error states for telemetry alerts linked to this dashboard.").
		Span(24).
		Height(5).
		Options(alertListOptions{
			ViewMode:                 "list",
			GroupMode:                "default",
			MaxItems:                 20,
			SortOrder:                3, // Grafana's Importance sort order.
			DashboardAlerts:          true,
			AlertName:                "",
			AlertInstanceLabelFilter: "",
			ShowInactiveAlerts:       false,
			StateFilter: alertListStateFilter{
				Firing:     true,
				Pending:    true,
				Recovering: false,
				NoData:     true,
				Normal:     false,
				Error:      true,
			},
		})
}

// buildTelemetryDashboard creates one stable topic-derived dashboard. Panels
// follow each expanded row at dashboard level because Grafana only preserves
// panels nested inside collapsed rows.
func buildTelemetryDashboard(sections []topicSection) (dashboard.Dashboard, error) {
	builder := dashboard.NewDashboardBuilder(telemetryTitle).
		Uid(telemetryUID).
		Refresh("1s").
		LiveNow(true).
		Time("now-15m", "now").
		WithPanel(alertListPanel())

	for _, section := range sections {
		builder = builder.WithRow(dashboard.NewRowBuilder(section.name).Collapsed(false))
		for _, signal := range section.signals {
			builder = builder.WithPanel(
				stat.NewPanelBuilder().
					Id(stablePanelID(signal.topic, 'l')).
					Title(signal.label + " (live)").
					Span(12).
					GraphMode(common.BigValueGraphModeArea).
					NoValue("No data").
					Datasource(dataSourceRef).
					DataLinks([]cog.Builder[dashboard.DashboardLink]{detailDashboardLink(signal.topic)}).
					WithTarget(NewMQTTQueryBuilder(signal.topic)),
			).WithPanel(
				timeseries.NewPanelBuilder().
					Id(stablePanelID(signal.topic, 'h')).
					Title(signal.label + " (history)").
					Span(12).
					Datasource(influxDBDataSourceRef).
					DataLinks([]cog.Builder[dashboard.DashboardLink]{detailDashboardLink(signal.topic)}).
					WithTarget(NewInfluxDBQueryBuilder(signal.topic)),
			)
		}
	}

	return builder.Build()
}

// buildSignalDetailDashboard creates a dedicated, literal-topic dashboard.
// MQTT targets do not support dashboard template variable interpolation, so a
// dashboard per topic preserves the exact subscription and query filter.
func buildSignalDetailDashboard(signal topicSignal) (dashboard.Dashboard, error) {
	return dashboard.NewDashboardBuilder(signal.label+detailTitleSuffix).
		Uid(detailDashboardKey(signal.topic)).
		Description("MQTT topic: "+signal.topic).
		Refresh("1s").
		LiveNow(true).
		Time("now-24h", "now").
		WithPanel(
			stat.NewPanelBuilder().
				Id(stablePanelID(signal.topic, 'd')).
				Title(signal.label + " (live)").
				Description("Latest value from MQTT topic: " + signal.topic).
				Span(24).
				GraphMode(common.BigValueGraphModeArea).
				NoValue("No data").
				Datasource(dataSourceRef).
				WithTarget(NewMQTTQueryBuilder(signal.topic)),
		).
		WithPanel(
			timeseries.NewPanelBuilder().
				Id(stablePanelID(signal.topic, 'D')).
				Title(signal.label + " (history)").
				Description("24-hour InfluxDB history for MQTT topic: " + signal.topic).
				Span(24).
				Datasource(influxDBDataSourceRef).
				WithTarget(NewInfluxDBQueryBuilder(signal.topic)),
		).
		Build()
}

// createDashboardsWithSignalTopics generates a stable overview and one
// deterministic drill-down dashboard per DBC signal-topic mapping.
func createDashboardsWithSignalTopics(signalTopics []vera.SignalTopic) (map[string]dashboard.Dashboard, error) {
	sections, err := parseSignalTopicHierarchy(signalTopics)
	if err != nil {
		return nil, err
	}

	telemetryDashboard, err := buildTelemetryDashboard(sections)
	if err != nil {
		return nil, fmt.Errorf("build telemetry dashboard: %w", err)
	}

	dashboards := map[string]dashboard.Dashboard{"telemetry": telemetryDashboard}
	for _, section := range sections {
		for _, signal := range section.signals {
			detailDashboard, err := buildSignalDetailDashboard(signal)
			if err != nil {
				return nil, fmt.Errorf("build detail dashboard for %q: %w", signal.topic, err)
			}
			dashboards[detailDashboardKey(signal.topic)] = detailDashboard
		}
	}

	return dashboards, nil
}

func createProviderFile(path string) error {
	path = path + "providers.yaml"
	providersFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer providersFile.Close()

	_, err = providersFile.Write([]byte(providers))
	if err != nil {
		return err
	}

	return nil
}
