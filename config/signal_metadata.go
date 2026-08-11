package main

import (
	"fmt"
	"strings"

	"github.com/ApexCorse/vera"
)

// signalsFromMetadata converts Vera's signal-scoped metadata into the compact
// inputs consumed by dashboard and alert provisioning. Signals without an MQTT
// topic are irrelevant to Grafana unless they define an alert or stale policy;
// in that case failing is safer than silently omitting the policy.
func signalsFromMetadata(config *vera.Config) ([]SignalTopic, []AlertSignal, error) {
	if config == nil {
		return nil, nil, fmt.Errorf("Vera config cannot be nil")
	}

	topics := make([]SignalTopic, 0)
	alerts := make([]AlertSignal, 0)
	for _, message := range config.Messages {
		for _, signal := range message.Signals {
			metadata := signal.Metadata
			hasAlertPolicy := metadata.WarningLow != nil ||
				metadata.WarningHigh != nil ||
				metadata.CriticalLow != nil ||
				metadata.CriticalHigh != nil ||
				metadata.StaleAfterMs != nil
			topic := strings.TrimSpace(metadata.MQTTTopic)

			if topic == "" {
				if hasAlertPolicy {
					return nil, nil, fmt.Errorf(
						"signal %q in message %q defines alert metadata but has no MQTT topic",
						signal.Name,
						message.Name,
					)
				}
				continue
			}

			topics = append(topics, SignalTopic{Topic: topic})
			if !hasAlertPolicy {
				continue
			}

			alerts = append(alerts, AlertSignal{
				Topic:             topic,
				WarningLow:        float64PointerFromFloat32(metadata.WarningLow),
				WarningHigh:       float64PointerFromFloat32(metadata.WarningHigh),
				CriticalLow:       float64PointerFromFloat32(metadata.CriticalLow),
				CriticalHigh:      float64PointerFromFloat32(metadata.CriticalHigh),
				StaleAfterSeconds: staleAfterSeconds(metadata.StaleAfterMs),
				DashboardUID:      detailDashboardKey(topic),
				PanelID:           int(stablePanelID(topic, 'D')),
			})
		}
	}

	return topics, alerts, nil
}

func float64PointerFromFloat32(value *float32) *float64 {
	if value == nil {
		return nil
	}
	converted := float64(*value)
	return &converted
}

// staleAfterSeconds rounds upward so that Grafana never considers a signal
// stale sooner than the DBC's millisecond policy permits.
func staleAfterSeconds(milliseconds *uint32) *int {
	if milliseconds == nil {
		return nil
	}
	seconds := int(*milliseconds / 1000)
	if *milliseconds%1000 != 0 {
		seconds++
	}
	return &seconds
}
