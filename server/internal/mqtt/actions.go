package mqtt

import (
	"context"
	"errors"

	"github.com/eclipse/paho.golang/paho"
)

func (c *MQTTClient) Publish(ctx context.Context, topic string, payload []byte) error {
	if topic == "" {
		return errors.New("topic cannot be empty")
	}

	if len(payload) == 0 {
		return errors.New("payload cannot be empty")
	}

	p := &paho.Publish{
		Topic:   topic,
		Payload: payload,
	}

	if _, err := c.c.Publish(ctx, p); err != nil {
		return err
	}

	return nil
}
