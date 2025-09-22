package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/ApexCorse/ephoros/server/internal/db"
	"github.com/gorilla/mux"
)

type APIConfig struct {
	Address string
	DB      *db.DB
	Router  *mux.Router
}

type API struct {
	db      *db.DB
	r       *mux.Router
	address string
}

func NewAPI(cfg *APIConfig) *API {
	log.Println("[API] Initializing API with configuration")

	if cfg.DB == nil {
		log.Println("[API] db is nil, caution")
	} else {
		log.Println("[API] Database connection configured")
	}

	if cfg.Router == nil {
		log.Println("[API] router is nil, caution")
	} else {
		log.Println("[API] Router configured")
	}

	if cfg.Address == "" {
		log.Println("[API] address is empty, caution")
	} else {
		log.Printf("[API] Server will listen on: %s", cfg.Address)
	}

	return &API{
		db:      cfg.DB,
		r:       cfg.Router,
		address: cfg.Address,
	}
}

func (a *API) Start() {
	log.Println("[API] Starting API server")

	a.r.HandleFunc("/auth", a.handleAuth).Methods("POST")
	a.r.HandleFunc("/data", a.handleSendData).Methods("POST")

	log.Printf("[API] Routes registered - listening on %s", a.address)

	if err := http.ListenAndServe(a.address, a.r); err != nil {
		log.Printf("[API] Server failed to start: %v", err)
	}
}

func (a *API) handleAuth(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] Auth request received from %s", r.RemoteAddr)

	token, err := a.getTokenFromRequest(r)
	if err != nil {
		log.Printf("[API] Authentication failed - token error: %v", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err := a.validateUser(token); err != nil {
		log.Printf("[API] Authentication failed - validation error: %v", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	log.Printf("[API] Authentication successful for token: %s", token+"...")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(
		map[string]string{
			"message": "Authorized",
		},
	)
}

func (a *API) handleSendData(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] Data request received from %s", r.RemoteAddr)

	token, err := a.getTokenFromRequest(r)
	if err != nil {
		log.Printf("[API] Data request failed - token error: %v", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err := a.validateUser(token); err != nil {
		log.Printf("[API] Data request failed - validation error: %v", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	body := &DataRequestBody{}
	if err := json.NewDecoder(r.Body).Decode(body); err != nil {
		log.Printf("[API] Data request failed - JSON decode error: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	log.Printf("[API] Processing data request - Sensor: %s, Module: %s, Section: %s",
		body.Sensor, body.Module, body.Section)

	if !body.Validate() {
		log.Printf("[API] Data request failed - validation failed for request body")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sensor, err := a.db.GetSensorByNameAndModuleAndSection(
		body.Sensor,
		body.Module,
		body.Section,
		body.From,
		body.To,
	)
	if err != nil {
		log.Printf("[API] Data request failed - sensor not found: %v", err)
		http.Error(w, "sensor not found", http.StatusNotFound)
		return
	}

	log.Printf("[API] Data request successful - found %d records for sensor %s",
		len(sensor.Records), sensor.Name)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(
		map[string]any{
			"module":  body.Module,
			"name":    sensor.Name,
			"records": sensor.Records,
			"section": body.Section,
		},
	)
}

func (a *API) validateUser(token string) error {
	log.Printf("[API] Validating user token: %s", token+"...")

	_, err := a.db.GetUserByToken(token)
	if err != nil {
		log.Printf("[API] User validation failed: %v", err)
		return errors.New("invalid credentials")
	}

	log.Println("[API] User validation successful")
	return nil
}

func (a *API) getTokenFromRequest(r *http.Request) (string, error) {
	token := r.Header.Get("Authorization")
	if token == "" {
		log.Println("[API] No authorization header provided")
		return "", errors.New("no token provided")
	}

	parts := strings.Split(token, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		log.Printf("[API] Invalid token format - expected 'Bearer <token>', got: %s", token)
		return "", errors.New("invalid token format")
	}

	log.Printf("[API] Token extracted successfully: %s", parts[1]+"...")
	return parts[1], nil
}
