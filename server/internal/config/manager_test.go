package config

import (
	"testing"

	"github.com/ApexCorse/ephoros/server/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestUpdateDB(t *testing.T) {
	gormDb, cleanUp, err := db.TestDB()
	if err != nil {
		t.Fatal("cannot setup db")
	}
	defer cleanUp()

	a := assert.New(t)

	config := &Config{
		SensorConfigs: []SensorConfig{
			{
				Name:    "NTC-1",
				Module:  "Module 1",
				Section: "Battery",
			},
			{
				Name:    "NTC-2",
				Module:  "Module 2",
				Section: "Battery",
			},
			{
				Name:    "NTC-3",
				Module:  "Module 1",
				Section: "Vehicle",
			},
		},
	}

	configManager := NewConfigManager(config, db.NewDB(gormDb))
	err = configManager.UpdateDB()

	a.Nil(err)

	sections := make([]db.Section, 0)
	gormDb.Find(&sections).Order("name DESC")

	a.Len(sections, 2)
	a.Equal("Battery", sections[0].Name)
	a.Equal("Vehicle", sections[1].Name)

	modules := make([]db.Module, 0)
	gormDb.Find(&modules).Order("name DESC").Order("section_id DESC")

	a.Len(modules, 3)
	a.Equal("Module 1", modules[0].Name)
	a.Equal(sections[0].ID, modules[0].SectionID)
	a.Equal("Module 2", modules[1].Name)
	a.Equal(sections[0].ID, modules[1].SectionID)
	a.Equal("Module 1", modules[2].Name)
	a.Equal(sections[1].ID, modules[2].SectionID)

	sensors := make([]db.Sensor, 0)
	gormDb.Find(&sensors).Order("name DESC").Order("section_id DESC")

	a.Len(sensors, 3)
	a.Equal("NTC-1", sensors[0].Name)
	a.Equal(modules[0].ID, sensors[0].ModuleID)
	a.Equal("NTC-2", sensors[1].Name)
	a.Equal(modules[1].ID, sensors[1].ModuleID)
	a.Equal("NTC-3", sensors[2].Name)
	a.Equal(modules[2].ID, sensors[2].ModuleID)

	for _, s := range sensors {
		a.NotZero(s.ID)
	}
}
