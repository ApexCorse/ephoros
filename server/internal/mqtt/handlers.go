package mqtt

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/ApexCorse/ephoros/server/internal/db"
	"github.com/eclipse/paho.golang/paho"
)

type MQTTHandler struct {
	db *db.DB
}

func NewMQTTHandler(db *db.DB) *MQTTHandler {
	return &MQTTHandler{db: db}
}

func (h *MQTTHandler) HandleAddRecordToDB(pr paho.PublishReceived) (bool, error) {
	if !strings.HasPrefix(pr.Packet.Topic, "raw/") {
		return false, nil
	}

	cleanTopic := strings.TrimPrefix(pr.Packet.Topic, "raw/")

	data := pr.Packet.Payload
	if len(data) != 12 {
		return false, fmt.Errorf("[HandleAddRecordToDB] invalid payload length: %v", data)
	}
	unsigned := binary.BigEndian.Uint32(data[8:])
	value := int32(unsigned)

	timestampUint64 := binary.BigEndian.Uint64(data[:8])
	timestamp := int64(timestampUint64)
	actualTime := time.Unix(timestamp, 0)

	sensor, err := h.db.GetSensorByTopic(cleanTopic)
	if err != nil {
		return false, fmt.Errorf("[HandleAddRecordToDB] sensor not found for topic '%s': %s", cleanTopic, err.Error())
	}

	err = h.db.InsertRecord(&db.Record{
		SensorID: sensor.ID,
		//TODO: Fix to integer
		Value:     float32(value),
		CreatedAt: actualTime,
	})
	if err != nil {
		return false, fmt.Errorf("[HandleAddRecordToDB] couldn't create record: %s", err.Error())
	}

	return true, nil
}
