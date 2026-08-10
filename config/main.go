package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ApexCorse/vera"
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/cog/plugins"
)

func main() {
	dashboardsPath := os.Getenv("DASHBOARDS_PATH")
	if dashboardsPath == "" {
		fmt.Println("missing env DASHBOARDS_PATH")
		os.Exit(1)
	}
	preconfigGrafana()

	if err := cleanProvisioningFolder(dashboardsPath); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err := createProviderFile(dashboardsPath); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	config, err := getDbcConfig()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	dashboards, err := createDashboardsWithSignalTopics(config.Topics)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	for key, dashboard := range dashboards {
		file, err := os.Create(dashboardsPath + key + ".json")
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		json, err := json.MarshalIndent(dashboard, "", "  ")
		if err != nil {
			fmt.Println(err.Error())
			file.Close()
			os.Exit(1)
		}

		_, err = file.Write(json)
		if err != nil {
			fmt.Println(err.Error())
			file.Close()
			os.Exit(1)
		}
		if err := file.Close(); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}
}

func preconfigGrafana() {
	// Required to correctly unmarshal panels and dataqueries
	plugins.RegisterDefaultPlugins()

	// This lets cog know about the newly created query type and how to unmarshal it.
	cog.NewRuntime().RegisterDataqueryVariant(MQTTQueryVariantConfig())
	cog.NewRuntime().RegisterDataqueryVariant(InfluxDBQueryVariantConfig())
}

func getDbcConfig() (*vera.Config, error) {
	dbcFilePath := os.Getenv("DBC_FILE_PATH")
	if dbcFilePath == "" {
		return nil, fmt.Errorf("DBC_FILE_PATH env var is not set")
	}

	dbcFile, err := os.Open(dbcFilePath)
	if err != nil {
		return nil, fmt.Errorf("error while opening DBC file: %w\n", err)
	}

	config, err := vera.Parse(dbcFile)
	if err != nil {
		return nil, fmt.Errorf("error in parsing DBC file: %w\n", err)
	}

	return config, nil
}

func cleanProvisioningFolder(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return err
	}

	return os.MkdirAll(path, os.ModePerm)
}
