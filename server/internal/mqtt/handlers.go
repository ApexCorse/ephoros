package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/ApexCorse/ephoros/server/internal/db"
	"github.com/eclipse/paho.golang/paho"
)

func HandleAddRecordToDB(DB *db.DB, pr paho.PublishReceived) (bool, error) {
	if !strings.HasPrefix(pr.Packet.Topic, "data/") {
		return true, nil
	}
	log.Printf("[HandleAddRecordToDB] data incoming from topic: %s\n", pr.Packet.Topic)
	topic := strings.TrimPrefix(pr.Packet.Topic, "data/")

	metric := &db.Metric{
		Topic: topic,
	}

	err := json.Unmarshal(pr.Packet.Payload, &metric)
	if err != nil {
		return false, fmt.Errorf("[HandleAddRecordToDB] couldn't parse JSON data: %s", err.Error())
	}

	err = DB.InsertMetric(metric)
	if err != nil {
		return false, fmt.Errorf("[HandleAddRecordToDB] couldn't create metric: %s", err.Error())
	}

	return true, nil
}
