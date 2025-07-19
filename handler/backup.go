package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func (h *handler) scanBackupsHandler(w http.ResponseWriter, r *http.Request) {
	dbName := mux.Vars(r)["dbname"]
	if dbName == "" {
		http.Error(w, "Missing dbname", http.StatusBadRequest)
		return
	}

	backups, err := h.backup.ListBackups(dbName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(backups); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *handler) backupHandler(w http.ResponseWriter, r *http.Request) {
	backups, err := h.backup.AllDatabases()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(backups)
}
