package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validDBC = `VERSION "test"
NS_ :
BS_:
BU_: ECU
BO_ 256 Powertrain: 8 ECU
 SG_ EngineSpeed : 0|16@1+ (1,0) [0|100] "rpm" ECU
CM_ SG_ 256 EngineSpeed "vera:mqtt-topic=data/powertrain/engine-speed";
`

func TestGetDbcConfig(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*testing.T) string
		wantError  string
		wantTopics int
	}{
		{name: "missing environment variable", setup: func(t *testing.T) string { return "" }, wantError: "DBC_FILE_PATH env var is not set"},
		{
			name: "parses configured file",
			setup: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "vehicle.dbc")
				require.NoError(t, os.WriteFile(path, []byte(validDBC), 0o600))
				return path
			},
			wantTopics: 1,
		},
		{
			name: "falls back from config dbc to example",
			setup: func(t *testing.T) string {
				root := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(root, "config.example.dbc"), []byte(validDBC), 0o600))
				return filepath.Join(root, "config.dbc")
			},
			wantTopics: 1,
		},
		{name: "reports missing custom file", setup: func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing.dbc") }, wantError: "error while opening DBC file"},
		{
			name: "reports malformed file",
			setup: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "bad.dbc")
				require.NoError(t, os.WriteFile(path, []byte("BO_ definitely-not-valid"), 0o600))
				return path
			},
			wantError: "error in parsing DBC file",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("DBC_FILE_PATH", test.setup(t))
			config, err := getDbcConfig()
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				assert.Nil(t, config)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, config)
			assert.Len(t, config.Topics, test.wantTopics)
			assert.Equal(t, "data/powertrain/engine-speed", config.Topics[0].Topic)
		})
	}
}

func TestCleanProvisioningFolder(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, string)
	}{
		{name: "creates absent directory", prepare: func(*testing.T, string) {}},
		{
			name: "removes existing contents",
			prepare: func(t *testing.T, path string) {
				require.NoError(t, os.MkdirAll(filepath.Join(path, "nested"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(path, "nested", "stale.json"), []byte("stale"), 0o600))
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "dashboards")
			test.prepare(t, path)
			require.NoError(t, cleanProvisioningFolder(path))
			entries, err := os.ReadDir(path)
			require.NoError(t, err)
			assert.Empty(t, entries)
		})
	}
}

func TestCreateProviderFile(t *testing.T) {
	tests := []struct {
		name      string
		setupPath func(*testing.T) string
		wantError bool
	}{
		{
			name: "writes provider configuration",
			setupPath: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "dashboards")
				require.NoError(t, os.MkdirAll(path, 0o755))
				return path + string(os.PathSeparator)
			},
		},
		{name: "reports missing directory", setupPath: func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing") + string(os.PathSeparator) }, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := test.setupPath(t)
			err := createProviderFile(path)
			if test.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			contents, err := os.ReadFile(filepath.Join(path, "providers.yaml"))
			require.NoError(t, err)
			assert.Equal(t, providers, string(contents))
		})
	}
}
