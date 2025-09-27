package utils

import "net/http"

type Healthcheck struct {
	ready bool
}

func NewHealtcheck() *Healthcheck {
	return &Healthcheck{}
}

func (h *Healthcheck) ReadyzHandler(w http.ResponseWriter, r *http.Request) {
	if !h.ready {
		http.Error(w, "service not ready", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Healthcheck) SetReady(ready bool) {
	h.ready = ready
}