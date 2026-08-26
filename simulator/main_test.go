package main

import (
	"encoding/json"
	"math"
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

func TestWriteTelemetryCatalog(t *testing.T) {
	tests := []struct {
		name    string
		signals []simulatorSignal
		path    func(string) string
		want    string
		wantErr string
	}{
		{
			name: "writes signal metadata",
			signals: []simulatorSignal{
				{Topic: "vehicle/speed", Name: "Speed", Unit: "km/h", Min: 0, Max: 100},
				{Topic: `vehicle/quoted"topic`, Name: "Temperature", Unit: "degC", Min: -40, Max: 215},
				{Topic: `vehicle/back\\slash`, Name: "Voltage", Unit: "V", Min: 0.125, Max: 65.535},
			},
			path: func(directory string) string { return filepath.Join(directory, "signals.h") },
			want: "// Code generated from config.dbc; DO NOT EDIT.\n#pragma once\n\n" +
				"typedef struct {\n" +
				"\tconst char *topic;\n\tconst char *name;\n\tconst char *unit;\n" +
				"\tfloat min;\n\tfloat max;\n" +
				"} ephoros_telemetry_simulator_signal_t;\n\n" +
				"#define EPHOROS_TELEMETRY_SIMULATOR_SIGNAL_COUNT 3\n\n" +
				"static const ephoros_telemetry_simulator_signal_t ephoros_telemetry_simulator_signals[] = {\n" +
				"\t{.topic = \"vehicle/speed\", .name = \"Speed\", .unit = \"km/h\", .min = 0.0f, .max = 100.0f},\n" +
				"\t{.topic = \"vehicle/quoted\\\"topic\", .name = \"Temperature\", .unit = \"degC\", .min = -40.0f, .max = 215.0f},\n" +
				"\t{.topic = \"vehicle/back\\\\\\\\slash\", .name = \"Voltage\", .unit = \"V\", .min = 0.125f, .max = 65.535f},\n" +
				"};\n",
		},
		{
			name:    "rejects empty catalog",
			path:    func(directory string) string { return filepath.Join(directory, "signals.h") },
			wantErr: "DBC config contains no MQTT topics",
		},
		{
			name: "rejects reversed bounds",
			signals: []simulatorSignal{
				{Topic: "vehicle/speed", Name: "Speed", Min: 100, Max: 0},
			},
			path:    func(directory string) string { return filepath.Join(directory, "signals.h") },
			wantErr: `DBC signal "Speed" has minimum 100 greater than maximum 0`,
		},
		{
			name: "rejects non-finite bounds",
			signals: []simulatorSignal{
				{Topic: "vehicle/speed", Name: "Speed", Min: 0, Max: float32(math.Inf(1))},
			},
			path:    func(directory string) string { return filepath.Join(directory, "signals.h") },
			wantErr: `DBC signal "Speed" has non-finite bounds`,
		},
		{
			name:    "create error",
			signals: []simulatorSignal{{Topic: "vehicle/speed", Name: "Speed", Min: 0, Max: 100}},
			path:    func(directory string) string { return directory },
			wantErr: "is a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()
			outputPath := tt.path(directory)

			err := writeTelemetryCatalog(outputPath, tt.signals)

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

func TestGetSimulatorSignalsFromConfig(t *testing.T) {
	config := &vera.Config{Messages: []vera.Message{
		{Signals: []vera.Signal{
			{
				Name: "Speed", Unit: "km/h", Min: 0, Max: 100,
				Metadata: vera.SignalMetadata{MQTTTopic: "vehicle/speed"},
			},
			{Name: "NoTopic", Unit: "V", Min: -1, Max: 1},
		}},
		{Signals: []vera.Signal{
			{
				Name: "SpeedBackup", Unit: "mph", Min: 5, Max: 80,
				Metadata: vera.SignalMetadata{MQTTTopic: "vehicle/speed"},
			},
		}},
	}}

	assert.Equal(t, []simulatorSignal{
		{Topic: "vehicle/speed", Name: "Speed", Unit: "km/h", Min: 0, Max: 100},
		{Topic: "vehicle/speed", Name: "SpeedBackup", Unit: "mph", Min: 5, Max: 80},
	}, getSimulatorSignalsFromConfig(config))
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
