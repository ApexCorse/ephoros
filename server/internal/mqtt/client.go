package mqtt

import (
	"context"
	"net/url"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

type MQTTClient struct {
	c *autopaho.ConnectionManager
}

type MQTTClientBuilder struct {
	cfg *autopaho.ClientConfig
}

func NewMQTTClient(c *autopaho.ConnectionManager) *MQTTClient {
	return &MQTTClient{c: c}
}

func NewMQTTClientBuilder(cfg *autopaho.ClientConfig) *MQTTClientBuilder {
	builder := &MQTTClientBuilder{}
	if cfg != nil {
		builder.cfg = cfg
	} else {
		builder.cfg = &autopaho.ClientConfig{}
	}

	return builder
}

func (b *MQTTClientBuilder) AddServers(urls []*url.URL) *MQTTClientBuilder {
	b.cfg.ServerUrls = urls

	return b
}

func (b *MQTTClientBuilder) AddKeepAlive(value uint16) *MQTTClientBuilder {
	b.cfg.KeepAlive = value

	return b
}

func (b *MQTTClientBuilder) AddCleanStartOnInitialConnection(value bool) *MQTTClientBuilder {
	b.cfg.CleanStartOnInitialConnection = value

	return b
}

func (b *MQTTClientBuilder) AddSessionExpiryInterval(value uint32) *MQTTClientBuilder {
	b.cfg.SessionExpiryInterval = value

	return b
}

func (b *MQTTClientBuilder) AddOnConnectionUp(f func(cm *autopaho.ConnectionManager, connAck *paho.Connack)) *MQTTClientBuilder {
	b.cfg.OnConnectionUp = f

	return b
}

func (b *MQTTClientBuilder) AddOnConnectionError(f func(err error)) *MQTTClientBuilder {
	b.cfg.OnConnectError = f

	return b
}

func (b *MQTTClientBuilder) AddClientId(id string) *MQTTClientBuilder {
	b.cfg.ClientConfig.ClientID = id

	return b
}

func (b *MQTTClientBuilder) AddOnPublishReceived(f func(pr paho.PublishReceived) (bool, error)) *MQTTClientBuilder {
	b.cfg.ClientConfig.OnPublishReceived = append(b.cfg.ClientConfig.OnPublishReceived, f)

	return b
}

func (b *MQTTClientBuilder) AddOnClientError(f func(err error)) *MQTTClientBuilder {
	b.cfg.ClientConfig.OnClientError = f

	return b
}

func (b *MQTTClientBuilder) AddOnServerDisconnect(f func(d *paho.Disconnect)) *MQTTClientBuilder {
	b.cfg.ClientConfig.OnServerDisconnect = f

	return b
}

func (b *MQTTClientBuilder) Build(ctx context.Context) (*MQTTClient, error) {
	cm, err := autopaho.NewConnection(ctx, *b.cfg)
	if err != nil {
		return nil, err
	}

	err = cm.AwaitConnection(ctx)
	if err != nil {
		return nil, err
	}

	return NewMQTTClient(cm), nil
}
