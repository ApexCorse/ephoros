package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ApexCorse/vera"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validDBC = `BO_ 123 Vehicle: 8 ECU
	SG_ Speed : 0|16@1+ (0.1,0) [0|100] "km/h" Gateway
BA_DEF_ SG_ "VeraMqttTopic" STRING ;
BA_ "VeraMqttTopic" SG_ 123 Speed "vehicle/speed";`

func TestGenerateRandomData(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "first payload"},
		{name: "second payload"},
		{name: "third payload"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			data, err := generateRandomData()
			after := time.Now()

			require.NoError(t, err)
			var payload struct {
				Value float32   `json:"value"`
				Time  time.Time `json:"time"`
				Unit  string    `json:"unit"`
			}
			require.NoError(t, json.Unmarshal(data, &payload))
			assert.GreaterOrEqual(t, payload.Value, float32(-500))
			assert.Less(t, payload.Value, float32(500))
			assert.Equal(t, "V", payload.Unit)
			assert.False(t, payload.Time.Before(before))
			assert.False(t, payload.Time.After(after))
		})
	}
}

func TestGetDbcConfig(t *testing.T) {
	tests := []struct {
		name       string
		requested  string
		files      map[string]string
		wantTopics []string
		wantErr    string
	}{
		{
			name:       "requested file",
			requested:  "vehicle.dbc",
			files:      map[string]string{"vehicle.dbc": validDBC},
			wantTopics: []string{"vehicle/speed"},
		},
		{
			name:       "example fallback for missing config dbc",
			requested:  "config.dbc",
			files:      map[string]string{"config.example.dbc": validDBC},
			wantTopics: []string{"vehicle/speed"},
		},
		{
			name:      "missing non-default file",
			requested: "vehicle.dbc",
			wantErr:   "error while opening DBC file",
		},
		{
			name:      "missing fallback file",
			requested: "config.dbc",
			wantErr:   "error while opening DBC file",
		},
		{
			name:      "parse error",
			requested: "invalid.dbc",
			files: map[string]string{
				"invalid.dbc": `CM_ SG_ 123 Speed "vera:mqtt-topic=vehicle/speed"`,
			},
			wantErr: "error in parsing DBC file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()
			for name, contents := range tt.files {
				require.NoError(t, os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600))
			}

			config, err := getDbcConfig(filepath.Join(directory, tt.requested))

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, config)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, config)
			assert.Equal(t, tt.wantTopics, getTopicsFromConfig(config))
		})
	}
}

func TestWriteTopicCatalog(t *testing.T) {
	tests := []struct {
		name    string
		topics  []string
		path    func(string) string
		want    string
		wantErr string
	}{
		{
			name:   "writes header",
			topics: []string{"vehicle/speed", `vehicle/quoted"topic`, `vehicle/back\\slash`},
			path:   func(directory string) string { return filepath.Join(directory, "topics.h") },
			want: "// Code generated from config.dbc; DO NOT EDIT.\n#pragma once\n\n" +
				"#define EPHOROS_TELEMETRY_SIMULATOR_TOPIC_COUNT 3\n\n" +
				"static const char * const ephoros_telemetry_simulator_topics[] = {\n" +
				"\t\"vehicle/speed\",\n\t\"vehicle/quoted\\\"topic\",\n\t\"vehicle/back\\\\\\\\slash\",\n};\n",
		},
		{
			name:    "rejects empty topics",
			path:    func(directory string) string { return filepath.Join(directory, "topics.h") },
			wantErr: "DBC config contains no MQTT topics",
		},
		{
			name:    "create error",
			topics:  []string{"vehicle/speed"},
			path:    func(directory string) string { return directory },
			wantErr: "is a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()
			outputPath := tt.path(directory)

			err := writeTopicCatalog(outputPath, tt.topics)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			contents, err := os.ReadFile(outputPath)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(contents))
		})
	}
}

func TestGetTopicsFromConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *vera.Config
		want   []string
	}{
		{name: "empty", config: &vera.Config{}, want: []string{}},
		{
			name: "preserves message and signal order, duplicates, and skips empty topics",
			config: &vera.Config{Messages: []vera.Message{
				{Signals: []vera.Signal{
					{Name: "Speed", Metadata: vera.SignalMetadata{MQTTTopic: "vehicle/speed"}},
					{Name: "NoTopic"},
					{Name: "Voltage", Metadata: vera.SignalMetadata{MQTTTopic: "vehicle/voltage"}},
				}},
				{Signals: []vera.Signal{
					{Name: "SpeedBackup", Metadata: vera.SignalMetadata{MQTTTopic: "vehicle/speed"}},
				}},
			}},
			want: []string{"vehicle/speed", "vehicle/voltage", "vehicle/speed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getTopicsFromConfig(tt.config))
		})
	}
}
