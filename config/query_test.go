package main

import (
	"strconv"
	"testing"

	"github.com/grafana/grafana-foundation-sdk/go/cog/variants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPointer(value bool) *bool { return &value }

func TestInfluxDBQueryBuilder(t *testing.T) {
	tests := []struct {
		name       string
		bucket     string
		topic      string
		refID      string
		hide       bool
		wantBucket string
	}{
		{name: "defaults to telemetry bucket", topic: "data/electrical/battery-voltage", refID: "A", hide: true, wantBucket: "telemetry"},
		{name: "uses configured bucket", bucket: "vehicle-metrics", topic: `data/interior/driver\"temperature`, refID: "B", hide: false, wantBucket: "vehicle-metrics"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("INFLUXDB_INIT_BUCKET", test.bucket)
			query, err := NewInfluxDBQueryBuilder(test.topic).RefId(test.refID).Hide(test.hide).Build()
			require.NoError(t, err)
			got, ok := query.(InfluxDBQuery)
			require.True(t, ok)
			assert.Equal(t, test.refID, got.RefId)
			assert.Equal(t, boolPointer(test.hide), got.Hide)
			assert.True(t, got.RawQuery)
			assert.Equal(t, "time_series", got.ResultFormat)
			assert.Contains(t, got.Query, `from(bucket: "`+test.wantBucket+`")`)
			assert.Contains(t, got.Query, `r["topic"] == `)
			assert.Contains(t, got.Query, strconv.Quote(test.topic))
			assert.Equal(t, "influxdb", got.DataqueryType())
			require.NoError(t, got.Validate())
		})
	}
}

func TestInfluxDBQueryEquals(t *testing.T) {
	base := InfluxDBQuery{RefId: "A", Query: "query", RawQuery: true, ResultFormat: "time_series"}
	tests := []struct {
		name  string
		other variants.Dataquery
		want  bool
	}{
		{name: "equal", other: base, want: true},
		{name: "different ref ID", other: InfluxDBQuery{RefId: "B", Query: "query", RawQuery: true, ResultFormat: "time_series"}},
		{name: "different query", other: InfluxDBQuery{RefId: "A", Query: "other", RawQuery: true, ResultFormat: "time_series"}},
		{name: "different variant", other: MQTTQuery{Topic: "query"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { assert.Equal(t, test.want, base.Equals(test.other)) })
	}
}

func TestInfluxDBQueryVariantConfig(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantError bool
		want      *InfluxDBQuery
	}{
		{name: "unmarshals query", raw: `{"refId":"A","query":"from()","rawQuery":true,"resultFormat":"time_series"}`, want: &InfluxDBQuery{RefId: "A", Query: "from()", RawQuery: true, ResultFormat: "time_series"}},
		{name: "rejects invalid JSON", raw: `{`, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := InfluxDBQueryVariantConfig()
			assert.Equal(t, "influxdb", config.Identifier)
			got, err := config.DataqueryUnmarshaler([]byte(test.raw))
			if test.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestMQTTQueryBuilder(t *testing.T) {
	tests := []struct {
		name, topic, refID string
		hide               bool
	}{
		{name: "visible query", topic: "data/powertrain/engine-speed", refID: "A", hide: false},
		{name: "hidden query", topic: "data/electrical/voltage", refID: "B", hide: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query, err := NewMQTTQueryBuilder(test.topic).RefId(test.refID).Hide(test.hide).Build()
			require.NoError(t, err)
			got, ok := query.(MQTTQuery)
			require.True(t, ok)
			assert.Equal(t, MQTTQuery{RefId: test.refID, Hide: boolPointer(test.hide), Topic: test.topic}, got)
			assert.Equal(t, "custom", got.DataqueryType())
			require.NoError(t, got.Validate())
		})
	}
}

func TestMQTTQueryEquals(t *testing.T) {
	base := MQTTQuery{RefId: "A", Hide: boolPointer(false), Topic: "data/a/value"}
	tests := []struct {
		name  string
		other variants.Dataquery
		want  bool
	}{
		{name: "equal", other: MQTTQuery{RefId: "A", Hide: boolPointer(false), Topic: "data/a/value"}, want: true},
		{name: "nil", other: nil},
		{name: "different variant", other: InfluxDBQuery{}},
		{name: "different ref ID", other: MQTTQuery{RefId: "B", Hide: boolPointer(false), Topic: "data/a/value"}},
		{name: "nil hide differs", other: MQTTQuery{RefId: "A", Topic: "data/a/value"}},
		{name: "hide differs", other: MQTTQuery{RefId: "A", Hide: boolPointer(true), Topic: "data/a/value"}},
		{name: "topic differs", other: MQTTQuery{RefId: "A", Hide: boolPointer(false), Topic: "data/b/value"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { assert.Equal(t, test.want, base.Equals(test.other)) })
	}
}

func TestMQTTQueryVariantConfig(t *testing.T) {
	tests := []struct {
		name, raw string
		wantError bool
		want      *MQTTQuery
	}{
		{name: "unmarshals query", raw: `{"refId":"A","hide":true,"topic":"data/a/value"}`, want: &MQTTQuery{RefId: "A", Hide: boolPointer(true), Topic: "data/a/value"}},
		{name: "rejects invalid JSON", raw: `{`, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := MQTTQueryVariantConfig()
			assert.Equal(t, "mqtt-datasource", config.Identifier)
			got, err := config.DataqueryUnmarshaler([]byte(test.raw))
			if test.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}
