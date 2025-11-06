package main

import (
	"fmt"
	"strings"

	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/stat"
)

var (
	dataSourceRef = dashboard.DataSourceRef{
		Uid:  cog.ToPtr("mqtt-datasource"),
		Type: cog.ToPtr("grafana-mqtt-datasource"),
	}
)

func createDashboardsWithMQTTTopics(topics []string) ([]dashboard.Dashboard, error) {
	topicsMap := make(map[string]map[string][]string)
	for _, topic := range topics {
		if !strings.HasPrefix(topic, "data/") {
			return nil, fmt.Errorf("each topic must start with 'data/': %s", topic)
		}

		strippedTopic := strings.TrimPrefix(topic, "data/")
		topicParts := strings.Split(strippedTopic, "/")

		_, ok := topicsMap[topicParts[0]]
		if !ok {
			topicsMap[topicParts[0]] = make(map[string][]string)
		}

		topicsMap[topicParts[0]][topicParts[1]] = append(topicsMap[topicParts[0]][topicParts[1]], topic)
		fmt.Println(topicsMap[topicParts[0]][topicParts[1]])
	}

	fmt.Println(topicsMap)

	dashboards := make([]dashboard.Dashboard, 0)
	for firstLevel := range topicsMap {
		dashboardBuilder := dashboard.NewDashboardBuilder(firstLevel).
			Uid(fmt.Sprintf("generated-%s", firstLevel)).
			Refresh("1m").
			Time("now-1m", "now")

		for secondLevel := range topicsMap[firstLevel] {
			row := dashboard.NewRowBuilder(secondLevel).Datasource(dataSourceRef)

			for _, topic := range topicsMap[firstLevel][secondLevel] {
				row = row.WithPanel(
					stat.NewPanelBuilder().
						Title(topic).
						WithTarget(NewMQTTQueryBuilder(topic)),
				)
			}

			dashboardBuilder = dashboardBuilder.WithRow(row)
		}

		dashboard, err := dashboardBuilder.Build()
		if err != nil {
			return nil, err
		}
		dashboards = append(dashboards, dashboard)
	}

	return dashboards, nil
}
