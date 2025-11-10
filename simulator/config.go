package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ApexCorse/vera"
)

// loadTopicsFromDBC reads topics from the config.dbc file using the vera package
func loadTopicsFromDBC(dbcPath string) ([]string, error) {
	log.Printf("[SIMULATOR] Loading topics from DBC file: %s", dbcPath)

	file, err := os.Open(dbcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DBC file: %w", err)
	}
	defer file.Close()

	config, err := vera.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DBC file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("DBC config validation failed: %w", err)
	}

	topics := make([]string, 0)

	// First, try to get topics from config.Topics (if vera populated them)
	for _, signalTopic := range config.Topics {
		if signalTopic.Topic != "" {
			topics = append(topics, signalTopic.Topic)
			log.Printf("[SIMULATOR] Found topic from config.Topics: %s (signal: %s)", signalTopic.Topic, signalTopic.Signal)
		}
	}

	// If no topics found, generate them from messages and signals
	if len(topics) == 0 {
		for _, msg := range config.Messages {
			for _, sig := range msg.Signals {
				// If signal has a topic, use it
				if sig.Topic != "" {
					topics = append(topics, sig.Topic)
					log.Printf("[SIMULATOR] Found topic from signal: %s", sig.Topic)
				} else {
					// Otherwise generate topic from message/signal name
					topic := fmt.Sprintf("%s/%s", msg.Name, sig.Name)
					topics = append(topics, topic)
					log.Printf("[SIMULATOR] Generated topic: %s", topic)
				}
			}
		}
	}

	// If still no topics, generate from message names only
	if len(topics) == 0 {
		log.Printf("[SIMULATOR] No signals found, generating topics from message names")
		for _, msg := range config.Messages {
			topic := msg.Name
			topics = append(topics, topic)
			log.Printf("[SIMULATOR] Generated topic from message: %s", topic)
		}
	}

	if len(topics) == 0 {
		return nil, fmt.Errorf("no topics found in DBC file")
	}

	log.Printf("[SIMULATOR] Loaded %d topics from DBC file", len(topics))
	return topics, nil
}
