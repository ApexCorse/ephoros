package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/stat"
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
)

// parseTopicHierarchy parses a list of MQTT topics into a hierarchical structure.
// Topics are expected to follow the pattern: data/firstLevel/secondLevel
func parseTopicHierarchy(topics []string) (map[string]map[string][]string, error) {
	topicsMap := make(map[string]map[string][]string)

	for _, topic := range topics {
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
			topicsMap[firstLevel] = make(map[string][]string)
		}

		topicsMap[firstLevel][secondLevel] = append(topicsMap[firstLevel][secondLevel], topic)
	}

	return topicsMap, nil
}

// buildDashboardForLevel creates a dashboard for a given first-level topic hierarchy.
func buildDashboardForLevel(firstLevel string, secondLevels map[string][]string) (dashboard.Dashboard, error) {
	dashboardBuilder := dashboard.NewDashboardBuilder(firstLevel).
		Uid(fmt.Sprintf("generated-%s", firstLevel)).
		Refresh("1m").
		Time("now-1m", "now")

	for secondLevel, topics := range secondLevels {
		row := dashboard.NewRowBuilder(secondLevel)

		for _, topic := range topics {
			row = row.WithPanel(
				stat.NewPanelBuilder().
					Title(topic).
					Datasource(dataSourceRef).
					WithTarget(NewMQTTQueryBuilder(topic)),
			)
		}

		dashboardBuilder = dashboardBuilder.WithRow(row)
	}

	return dashboardBuilder.Build()
}

// createDashboardsWithMQTTTopics generates Grafana dashboards from a list of MQTT topics.
func createDashboardsWithMQTTTopics(topics []string) (map[string]dashboard.Dashboard, error) {
	topicsMap, err := parseTopicHierarchy(topics)
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
