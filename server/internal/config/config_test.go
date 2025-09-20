package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSensorConfigValidate(t *testing.T) {
	tests := []struct {
		config     *SensorConfig
		shouldPass bool
	}{
		{
			config: &SensorConfig{
				Name:    "NTC-1",
				ID:      1,
				Section: "Battery",
				Module:  "Module 1",
				Type:    0,
			},
			shouldPass: true,
		},
		{
			config: &SensorConfig{
				Name:    "",
				ID:      1,
				Section: "Battery",
				Module:  "Module 1",
				Type:    0,
			},
			shouldPass: false,
		},
		{
			config: &SensorConfig{
				Name:    "NTC-1",
				ID:      1,
				Section: "",
				Module:  "Module 1",
				Type:    0,
			},
			shouldPass: false,
		},
		{
			config: &SensorConfig{
				Name:    "NTC-1",
				ID:      1,
				Section: "Battery",
				Module:  "",
				Type:    0,
			},
			shouldPass: false,
		},
	}

	for _, test := range tests {
		res := test.config.Validate()

		assert.Equal(t, test.shouldPass, res)
	}
}

func TestNewConfigFromReader(t *testing.T) {
	tests := []struct {
		readerString string
		nConfigs     int
		returnsError bool
	}{
		{
			readerString: `
		{
			"sensors": [
				{
					"name": "NTC-1",
					"id": 1,
					"section": "Battery",
					"module": "Module 1",
					"type": 1
				},
				{
					"name": "NTC-2",
					"id": 2,
					"section": "Battery",
					"module": "Module 1",
					"type": 1
				},
				{
					"name": "NTC-3",
					"id": 3,
					"section": "Battery",
					"module": "Module 1",
					"type": 1
				}
			]
		}
			`,
			nConfigs:     3,
			returnsError: false,
		},
		{
			// error in curly braces
			readerString: `
		{
			"sensors": [
				{}
					"name": "NTC-1",
					"id": 1,
					"section": "Battery",
					"module": "Module 1",
					"type": 1
				},
				{
					"name": "NTC-2",
					"id": 2,
					"section": "Battery",
					"module": "Module 1",
					"type": 1
				},
				{
					"name": "NTC-3",
					"id": 3,
					"section": "Battery",
					"module": "Module 1",
					"type": 1
				}
			]
		}
			`,
			returnsError: true,
		}, {
			// missing name in third config
			readerString: `
		{
			"sensors": [
				{
					"name": "NTC-1",
					"id": 1,
					"section": "Battery",
					"module": "Module 1",
					"type": 1
				},
				{
					"name": "NTC-2",
					"id": 2,
					"section": "Battery",
					"module": "Module 1",
					"type": 1
				},
				{
					"name": "",
					"id": 3,
					"section": "Battery",
					"module": "Module 1",
					"type": 1
				}
			]
		}
			`,
			nConfigs:     3,
			returnsError: true,
		},
		{
			readerString: `
		{
			"sensors": [
				{
					"name": "NTC-1",
					"id": 1,
					"section": "Battery",
					"module": "Module 1",
					"type": 1
				}
			]
		}
			`,
			nConfigs:     1,
			returnsError: false,
		},
		{
			readerString: `
		{
			"sensors": [
				{
					"name": "Temperature-1",
					"id": 1,
					"section": "Engine",
					"module": "Thermal",
					"type": 2
				},
				{
					"name": "Pressure-1",
					"id": 2,
					"section": "Engine",
					"module": "Hydraulic",
					"type": 3
				},
				{
					"name": "Voltage-1",
					"id": 3,
					"section": "Electrical",
					"module": "Power",
					"type": 4
				}
			]
		}
			`,
			nConfigs:     3,
			returnsError: false,
		},
		{
			readerString: `
		{
			"sensors": []
		}
			`,
			nConfigs:     0,
			returnsError: false,
		},
	}

	for _, test := range tests {
		reader := strings.NewReader(test.readerString)
		config, err := NewConfigFromReader(reader)

		if !test.returnsError {
			assert.Nil(t, err)
			assert.Len(t, config.SensorConfigs, test.nConfigs)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestGetSensorIDFromConfig(t *testing.T) {
	t.Run("should return sensor id", func(t *testing.T) {
		a := assert.New(t)

		config := &Config{
			SensorConfigs: []SensorConfig{
				{
					Name:    "NTC-1",
					ID:      1,
					Section: "Battery",
					Module:  "Module 1",
					Type:    0,
				},
			},
		}

		id, err := config.GetSensorIdFromData("Battery", "Module 1", "NTC-1")
		a.Nil(err)
		a.Equal(uint(1), id)
	})

	t.Run("should return sensor id", func(t *testing.T) {
		a := assert.New(t)

		config := &Config{
			SensorConfigs: []SensorConfig{
				{
					Name:    "NTC-1",
					ID:      1,
					Section: "Battery",
					Module:  "Module 1",
					Type:    0,
				},
			},
		}

		id, err := config.GetSensorIdFromData("Vehicle", "Module 1", "NTC-1")
		a.Error(err)
		a.Zero(id)
	})
}
