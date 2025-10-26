package db

import "time"

type Metric struct {
	Time  time.Time      `gorm:"primaryKey" json:"time"`
	Topic string         `gorm:"primaryKey" json:"topic"`
	Value float32        `json:"value"`
	Unit  string         `json:"unit"`
	Tags  map[string]any `json:"tags"`
}

const (
	CREATE_METRIC_TABLE = `CREATE TABLE metrics (
	"time" timestamptz not null,
	topic text not null,
	value real,
	unit text,
	tags jsonb
)
WITH(
	timescaledb.hypertable,
	timescaledb.partition_column='time',
	timescaledb.chunk_interval='1 day'
)
`
)
