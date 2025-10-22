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

func HandleAddRecordToDB(DB *db.DB, pr paho.PublishReceived) (bool, error) {
	if !strings.HasPrefix(pr.Packet.Topic, "data/") {
		return true, nil
	}
	log.Printf("[HandleAddRecordToDB] data incoming from topic: %s\n", pr.Packet.Topic)
	topic := strings.TrimPrefix(pr.Packet.Topic, "data/")

	data := struct {
		Value     float32   `json:"value"`
		Timestamp time.Time `json:"timestamp"`
	}{}

	err := json.Unmarshal(pr.Packet.Payload, &data)
	if err != nil {
		return false, fmt.Errorf("[HandleAddRecordToDB] couldn't parse JSON data: %s", err.Error())
	}

	sensor, err := DB.GetSensorByTopic(topic)
	if err != nil {
		return false, fmt.Errorf("[HandleAddRecordToDB] sensor not found for topic '%s': %s", topic, err.Error())
	}

	err = DB.InsertRecord(&db.Record{
		SensorID:  sensor.ID,
		Value:     data.Value,
		CreatedAt: data.Timestamp,
	})
	if err != nil {
		return false, fmt.Errorf("[HandleAddRecordToDB] couldn't create record: %s", err.Error())
	}

	return true, nil
}
