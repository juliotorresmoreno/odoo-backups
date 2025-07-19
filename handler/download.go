package handler

import (
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"
)

func (h *handler) downloadBackupHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dbName, fileName := vars["dbname"], vars["file"]

	if dbName == "" || fileName == "" {
		http.Error(w, "Missing dbname or file", http.StatusBadRequest)
		return
	}

	base := os.Getenv("ODOO_BACKUP_PATH")
	filePath := path.Join(base, dbName, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Backup file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, filePath)
}
