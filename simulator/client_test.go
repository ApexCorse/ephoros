package main

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMQTTClient(t *testing.T) {
	tests := []struct {
		name string
		cm   *autopaho.ConnectionManager
	}{
		{name: "connection manager", cm: &autopaho.ConnectionManager{}},
		{name: "nil connection manager"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewMQTTClient(tt.cm)

			require.NotNil(t, client)
			assert.Same(t, tt.cm, client.c)
		})
	}
}

func TestNewMQTTClientBuilder(t *testing.T) {
	existing := &autopaho.ClientConfig{KeepAlive: 12}
	tests := []struct {
		name string
		cfg  *autopaho.ClientConfig
		want *autopaho.ClientConfig
	}{
		{name: "uses provided config", cfg: existing, want: existing},
		{name: "creates default config", want: &autopaho.ClientConfig{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewMQTTClientBuilder(tt.cfg)

			require.NotNil(t, builder)
			require.NotNil(t, builder.cfg)
			if tt.cfg != nil {
				assert.Same(t, tt.want, builder.cfg)
			} else {
				assert.Equal(t, tt.want, builder.cfg)
			}
		})
	}
}

func TestMQTTClientBuilderAdders(t *testing.T) {
	serverURL := &url.URL{Scheme: "tcp", Host: "localhost:1883"}
	callbackErr := errors.New("callback error")
	tests := []struct {
		name   string
		apply  func(*MQTTClientBuilder) *MQTTClientBuilder
		assert func(*testing.T, *autopaho.ClientConfig)
	}{
		{
			name:  "servers",
			apply: func(b *MQTTClientBuilder) *MQTTClientBuilder { return b.AddServers([]*url.URL{serverURL}) },
			assert: func(t *testing.T, cfg *autopaho.ClientConfig) {
				require.Len(t, cfg.ServerUrls, 1)
				assert.Same(t, serverURL, cfg.ServerUrls[0])
			},
		},
		{
			name:  "keep alive",
			apply: func(b *MQTTClientBuilder) *MQTTClientBuilder { return b.AddKeepAlive(30) },
			assert: func(t *testing.T, cfg *autopaho.ClientConfig) {
				assert.Equal(t, uint16(30), cfg.KeepAlive)
			},
		},
		{
			name:  "clean start",
			apply: func(b *MQTTClientBuilder) *MQTTClientBuilder { return b.AddCleanStartOnInitialConnection(true) },
			assert: func(t *testing.T, cfg *autopaho.ClientConfig) {
				assert.True(t, cfg.CleanStartOnInitialConnection)
			},
		},
		{
			name:  "session expiry",
			apply: func(b *MQTTClientBuilder) *MQTTClientBuilder { return b.AddSessionExpiryInterval(12345) },
			assert: func(t *testing.T, cfg *autopaho.ClientConfig) {
				assert.Equal(t, uint32(12345), cfg.SessionExpiryInterval)
			},
		},
		{
			name: "connection up callback",
			apply: func(b *MQTTClientBuilder) *MQTTClientBuilder {
				return b.AddOnConnectionUp(func(*autopaho.ConnectionManager, *paho.Connack) {})
			},
			assert: func(t *testing.T, cfg *autopaho.ClientConfig) {
				require.NotNil(t, cfg.OnConnectionUp)
				assert.NotPanics(t, func() { cfg.OnConnectionUp(nil, nil) })
			},
		},
		{
			name: "connection error callback",
			apply: func(b *MQTTClientBuilder) *MQTTClientBuilder {
				return b.AddOnConnectionError(func(error) {})
			},
			assert: func(t *testing.T, cfg *autopaho.ClientConfig) {
				require.NotNil(t, cfg.OnConnectError)
				assert.NotPanics(t, func() { cfg.OnConnectError(callbackErr) })
			},
		},
		{
			name:  "client id",
			apply: func(b *MQTTClientBuilder) *MQTTClientBuilder { return b.AddClientId("my-client") },
			assert: func(t *testing.T, cfg *autopaho.ClientConfig) {
				assert.Equal(t, "my-client", cfg.ClientConfig.ClientID)
			},
		},
		{
			name: "publish received callback",
			apply: func(b *MQTTClientBuilder) *MQTTClientBuilder {
				return b.AddOnPublishReceived(func(paho.PublishReceived) (bool, error) { return true, callbackErr })
			},
			assert: func(t *testing.T, cfg *autopaho.ClientConfig) {
				require.Len(t, cfg.ClientConfig.OnPublishReceived, 1)
				got, err := cfg.ClientConfig.OnPublishReceived[0](paho.PublishReceived{})
				assert.True(t, got)
				assert.ErrorIs(t, err, callbackErr)
			},
		},
		{
			name: "client error callback",
			apply: func(b *MQTTClientBuilder) *MQTTClientBuilder {
				return b.AddOnClientError(func(error) {})
			},
			assert: func(t *testing.T, cfg *autopaho.ClientConfig) {
				require.NotNil(t, cfg.ClientConfig.OnClientError)
				assert.NotPanics(t, func() { cfg.ClientConfig.OnClientError(callbackErr) })
			},
		},
		{
			name: "server disconnect callback",
			apply: func(b *MQTTClientBuilder) *MQTTClientBuilder {
				return b.AddOnServerDisconnect(func(*paho.Disconnect) {})
			},
			assert: func(t *testing.T, cfg *autopaho.ClientConfig) {
				require.NotNil(t, cfg.ClientConfig.OnServerDisconnect)
				assert.NotPanics(t, func() { cfg.ClientConfig.OnServerDisconnect(nil) })
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &autopaho.ClientConfig{}
			builder := NewMQTTClientBuilder(cfg)

			returned := tt.apply(builder)

			assert.Same(t, builder, returned)
			tt.assert(t, cfg)
		})
	}
}

func TestMQTTClientBuilderBuildErrors(t *testing.T) {
	canceledContext, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name    string
		ctx     context.Context
		servers []*url.URL
		wantErr string
	}{
		{name: "missing servers", ctx: context.Background(), wantErr: "no server urls provided"},
		{
			name:    "connection context canceled",
			ctx:     canceledContext,
			servers: []*url.URL{{Scheme: "tcp", Host: "127.0.0.1:1"}},
			wantErr: context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewMQTTClientBuilder(nil).AddServers(tt.servers)

			client, err := builder.Build(tt.ctx)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Nil(t, client)
		})
	}
}
