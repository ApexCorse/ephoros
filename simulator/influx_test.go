package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestInfluxWriterWriteUsesDashboardSchema(t *testing.T) {
	var gotRequest *http.Request
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotRequest = request
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		gotBody = string(body)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	writer := &InfluxWriter{baseURL: baseURL, token: "test-token", org: "test-org", bucket: "test-bucket", client: server.Client()}
	payload := []byte(`{"value":12.5,"time":"2026-08-10T12:30:00.123456789Z","unit":"V"}`)

	if err := writer.Write(context.Background(), "data/electrical/battery voltage", payload); err != nil {
		t.Fatal(err)
	}

	if gotRequest.URL.Path != "/api/v2/write" {
		t.Errorf("path = %q, want /api/v2/write", gotRequest.URL.Path)
	}
	if gotRequest.URL.Query().Get("org") != "test-org" || gotRequest.URL.Query().Get("bucket") != "test-bucket" {
		t.Errorf("query = %q, want org and bucket", gotRequest.URL.RawQuery)
	}
	if gotRequest.Header.Get("Authorization") != "Token test-token" {
		t.Errorf("authorization = %q", gotRequest.Header.Get("Authorization"))
	}
	if !strings.HasPrefix(gotBody, "can_signal,topic=data/electrical/battery\\ voltage value=12.5 1786365000123456789\n") {
		t.Errorf("line protocol = %q", gotBody)
	}
}

func TestEscapeInfluxTag(t *testing.T) {
	if got, want := escapeInfluxTag(`a,b=c d\\e`), `a\,b\=c\ d\\\\e`; got != want {
		t.Errorf("escapeInfluxTag() = %q, want %q", got, want)
	}
}
