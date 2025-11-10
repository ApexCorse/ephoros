package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

// MQTTClient wraps the autopaho connection manager
type MQTTClient struct {
	c *autopaho.ConnectionManager
}

// Publish sends a message to the specified topic
func (c *MQTTClient) Publish(ctx context.Context, topic string, payload []byte) error {
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}
	if len(payload) == 0 {
		return fmt.Errorf("payload cannot be empty")
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

// newMQTTClient creates a new MQTT client from the autopaho connection manager
func newMQTTClient(c *autopaho.ConnectionManager) *MQTTClient {
	return &MQTTClient{c: c}
}

// MQTTClientBuilder helps build MQTT clients
type MQTTClientBuilder struct {
	cfg *autopaho.ClientConfig
}

// newMQTTClientBuilder creates a new builder
func newMQTTClientBuilder(cfg *autopaho.ClientConfig) *MQTTClientBuilder {
	builder := &MQTTClientBuilder{}
	if cfg != nil {
		builder.cfg = cfg
	} else {
		builder.cfg = &autopaho.ClientConfig{}
	}
	return builder
}

func (b *MQTTClientBuilder) addServers(urls []*url.URL) *MQTTClientBuilder {
	b.cfg.ServerUrls = urls
	return b
}

func (b *MQTTClientBuilder) addKeepAlive(value uint16) *MQTTClientBuilder {
	b.cfg.KeepAlive = value
	return b
}

func (b *MQTTClientBuilder) addCleanStartOnInitialConnection(value bool) *MQTTClientBuilder {
	b.cfg.CleanStartOnInitialConnection = value
	return b
}

func (b *MQTTClientBuilder) addOnConnectionUp(f func(cm *autopaho.ConnectionManager, connAck *paho.Connack)) *MQTTClientBuilder {
	b.cfg.OnConnectionUp = f
	return b
}

func (b *MQTTClientBuilder) addOnConnectionError(f func(err error)) *MQTTClientBuilder {
	b.cfg.OnConnectError = f
	return b
}

func (b *MQTTClientBuilder) addClientId(id string) *MQTTClientBuilder {
	b.cfg.ClientConfig.ClientID = id
	return b
}

func (b *MQTTClientBuilder) build(ctx context.Context) (*MQTTClient, error) {
	cm, err := autopaho.NewConnection(ctx, *b.cfg)
	if err != nil {
		return nil, err
	}

	err = cm.AwaitConnection(ctx)
	if err != nil {
		return nil, err
	}

	return newMQTTClient(cm), nil
}
