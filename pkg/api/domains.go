package api

import (
	"encoding/json"
	"net/http"
	"pkg/config"
)

func UpdateDomainConfigsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var newConfigs []config.DomainConfig
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&newConfigs); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := config.SaveDomainConfigs(newConfigs); err != nil {
		http.Error(w, "Failed to save domain configurations", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Domain configurations updated successfully"))
}
