package db

import "time"

type Record struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Value     float32   `json:"value"`

	SensorID uint `json:"sensor_id"`
}

type Sensor struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Name      string    `gorm:"uniqueIndex:sensor_name_module" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Topic     string

	Records  []Record
	ModuleID uint `gorm:"uniqueIndex:sensor_name_module"`
}

type Module struct {
	ID   uint   `gorm:"primarykey" json:"id"`
	Name string `gorm:"uniqueIndex:module_name_section" json:"name"`

	Sensors   []Sensor
	SectionID uint `gorm:"uniqueIndex:module_name_section"`
}

type Section struct {
	ID   uint   `gorm:"primarykey" json:"id"`
	Name string `gorm:"uniqueIndex" json:"name"`

	Modules []Module
}

type User struct {
	Token     string    `gorm:"primarykey" json:"token"`
	CreatedAt time.Time `json:"created_at"`
	Username  string    `gorm:"index" json:"username"`
}
