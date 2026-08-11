package main

import (
	"testing"

	"github.com/ApexCorse/vera"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func float32PointerForMetadata(value float32) *float32 { return &value }
func uint32PointerForMetadata(value uint32) *uint32    { return &value }
func intPointerForMetadata(value int) *int             { return &value }

func TestSignalsFromMetadata(t *testing.T) {
	config := &vera.Config{Messages: []vera.Message{{
		Name: "Powertrain",
		Signals: []vera.Signal{
			{
				Name: "EngineSpeed",
				Metadata: vera.SignalMetadata{
					MQTTTopic:    "data/powertrain/engine-speed",
					WarningLow:   float32PointerForMetadata(600),
					WarningHigh:  float32PointerForMetadata(6500),
					CriticalLow:  float32PointerForMetadata(300),
					CriticalHigh: float32PointerForMetadata(7000),
					StaleAfterMs: uint32PointerForMetadata(250),
				},
			},
			{
				Name:     "AmbientLight",
				Metadata: vera.SignalMetadata{MQTTTopic: "data/interior/ambient-light"},
			},
			{Name: "Unpublished"},
		},
	}}}

	topics, alerts, err := signalsFromMetadata(config)
	require.NoError(t, err)
	assert.Equal(t, []SignalTopic{
		{Topic: "data/powertrain/engine-speed"},
		{Topic: "data/interior/ambient-light"},
	}, topics)
	require.Len(t, alerts, 1)
	assert.Equal(t, "data/powertrain/engine-speed", alerts[0].Topic)
	require.NotNil(t, alerts[0].WarningLow)
	assert.Equal(t, 600.0, *alerts[0].WarningLow)
	require.NotNil(t, alerts[0].WarningHigh)
	assert.Equal(t, 6500.0, *alerts[0].WarningHigh)
	require.NotNil(t, alerts[0].CriticalLow)
	assert.Equal(t, 300.0, *alerts[0].CriticalLow)
	require.NotNil(t, alerts[0].CriticalHigh)
	assert.Equal(t, 7000.0, *alerts[0].CriticalHigh)
	require.NotNil(t, alerts[0].StaleAfterSeconds)
	assert.Equal(t, 1, *alerts[0].StaleAfterSeconds)
	assert.Equal(t, detailDashboardKey("data/powertrain/engine-speed"), alerts[0].DashboardUID)
	assert.Equal(t, int(stablePanelID("data/powertrain/engine-speed", 'D')), alerts[0].PanelID)
}

func TestSignalsFromMetadataRejectsPolicyWithoutTopic(t *testing.T) {
	config := &vera.Config{Messages: []vera.Message{{
		Name: "Powertrain",
		Signals: []vera.Signal{{
			Name:     "EngineSpeed",
			Metadata: vera.SignalMetadata{StaleAfterMs: uint32PointerForMetadata(1000)},
		}},
	}}}

	_, _, err := signalsFromMetadata(config)
	require.EqualError(t, err, `signal "EngineSpeed" in message "Powertrain" defines alert metadata but has no MQTT topic`)
}

func TestStaleAfterSecondsRoundsUp(t *testing.T) {
	tests := []struct {
		milliseconds *uint32
		want         *int
	}{
		{nil, nil},
		{uint32PointerForMetadata(1), intPointerForMetadata(1)},
		{uint32PointerForMetadata(1000), intPointerForMetadata(1)},
		{uint32PointerForMetadata(1001), intPointerForMetadata(2)},
		{uint32PointerForMetadata(^uint32(0)), intPointerForMetadata(4294968)},
	}

	for _, test := range tests {
		got := staleAfterSeconds(test.milliseconds)
		assert.Equal(t, test.want, got)
	}
}
