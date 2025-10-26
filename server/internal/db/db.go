package db

import (
	"gorm.io/gorm"
)

type DB struct {
	db *gorm.DB
}

func NewDB(db *gorm.DB) *DB {
	return &DB{db: db}
}

func (d *DB) ConfigureTimescale() error {
	tx := d.db.Exec(CREATE_METRIC_TABLE)

	return tx.Error
}

func (d *DB) InsertMetric(metric *Metric) error {
	tx := d.db.Create(metric)

	return tx.Error
}
