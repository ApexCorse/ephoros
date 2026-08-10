package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMQTTClientPublishValidation(t *testing.T) {
	tests := []struct {
		name    string
		topic   string
		payload []byte
		wantErr string
	}{
		{name: "empty topic", payload: []byte("payload"), wantErr: "topic cannot be empty"},
		{name: "nil payload", topic: "telemetry/speed", wantErr: "payload cannot be empty"},
		{name: "empty payload", topic: "telemetry/speed", payload: []byte{}, wantErr: "payload cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewMQTTClient(nil)

			err := client.Publish(context.Background(), tt.topic, tt.payload)

			assert.EqualError(t, err, tt.wantErr)
		})
	}
}
