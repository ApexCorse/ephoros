package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ApexCorse/vera"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

func main() {
	dbcFilePath := flag.String("dbc-file", os.Getenv("DBC_FILE_PATH"), "path to the DBC file")
	catalogOutput := flag.String("catalog-output", "", "write a C telemetry catalog to this file and exit")
	flag.Parse()

	if *dbcFilePath == "" {
		log.Fatalln("[SIMULATOR_MAIN] DBC_FILE_PATH or --dbc-file is required")
	}

	config, err := getDbcConfig(*dbcFilePath)
	if err != nil {
		log.Fatalf("[SIMULATOR_MAIN] couldn't load DBC config: %s\n", err.Error())
	}

	if *catalogOutput != "" {
		if err := writeTelemetryCatalog(*catalogOutput, getSimulatorSignalsFromConfig(config)); err != nil {
			log.Fatalf("[SIMULATOR_MAIN] couldn't write telemetry catalog: %s\n", err.Error())
		}
		return
	}

	brokerUrl := os.Getenv("BROKER_URL")
	if brokerUrl == "" {
		log.Fatalln("[SIMULATOR_MAIN] missing env variables")
		os.Exit(1)
	}

	intervalStr := os.Getenv("SIMULATOR_INTERVAL")
	interval := 200
	if intervalStr != "" {
		newInterval, err := strconv.Atoi(intervalStr)
		if err == nil {
			interval = newInterval
		}
	}

	parsedUrl, err := url.Parse(brokerUrl)
	if err != nil {
		log.Fatalf("[SIMULATOR_MAIN] couldn't parse url: %s\n", err.Error())
		os.Exit(1)
	}

	topics := getTopicsFromConfig(config)
	if len(topics) == 0 {
		log.Fatalln("[SIMULATOR_MAIN] DBC config contains no MQTT topics")
	}
	log.Printf("[SIMULATOR_MAIN] got %d topics: %v\n", len(topics), topics)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("[SIMULATOR_MAIN] starting MQTT simulator")
	client, err := NewMQTTClientBuilder(nil).
		AddServers([]*url.URL{parsedUrl}).
		AddKeepAlive(20).
		AddCleanStartOnInitialConnection(false).
		AddOnConnectionUp(func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			log.Println("[SIMULATOR_MAIN] MQTT connection up")
		}).
		AddOnConnectionError(func(err error) {
			log.Printf("[SIMULATOR_MAIN] MQTT connection error: %s\n", err.Error())
		}).
		AddClientId("simulator").
		Build(ctx)
	if err != nil {
		log.Fatalf("[SIMULATOR_MAIN] couldn't create MQTT simulator: %s\n", err.Error())
		os.Exit(1)
	}
	log.Println("[SIMULATOR_MAIN] MQTT simulator started")

	influxWriter, err := NewInfluxWriterFromEnvironment()
	if err != nil {
		log.Fatalf("[SIMULATOR_MAIN] couldn't create InfluxDB writer: %s\n", err.Error())
	}
	log.Println("[SIMULATOR_MAIN] InfluxDB writer started")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			data, err := generateRandomData()
			if err != nil {
				log.Fatalf("[SIMULATOR_MAIN] couldn't generate data: %s\n", err.Error())
			}

			i := rand.Intn(len(topics))
			topic := topics[i]
			writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)

			if err := client.Publish(writeCtx, topic, data); err != nil {
				cancel()
				log.Fatalf("[SIMULATOR_MAIN] couldn't send data: %s\n", err.Error())
			}
			if err := influxWriter.Write(writeCtx, topic, data); err != nil {
				cancel()
				log.Fatalf("[SIMULATOR_MAIN] couldn't write data to InfluxDB: %s\n", err.Error())
			}
			cancel()
			log.Printf("[SIMULATOR_MAIN] sent data to topic: %s\n", topic)
		}

		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}

func generateRandomData() ([]byte, error) {
	value := rand.Float32()*1000 - 500

	jsonPayload := struct {
		Value float32   `json:"value"`
		Time  time.Time `json:"time"`
		Unit  string    `json:"unit"`
	}{
		Value: value,
		Time:  time.Now(),
		Unit:  "V",
	}

	data, err := json.Marshal(jsonPayload)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// getDbcConfig reads and parses the config.dbc file
func getDbcConfig(dbcFilePath string) (*vera.Config, error) {
	dbcFile, err := os.Open(dbcFilePath)
	if os.IsNotExist(err) && filepath.Base(dbcFilePath) == "config.dbc" {
		dbcFilePath = filepath.Join(filepath.Dir(dbcFilePath), "config.example.dbc")
		dbcFile, err = os.Open(dbcFilePath)
	}
	if err != nil {
		return nil, fmt.Errorf("error while opening DBC file: %w", err)
	}
	defer dbcFile.Close()

	config, err := vera.Parse(dbcFile)
	if err != nil {
		return nil, fmt.Errorf("error in parsing DBC file: %w", err)
	}

	return config, nil
}

type simulatorSignal struct {
	Topic string
	Name  string
	Unit  string
	Min   float32
	Max   float32
}

// writeTelemetryCatalog exports the DBC signal data needed by the embedded
// telemetry simulator as a C header.
func writeTelemetryCatalog(outputPath string, signals []simulatorSignal) error {
	if len(signals) == 0 {
		return fmt.Errorf("DBC config contains no MQTT topics")
	}
	for _, signal := range signals {
		if math.IsNaN(float64(signal.Min)) || math.IsInf(float64(signal.Min), 0) ||
			math.IsNaN(float64(signal.Max)) || math.IsInf(float64(signal.Max), 0) {
			return fmt.Errorf("DBC signal %q has non-finite bounds", signal.Name)
		}
		if signal.Min > signal.Max {
			return fmt.Errorf("DBC signal %q has minimum %g greater than maximum %g",
				signal.Name, signal.Min, signal.Max)
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err = fmt.Fprint(file, "// Code generated from config.dbc; DO NOT EDIT.\n#pragma once\n\n"); err != nil {
		return err
	}
	if _, err = fmt.Fprint(file, "typedef struct {\n"+
		"\tconst char *topic;\n"+
		"\tconst char *name;\n"+
		"\tconst char *unit;\n"+
		"\tfloat min;\n"+
		"\tfloat max;\n"+
		"} ephoros_telemetry_simulator_signal_t;\n\n"); err != nil {
		return err
	}
	if _, err = fmt.Fprintf(file, "#define EPHOROS_TELEMETRY_SIMULATOR_SIGNAL_COUNT %d\n\n", len(signals)); err != nil {
		return err
	}
	if _, err = fmt.Fprintln(file, "static const ephoros_telemetry_simulator_signal_t ephoros_telemetry_simulator_signals[] = {"); err != nil {
		return err
	}
	for _, signal := range signals {
		// %q produces a valid quoted Go string, whose escaping is also valid C.
		if _, err = fmt.Fprintf(file,
			"\t{.topic = %q, .name = %q, .unit = %q, .min = %s, .max = %s},\n",
			signal.Topic, signal.Name, signal.Unit, formatCFloat(signal.Min), formatCFloat(signal.Max)); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(file, "};")
	return err
}

func formatCFloat(value float32) string {
	formatted := strconv.FormatFloat(float64(value), 'g', -1, 32)
	if !strings.ContainsAny(formatted, ".eE") {
		formatted += ".0"
	}
	return formatted + "f"
}

func getSimulatorSignalsFromConfig(config *vera.Config) []simulatorSignal {
	signals := make([]simulatorSignal, 0)
	for _, message := range config.Messages {
		for _, signal := range message.Signals {
			if signal.Metadata.MQTTTopic == "" {
				continue
			}
			signals = append(signals, simulatorSignal{
				Topic: signal.Metadata.MQTTTopic,
				Name:  signal.Name,
				Unit:  signal.Unit,
				Min:   signal.Min,
				Max:   signal.Max,
			})
		}
	}
	return signals
}

// getTopicsFromConfig extracts MQTT topics from the vera config
func getTopicsFromConfig(config *vera.Config) []string {
	signals := getSimulatorSignalsFromConfig(config)
	topics := make([]string, 0, len(signals))
	for _, signal := range signals {
		topics = append(topics, signal.Topic)
	}

	return topics
}
