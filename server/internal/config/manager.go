package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/ApexCorse/ephoros/server/internal/db"
)

type ConfigManager struct {
	config *Config
	db     *db.DB
}

func NewConfigManager(config *Config, db *db.DB) *ConfigManager {
	log.Println("[CONFIG] Creating new configuration manager")

	if config == nil {
		log.Println("[CONFIG] Warning: config is nil")
	}

	if db == nil {
		log.Println("[CONFIG] Warning: database is nil")
	}

	manager := &ConfigManager{
		config: config,
		db:     db,
	}

	if config != nil {
		log.Printf("[CONFIG] Configuration manager created - Sensors: %d",
			len(config.SensorConfigs))
	} else {
		log.Printf("[CONFIG] Configuration manager created - No configuration provided")
	}

	return manager
}

func (m *ConfigManager) UpdateConfigurationFile(w io.Writer) error {
	bytes, err := json.MarshalIndent(m.config, "", "  ")

	if err != nil {
		return err
	}

	_, err = w.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}

func (m *ConfigManager) UpdateDB() error {
	log.Println("[CONFIG] Starting database update from configuration")

	if m.config == nil {
		log.Println("[CONFIG] Error: No configuration available for database update")
		return fmt.Errorf("no configuration available")
	}

	for i := range m.config.SensorConfigs {
		// take a pointer to the actual slice element so modifications persist
		sConfig := &m.config.SensorConfigs[i]
		log.Printf("[CONFIG] Processing sensor config %d/%d - Name: %s, Section: %s, Module: %s",
			i+1, len(m.config.SensorConfigs), sConfig.Name, sConfig.Section, sConfig.Module)

		err := m.createSensorIfNotExists(sConfig)
		if err != nil {
			log.Printf("[CONFIG] Error creating sensor - Name: %s, Section: %s, Module: %s, Error: %v",
				sConfig.Name, sConfig.Section, sConfig.Module, err)
			return err
		}

		log.Printf("[CONFIG] Successfully processed sensor config - Name: %s", sConfig.Name)
	}

	log.Println("[CONFIG] Database update completed successfully")
	return nil
}

func (m *ConfigManager) createSectionIfNotExists(sectionName string) (*db.Section, error) {
	log.Printf("[CONFIG] Checking if section exists: %s", sectionName)

	section, err := m.db.GetSectionByName(sectionName)

	if err != nil {
		log.Printf("[CONFIG] Section not found, creating new section: %s", sectionName)

		section = &db.Section{
			Name: sectionName,
		}

		err = m.db.InsertSection(section)
		if err != nil && !strings.Contains(err.Error(), "unique constraint") {
			log.Printf("[CONFIG] Error creating section - Name: %s, Error: %v", sectionName, err)
			return nil, err
		}

		log.Printf("[CONFIG] Section created successfully - ID: %d, Name: %s", section.ID, section.Name)
	} else {
		log.Printf("[CONFIG] Section already exists - ID: %d, Name: %s", section.ID, section.Name)
	}

	return section, nil
}

func (m *ConfigManager) createModuleIfNotExists(sectionName, moduleName string) (*db.Module, error) {
	log.Printf("[CONFIG] Creating module if not exists - Section: %s, Module: %s", sectionName, moduleName)

	section, err := m.createSectionIfNotExists(sectionName)
	if err != nil {
		log.Printf("[CONFIG] Error creating section for module - Section: %s, Module: %s, Error: %v",
			sectionName, moduleName, err)
		return nil, err
	}

	module, err := m.db.GetModuleByNameAndSection(moduleName, sectionName)

	if err != nil {
		log.Printf("[CONFIG] Module not found, creating new module - Section: %s, Module: %s", sectionName, moduleName)

		module = &db.Module{
			Name:      moduleName,
			SectionID: section.ID,
		}

		err = m.db.InsertModule(module)
		if err != nil && !strings.Contains(err.Error(), "unique constraint") {
			log.Printf("[CONFIG] Error creating module - Section: %s, Module: %s, Error: %v",
				sectionName, moduleName, err)
			return nil, err
		}

		log.Printf("[CONFIG] Module created successfully - ID: %d, Name: %s, SectionID: %d",
			module.ID, module.Name, module.SectionID)
	} else {
		log.Printf("[CONFIG] Module already exists - ID: %d, Name: %s, SectionID: %d",
			module.ID, module.Name, module.SectionID)
	}

	return module, nil
}

func (m *ConfigManager) createSensorIfNotExists(sensorConfig *SensorConfig) error {
	log.Printf("[CONFIG] Creating sensor if not exists - Section: %s, Module: %s, Sensor: %s",
		sensorConfig.Section, sensorConfig.Module, sensorConfig.Name)

	module, err := m.createModuleIfNotExists(sensorConfig.Section, sensorConfig.Module)
	if err != nil {
		log.Printf("[CONFIG] Error creating module for sensor - Section: %s, Module: %s, Sensor: %s, Error: %v",
			sensorConfig.Section, sensorConfig.Module, sensorConfig.Name, err)
		return err
	}

	sensor, err := m.db.GetSensorByNameAndModuleAndSection(sensorConfig.Name, sensorConfig.Module, sensorConfig.Section, time.Now(), time.Now())

	if err != nil {
		log.Printf("[CONFIG] Sensor not found, creating new sensor - Section: %s, Module: %s, Sensor: %s",
			sensorConfig.Section, sensorConfig.Module, sensorConfig.Name)

		sensor = &db.Sensor{
			Name:     sensorConfig.Name,
			ModuleID: module.ID,
			Topic: strings.Join(
				[]string{
					sensorConfig.Section,
					sensorConfig.Module,
					sensorConfig.Name,
				},
				"/",
			),
		}

		err = m.db.InsertSensor(sensor)
		if err != nil && !strings.Contains(err.Error(), "unique constraint") {
			log.Printf("[CONFIG] Error creating sensor - Section: %s, Module: %s, Sensor: %s, Error: %v",
				sensorConfig.Section, sensorConfig.Module, sensorConfig.Name, err)
			return err
		}

		log.Printf("[CONFIG] Sensor created successfully - ID: %d, Name: %s, ModuleID: %d",
			sensor.ID, sensor.Name, sensor.ModuleID)
	} else {
		log.Printf("[CONFIG] Sensor already exists - Section: %s, Module: %s, Sensor: %s",
			sensorConfig.Section, sensorConfig.Module, sensorConfig.Name)
	}

	sensorConfig.ID = sensor.ID
	log.Printf("[CONFIG] Sensor config updated: ID = %d", sensorConfig.ID)

	return nil
}
