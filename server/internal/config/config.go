package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
)

type SensorConfig struct {
	Name    string `json:"name"`
	ID      uint   `json:"id"`
	Section string `json:"section"`
	Module  string `json:"module"`
	Type    uint   `json:"type"`
}

func (c *SensorConfig) Validate() bool {
	log.Printf("[CONFIG] Validating sensor config - Name: %s, ID: %d, Section: %s, Module: %s, Type: %d",
		c.Name, c.ID, c.Section, c.Module, c.Type)

	isValid := c.Name != "" && c.Section != "" && c.Module != ""

	if !isValid {
		log.Printf("[CONFIG] Sensor config validation failed - Name: %s, Section: %s, Module: %s",
			c.Name, c.Section, c.Module)
	} else {
		log.Printf("[CONFIG] Sensor config validation successful - Name: %s", c.Name)
	}

	return isValid
}

type Config struct {
	SensorConfigs []SensorConfig   `json:"sensors"`
}

func NewConfig(configs []SensorConfig) *Config {
	log.Printf("[CONFIG] Creating new configuration - Sensors: %d\n",
		len(configs))

	config := &Config{SensorConfigs: configs}

	log.Println("[CONFIG] Configuration created successfully")
	return config
}

func NewConfigFromReader(reader io.Reader) (*Config, error) {
	log.Println("[CONFIG] Loading configuration from reader")

	config := &Config{}

	err := json.NewDecoder(reader).Decode(config)
	if err != nil {
		log.Printf("[CONFIG] Error decoding JSON configuration: %v", err)
		return nil, err
	}

	log.Printf("[CONFIG] JSON decoded successfully - Sensors: %d\n",
		len(config.SensorConfigs))

	log.Println("[CONFIG] Validating sensor configurations")
	for i, sConfig := range config.SensorConfigs {
		if !sConfig.Validate() {
			log.Printf("[CONFIG] Sensor config validation failed at index %d", i+1)
			return nil, fmt.Errorf("config nº%d not valid", i+1)
		}
	}

	log.Println("[CONFIG] Validating MQTT configurations")

	log.Println("[CONFIG] All configurations validated successfully")
	return config, nil
}
