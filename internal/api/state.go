package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleGetState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Fetch the thread-safe snapshot from the injected state service
	snapshot := "" // snapshot.GetLatest()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(snapshot); err != nil {
		http.Error(w, "Failed to encode state", http.StatusInternalServerError)
		return
	}
}
