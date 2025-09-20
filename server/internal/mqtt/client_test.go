package mqtt

import (
	"net/url"
	"testing"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/stretchr/testify/assert"
)

func TestNewMQTTClient(t *testing.T) {
	a := assert.New(t)
	cm := &autopaho.ConnectionManager{}
	c := NewMQTTClient(cm)
	a.NotNil(c, "client should not be nil")
	a.Equal(cm, c.c, "underlying connection manager should be set")
}

func TestNewMQTTClientBuilder_DefaultsAndAdders(t *testing.T) {
	a := assert.New(t)

	cfg := &autopaho.ClientConfig{
		ClientConfig: paho.ClientConfig{},
	}
	b := NewMQTTClientBuilder(cfg)
	a.NotNil(b)
	a.NotNil(b.cfg)

	// servers
	urls := []*url.URL{{Scheme: "tcp", Host: "localhost:1883"}}
	b.AddServers(urls)
	a.Len(cfg.ServerUrls, 1)
	a.Equal("localhost:1883", cfg.ServerUrls[0].Host)

	b.AddKeepAlive(30)
	a.Equal(uint16(30), cfg.KeepAlive)

	b.AddCleanStartOnInitialConnection(true)
	a.Equal(true, cfg.CleanStartOnInitialConnection)

	b.AddSessionExpiryInterval(12345)
	a.Equal(uint32(12345), cfg.SessionExpiryInterval)

	flagOnConnUp := false
	b.AddOnConnectionUp(func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
		flagOnConnUp = true
	})
	a.NotNil(cfg.OnConnectionUp)
	cfg.OnConnectionUp(nil, nil)
	a.True(flagOnConnUp)

	flagOnConnErr := false
	b.AddOnConnectionError(func(err error) {
		flagOnConnErr = true
	})
	a.NotNil(cfg.OnConnectError)
	cfg.OnConnectError(nil)
	a.True(flagOnConnErr)

	b.AddClientId("my-client")
	a.Equal("my-client", cfg.ClientConfig.ClientID)

	b.AddOnPublishReceived(func(pr paho.PublishReceived) (bool, error) {
		return true, nil
	})
	a.Len(cfg.ClientConfig.OnPublishReceived, 1)
	ok, err := cfg.ClientConfig.OnPublishReceived[0](paho.PublishReceived{})
	a.NoError(err)
	a.True(ok)

	flagOnClientErr := false
	b.AddOnClientError(func(err error) {
		flagOnClientErr = true
	})
	a.NotNil(cfg.ClientConfig.OnClientError)
	cfg.ClientConfig.OnClientError(nil)
	a.True(flagOnClientErr)

	flagOnServerDisc := false
	b.AddOnServerDisconnect(func(d *paho.Disconnect) {
		flagOnServerDisc = true
	})
	a.NotNil(cfg.ClientConfig.OnServerDisconnect)
	cfg.ClientConfig.OnServerDisconnect(nil)
	a.True(flagOnServerDisc)
}
