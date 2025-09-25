package mqtt

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/ApexCorse/ephoros/server/internal/db"
	"github.com/eclipse/paho.golang/paho"
	"github.com/stretchr/testify/assert"
)

func TestHandleAddRecordToDB(t *testing.T) {
	t.Run("should create record in DB", func(t *testing.T) {
		gormDb, cleanUp, err := db.TestDB()
		if err != nil {
			t.Fatal("could not open db")
		}
		defer cleanUp()

		a := assert.New(t)
		DB := db.NewDB(gormDb)

		section := &db.Section{
			Name: "Battery",
		}
		gormDb.Create(section)

		module := &db.Module{
			Name:      "Module-1",
			SectionID: section.ID,
		}
		gormDb.Create(module)

		sensor := &db.Sensor{
			Name:     "NTC-1",
			ModuleID: module.ID,
			Topic:    "Battery/Module-1/NTC-1",
		}
		gormDb.Create(sensor)

		time := time.Now()
		buf := &bytes.Buffer{}
		err = binary.Write(buf, binary.BigEndian, time.UnixMilli())
		a.NoError(err)
		a.Equal(8, buf.Len())
		timeBytes := buf.Bytes()

		var value int32 = 42
		buf = &bytes.Buffer{}
		err = binary.Write(buf, binary.BigEndian, value)
		a.NoError(err)
		a.Equal(4, buf.Len())
		valueBytes := buf.Bytes()

		finalBytes := []byte{}
		finalBytes = append(finalBytes, timeBytes...)
		finalBytes = append(finalBytes, valueBytes...)

		pr := paho.PublishReceived{
			Packet: &paho.Publish{
				Topic:   "raw/Battery/Module-1/NTC-1",
				Payload: finalBytes,
			},
		}

		ok, err := HandleAddRecordToDB(DB, pr)
		a.NoError(err)
		a.True(ok)

		records := make([]db.Record, 0)
		tx := gormDb.Find(&records)
		a.NoError(tx.Error)
		a.Len(records, 1)
		a.Equal(sensor.ID, records[0].SensorID)
	})

	t.Run("topic without 'raw/' prefix, return false and no error", func(t *testing.T) {
		a := assert.New(t)

		pr := paho.PublishReceived{
			Packet: &paho.Publish{
				Topic: "Battery/Module-1/NTC-1",
			},
		}

		ok, err := HandleAddRecordToDB(nil, pr)
		a.NoError(err)
		a.False(ok)
	})
}
