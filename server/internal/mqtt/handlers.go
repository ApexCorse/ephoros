package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
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
	if !strings.HasPrefix(pr.Packet.Topic, "clean/") {
		return false, nil
	}
	log.Printf("[HandleAddRecordToDB] data incoming from topic: %s\n", pr.Packet.Topic)

	cleanTopic := strings.TrimPrefix(pr.Packet.Topic, "clean/")

	data := struct {
		Value     float32   `json:"value"`
		Timestamp time.Time `json:"timestamp"`
	}{}

	err := json.Unmarshal(pr.Packet.Payload, &data)
	if err != nil {
		return false, fmt.Errorf("[HandleAddRecordToDB] couldn't parse JSON data: %s", err.Error())
	}

	sensor, err := h.db.GetSensorByTopic(cleanTopic)
	if err != nil {
		return false, fmt.Errorf("[HandleAddRecordToDB] sensor not found for topic '%s': %s", cleanTopic, err.Error())
	}

	err = h.db.InsertRecord(&db.Record{
		SensorID: sensor.ID,
		//TODO: Fix to integer
		Value:     float32(data.Value),
		CreatedAt: data.Timestamp,
	})
	if err != nil {
		return false, fmt.Errorf("[HandleAddRecordToDB] couldn't create record: %s", err.Error())
	}

	return true, nil
}
