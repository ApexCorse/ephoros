package mqtt

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ApexCorse/ephoros/server/internal/db"
	"github.com/eclipse/paho.golang/paho"
)

func HandleProcessRawData(ctx context.Context, pr paho.PublishReceived) (bool, error) {
	if !strings.HasPrefix(pr.Packet.Topic, "raw/") {
		return false, nil
	}
	log.Printf("[HandleProcessRawData] data incoming from topic: %s\n", pr.Packet.Topic)

	cleanTopic := strings.TrimPrefix(pr.Packet.Topic, "raw/")

	data := pr.Packet.Payload
	if len(data) != 12 {
		return false, fmt.Errorf("[HandleProcessRawData] invalid payload length: %v", data)
	}
	unsigned := binary.BigEndian.Uint32(data[8:])
	value := int32(unsigned)

	timestampUint64 := binary.BigEndian.Uint64(data[:8])
	timestamp := int64(timestampUint64)

	actualTime := time.UnixMilli(timestamp)

	jsonData := struct {
		Value     float32   `json:"value"`
		Timestamp time.Time `json:"timestamp"`
	}{
		Value:     float32(value),
		Timestamp: actualTime,
	}

	newPayload, err := json.Marshal(jsonData)
	if err != nil {
		return false, fmt.Errorf("[HandleProcessRawData] couldn't parse data: %s", err.Error())
	}

	ctx, stop := context.WithTimeout(ctx, 10*time.Second)
	defer stop()

	_, err = pr.Client.Publish(ctx, &paho.Publish{
		Topic:   fmt.Sprintf("p/%s", cleanTopic),
		Payload: newPayload,
	})
	if err != nil {
		return false, fmt.Errorf("[HandleProcessRawData] couldn't publish processed data: %s", err.Error())
	}

	return true, nil
}

func HandleAddRecordToDB(DB *db.DB, pr paho.PublishReceived) (bool, error) {
	if !strings.HasPrefix(pr.Packet.Topic, "p/") {
		return false, nil
	}
	log.Printf("[HandleAddRecordToDB] data incoming from topic: %s\n", pr.Packet.Topic)

	cleanTopic := strings.TrimPrefix(pr.Packet.Topic, "p/")

	data := struct {
		Value     float32   `json:"value"`
		Timestamp time.Time `json:"timestamp"`
	}{}

	err := json.Unmarshal(pr.Packet.Payload, &data)
	if err != nil {
		return false, fmt.Errorf("[HandleAddRecordToDB] couldn't parse JSON data: %s", err.Error())
	}

	sensor, err := DB.GetSensorByTopic(cleanTopic)
	if err != nil {
		return false, fmt.Errorf("[HandleAddRecordToDB] sensor not found for topic '%s': %s", cleanTopic, err.Error())
	}

	err = DB.InsertRecord(&db.Record{
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
