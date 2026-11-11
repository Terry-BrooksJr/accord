package utility

import (
	"encoding/json"
	"net/http"
)

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]string{
		"status":  "healthy",
		"db":      "Not Implemented",
		"workers": "Not Implemented",
		"gateway": "OK",
	}

	json.NewEncoder(w).Encode(response)
}
