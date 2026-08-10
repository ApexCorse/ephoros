package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestNewInfluxWriterFromEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		environment map[string]string
		wantURL     string
		wantOrg     string
		wantBucket  string
		wantErr     string
	}{
		{
			name:        "defaults",
			environment: map[string]string{"INFLUXDB_TOKEN": "token"},
			wantURL:     defaultInfluxDBURL,
			wantOrg:     defaultInfluxDBOrg,
			wantBucket:  defaultInfluxDBBucket,
		},
		{
			name: "custom values",
			environment: map[string]string{
				"INFLUXDB_URL":         "https://influx.example.test/base",
				"INFLUXDB_TOKEN":       "custom-token",
				"INFLUXDB_INIT_ORG":    "custom-org",
				"INFLUXDB_INIT_BUCKET": "custom-bucket",
			},
			wantURL:    "https://influx.example.test/base",
			wantOrg:    "custom-org",
			wantBucket: "custom-bucket",
		},
		{
			name:        "malformed URL",
			environment: map[string]string{"INFLUXDB_URL": "://bad", "INFLUXDB_TOKEN": "token"},
			wantErr:     "parse INFLUXDB_URL",
		},
		{
			name:        "relative URL",
			environment: map[string]string{"INFLUXDB_URL": "influxdb:8086", "INFLUXDB_TOKEN": "token"},
			wantErr:     "INFLUXDB_URL must be an absolute URL",
		},
		{
			name:        "missing token",
			environment: map[string]string{"INFLUXDB_URL": "https://influx.example.test"},
			wantErr:     "INFLUXDB_TOKEN env var is not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, name := range []string{"INFLUXDB_URL", "INFLUXDB_TOKEN", "INFLUXDB_INIT_ORG", "INFLUXDB_INIT_BUCKET"} {
				t.Setenv(name, tt.environment[name])
			}

			writer, err := NewInfluxWriterFromEnvironment()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, writer)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, writer)
			assert.Equal(t, tt.wantURL, writer.baseURL.String())
			assert.Equal(t, tt.environment["INFLUXDB_TOKEN"], writer.token)
			assert.Equal(t, tt.wantOrg, writer.org)
			assert.Equal(t, tt.wantBucket, writer.bucket)
			assert.Same(t, http.DefaultClient, writer.client)
		})
	}
}

func TestInfluxWriterWrite(t *testing.T) {
	transportErr := errors.New("transport unavailable")
	tests := []struct {
		name      string
		topic     string
		payload   string
		roundTrip roundTripFunc
		wantErr   string
		wantErrIs error
	}{
		{
			name:    "dashboard schema request",
			topic:   `data/electrical/battery voltage,primary=main\bus`,
			payload: `{"value":12.5,"time":"2026-08-10T12:30:00.123456789Z","unit":"V"}`,
			roundTrip: func(request *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(request.Body)
				require.NoError(t, err)
				assert.Equal(t, http.MethodPost, request.Method)
				assert.Equal(t, "/api/v2/write", request.URL.Path)
				assert.Equal(t, "test-org", request.URL.Query().Get("org"))
				assert.Equal(t, "test-bucket", request.URL.Query().Get("bucket"))
				assert.Equal(t, "ns", request.URL.Query().Get("precision"))
				assert.Equal(t, "Token test-token", request.Header.Get("Authorization"))
				assert.Equal(t, "text/plain; charset=utf-8", request.Header.Get("Content-Type"))
				assert.Equal(t, "can_signal,topic=data/electrical/battery\\ voltage\\,primary\\=main\\\\bus value=12.5 1786365000123456789\n", string(body))
				return response(http.StatusNoContent, ""), nil
			},
		},
		{
			name:    "invalid JSON",
			topic:   "data/speed",
			payload: `{`,
			wantErr: "decode simulated payload",
		},
		{
			name:    "invalid timestamp",
			topic:   "data/speed",
			payload: `{"value":1,"time":"not-a-time"}`,
			wantErr: "parse simulated timestamp",
		},
		{
			name:      "transport error",
			topic:     "data/speed",
			payload:   `{"value":1,"time":"2026-08-10T12:30:00Z"}`,
			roundTrip: func(*http.Request) (*http.Response, error) { return nil, transportErr },
			wantErr:   "write to InfluxDB",
			wantErrIs: transportErr,
		},
		{
			name:    "server error with body",
			topic:   "data/speed",
			payload: `{"value":1,"time":"2026-08-10T12:30:00Z"}`,
			roundTrip: func(*http.Request) (*http.Response, error) {
				return response(http.StatusBadRequest, "invalid line protocol\n"), nil
			},
			wantErr: "InfluxDB returned 400 Bad Request: invalid line protocol",
		},
		{
			name:    "redirect status is rejected",
			topic:   "data/speed",
			payload: `{"value":1,"time":"2026-08-10T12:30:00Z"}`,
			roundTrip: func(*http.Request) (*http.Response, error) {
				return response(http.StatusPermanentRedirect, ""), nil
			},
			wantErr: "InfluxDB returned 308 Permanent Redirect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, err := url.Parse("https://influx.example.test/base")
			require.NoError(t, err)
			client := &http.Client{}
			if tt.roundTrip != nil {
				client.Transport = tt.roundTrip
			}
			writer := &InfluxWriter{baseURL: baseURL, token: "test-token", org: "test-org", bucket: "test-bucket", client: client}

			err = writer.Write(context.Background(), tt.topic, []byte(tt.payload))

			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			if tt.wantErrIs != nil {
				assert.ErrorIs(t, err, tt.wantErrIs)
			}
		})
	}
}

func TestEnvironmentOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		want     string
	}{
		{name: "environment value", value: "configured", fallback: "fallback", want: "configured"},
		{name: "fallback", fallback: "fallback", want: "fallback"},
		{name: "both empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SIMULATOR_TEST_VALUE", tt.value)
			assert.Equal(t, tt.want, environmentOrDefault("SIMULATOR_TEST_VALUE", tt.fallback))
		})
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    time.Time
		wantErr string
	}{
		{name: "nanoseconds", value: "2026-08-10T12:30:00.123456789Z", want: time.Date(2026, 8, 10, 12, 30, 0, 123456789, time.UTC)},
		{name: "timezone offset", value: "2026-08-10T14:30:00+02:00", want: time.Date(2026, 8, 10, 14, 30, 0, 0, time.FixedZone("", 2*60*60))},
		{name: "empty", wantErr: "parse simulated timestamp"},
		{name: "invalid", value: "10 August 2026", wantErr: "parse simulated timestamp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimestamp(tt.value)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.True(t, got.IsZero())
				return
			}
			require.NoError(t, err)
			assert.True(t, tt.want.Equal(got))
		})
	}
}

func TestEscapeInfluxTag(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "plain", value: "vehicle/speed", want: "vehicle/speed"},
		{name: "reserved characters", value: `a,b=c d\e`, want: `a\,b\=c\ d\\e`},
		{name: "empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, escapeInfluxTag(tt.value))
		})
	}
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
