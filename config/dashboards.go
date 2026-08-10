package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ApexCorse/vera"
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/stat"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"
)

const (
	topicPrefix = "data/"
	providers   = `apiVersion: 1

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

// parseSignalTopicHierarchy groups DBC signal topics by their first two topic levels.
// Topics are expected to follow the pattern: data/firstLevel/secondLevel.
func parseSignalTopicHierarchy(signalTopics []vera.SignalTopic) (map[string]map[string][]vera.SignalTopic, error) {
	topicsMap := make(map[string]map[string][]vera.SignalTopic)

	for _, signalTopic := range signalTopics {
		topic := signalTopic.Topic
		if !strings.HasPrefix(topic, topicPrefix) {
			return nil, fmt.Errorf("each topic must start with '%s': %s", topicPrefix, topic)
		}

		strippedTopic := strings.TrimPrefix(topic, topicPrefix)
		topicParts := strings.Split(strippedTopic, "/")

		if len(topicParts) < 2 {
			return nil, fmt.Errorf("topic must have at least 2 levels after '%s': %s", topicPrefix, topic)
		}

		firstLevel := topicParts[0]
		secondLevel := topicParts[1]

		if firstLevel == "" || secondLevel == "" {
			return nil, fmt.Errorf("topic levels cannot be empty: %s", topic)
		}

		if _, exists := topicsMap[firstLevel]; !exists {
			topicsMap[firstLevel] = make(map[string][]vera.SignalTopic)
		}

		topicsMap[firstLevel][secondLevel] = append(topicsMap[firstLevel][secondLevel], signalTopic)
	}

	return topicsMap, nil
}

// buildDashboardForLevel creates live MQTT and historical InfluxDB panels for every signal.
func buildDashboardForLevel(firstLevel string, secondLevels map[string][]vera.SignalTopic) (dashboard.Dashboard, error) {
	dashboardBuilder := dashboard.NewDashboardBuilder(firstLevel).
		Uid(fmt.Sprintf("generated-%s", firstLevel)).
		Refresh("1m").
		Time("now-1m", "now")

	for secondLevel, signalTopics := range secondLevels {
		row := dashboard.NewRowBuilder(secondLevel)

		for _, signalTopic := range signalTopics {
			row = row.WithPanel(
				stat.NewPanelBuilder().
					Title(signalTopic.Topic + " (live)").
					GraphMode(common.BigValueGraphModeNone).
					Datasource(dataSourceRef).
					WithTarget(NewMQTTQueryBuilder(signalTopic.Topic)),
			).WithPanel(
				timeseries.NewPanelBuilder().
					Title(signalTopic.Topic + " (history)").
					Datasource(influxDBDataSourceRef).
					WithTarget(NewInfluxDBQueryBuilder(signalTopic.Topic)),
			)
		}

		dashboardBuilder = dashboardBuilder.WithRow(row)
	}

	return dashboardBuilder.Build()
}

// createDashboardsWithSignalTopics generates Grafana dashboards from DBC signal-topic mappings.
func createDashboardsWithSignalTopics(signalTopics []vera.SignalTopic) (map[string]dashboard.Dashboard, error) {
	topicsMap, err := parseSignalTopicHierarchy(signalTopics)
	if err != nil {
		return nil, err
	}

	dashboards := make(map[string]dashboard.Dashboard)
	for firstLevel, secondLevels := range topicsMap {
		dashboard, err := buildDashboardForLevel(firstLevel, secondLevels)
		if err != nil {
			return nil, fmt.Errorf("failed to build dashboard for '%s': %w", firstLevel, err)
		}
		dashboards[firstLevel] = dashboard
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
