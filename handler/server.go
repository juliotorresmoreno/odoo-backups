package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/juliotorresmoreno/odoo-backups/config"
	"github.com/juliotorresmoreno/odoo-backups/db"
	"github.com/juliotorresmoreno/odoo-backups/odoo"
	"github.com/juliotorresmoreno/odoo-backups/storage"
)

type handler struct {
	storage *storage.StorageClient
	backup  *odoo.OdooAdmin
}

func ConfigureHandler() http.Handler {
	config := config.GetConfig()
	if config.AdminURL == "" || config.AdminPassword == "" {
		log.Fatal("AdminURL and AdminPassword must be set in the environment variables")
	}

	h := &handler{
		storage: storage.NewStorageClient(nil, config.Namespace),
		backup: odoo.NewOdooAdmin(odoo.OdooAdminConfig{
			OdooURL:        config.AdminURL,
			MasterPassword: config.AdminPassword,
			Namespace:      config.Namespace,
		}),
	}
	r := mux.NewRouter()

	r.HandleFunc("/", h.listDatabasesHandler).Methods("GET")
	r.HandleFunc("/configure", h.configureHandler).Methods("POST")
	r.HandleFunc("/backup", h.backupHandler).Methods("POST")
	r.HandleFunc("/backup/{dbname}", h.scanBackupsHandler).Methods("GET")
	r.HandleFunc("/download/{dbname}/{file}", h.downloadBackupHandler).Methods("GET")

	return r
}

func (h *handler) listDatabasesHandler(w http.ResponseWriter, r *http.Request) {
	databases, err := db.ListDatabases()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(databases)
}
