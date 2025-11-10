package main

import (
	"encoding/json"
	"math/rand"
	"time"
)

// generateRandomData creates a random sensor data payload
func generateRandomData() ([]byte, error) {
	timestamp := time.Now()
	value := rand.Float32()*1000 - 500

	jsonPayload := struct {
		Value     float32   `json:"value"`
		Timestamp time.Time `json:"timestamp"`
		Unit      string    `json:"unit"`
	}{
		Value:     value,
		Timestamp: timestamp,
		Unit:      "V",
	}

	data, err := json.Marshal(jsonPayload)
	if err != nil {
		return nil, err
	}

	return data, nil
}
