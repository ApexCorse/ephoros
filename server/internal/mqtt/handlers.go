package mqtt

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ApexCorse/ephoros/server/internal/config"
	"github.com/ApexCorse/ephoros/server/internal/db"
	"github.com/eclipse/paho.golang/paho"
)

type MQTTHandler struct {
	db  *db.DB
	cfg *config.Config
}

func NewMQTTHandler(db *db.DB, cfg *config.Config) *MQTTHandler {
	return &MQTTHandler{db: db, cfg: cfg}
}

func (h *MQTTHandler) HandleAddRecordToDB(pr paho.PublishReceived) (bool, error) {
	if !strings.HasPrefix(pr.Packet.Topic, "raw/") {
		return false, nil
	}

	sensorData, err := getSensorDataFromTopic(strings.TrimPrefix(pr.Packet.Topic, "raw/"))
	if err != nil {
		return false, errors.Join(
			errors.New("[HandleAddRecordToDB] error getting sensor data from topic"),
			err,
		)
	}

	data := pr.Packet.Payload
	if len(data) != 12 {
		return false, fmt.Errorf("[HandleAddRecordToDB] invalid payload length: %v", data)
	}
	unsigned := binary.BigEndian.Uint32(data[8:])
	value := int32(unsigned)

	timestampUint64 := binary.BigEndian.Uint64(data[:8])
	timestamp := int64(timestampUint64)
	actualTime := time.Unix(timestamp, 0)

	sensorId, err := h.cfg.GetSensorIdFromData(sensorData.section, sensorData.module, sensorData.sensor)
	if err != nil {
		return false, fmt.Errorf("[HandleAddRecordToDB] sensor not found in configuration: %s", err.Error())
	}

	err = h.db.InsertRecord(&db.Record{
		SensorID: sensorId,
		//TODO: Fix to integer
		Value:     float32(value),
		CreatedAt: actualTime,
	})
	if err != nil {
		return false, fmt.Errorf("[HandleAddRecordToDB] couldn't create record: %s", err.Error())
	}

	return true, nil
}

type sensorData struct {
	section, module, sensor string
}

func getSensorDataFromTopic(topic string) (*sensorData, error) {
	parts := strings.Split(topic, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid topic: %s", topic)
	}

	return &sensorData{
		section: parts[0],
		module:  parts[1],
		sensor:  parts[2],
	}, nil
}
