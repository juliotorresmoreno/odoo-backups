package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

func (h *handler) downloadBackupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dbName, fileName := vars["dbname"], vars["file"]

	if dbName == "" || fileName == "" {
		http.Error(w, "Missing dbname or file", http.StatusBadRequest)
		return
	}

	file, err := h.backup.Download(dbName, fileName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error downloading backup: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := io.Copy(w, file); err != nil {
		http.Error(w, fmt.Sprintf("Error writing response: %v", err), http.StatusInternalServerError)
		return
	}
}
