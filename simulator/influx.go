package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultInfluxDBURL    = "http://influxdb:8086"
	defaultInfluxDBOrg    = "ephoros"
	defaultInfluxDBBucket = "telemetry"
)

// InfluxWriter stores simulated MQTT payloads in InfluxDB using the schema
// consumed by the generated Grafana dashboards.
type InfluxWriter struct {
	baseURL *url.URL
	token   string
	org     string
	bucket  string
	client  *http.Client
}

func NewInfluxWriterFromEnvironment() (*InfluxWriter, error) {
	baseURL := environmentOrDefault("INFLUXDB_URL", defaultInfluxDBURL)
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse INFLUXDB_URL: %w", err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("INFLUXDB_URL must be an absolute URL")
	}

	token := os.Getenv("INFLUXDB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("INFLUXDB_TOKEN env var is not set")
	}

	return &InfluxWriter{
		baseURL: parsedURL,
		token:   token,
		org:     environmentOrDefault("INFLUXDB_INIT_ORG", defaultInfluxDBOrg),
		bucket:  environmentOrDefault("INFLUXDB_INIT_BUCKET", defaultInfluxDBBucket),
		client:  http.DefaultClient,
	}, nil
}

func (w *InfluxWriter) Write(ctx context.Context, topic string, payload []byte) error {
	var data struct {
		Value float32 `json:"value"`
		Time  string  `json:"time"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("decode simulated payload: %w", err)
	}

	timestamp, err := parseTimestamp(data.Time)
	if err != nil {
		return err
	}

	writeURL := w.baseURL.ResolveReference(&url.URL{Path: "/api/v2/write"})
	query := writeURL.Query()
	query.Set("org", w.org)
	query.Set("bucket", w.bucket)
	query.Set("precision", "ns")
	writeURL.RawQuery = query.Encode()

	line := fmt.Sprintf("can_signal,topic=%s value=%s %d\n",
		escapeInfluxTag(topic),
		strconv.FormatFloat(float64(data.Value), 'g', -1, 32),
		timestamp.UnixNano(),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, writeURL.String(), strings.NewReader(line))
	if err != nil {
		return fmt.Errorf("create InfluxDB write request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+w.token)
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	response, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("write to InfluxDB: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return fmt.Errorf("InfluxDB returned %s: %s", response.Status, bytes.TrimSpace(body))
	}

	return nil
}

func environmentOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func parseTimestamp(value string) (time.Time, error) {
	timestamp, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse simulated timestamp: %w", err)
	}
	return timestamp, nil
}

func escapeInfluxTag(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", ",", "\\,", "=", "\\=", " ", "\\ ")
	return replacer.Replace(value)
}
