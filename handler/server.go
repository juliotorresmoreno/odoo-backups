package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gorilla/mux"
	"github.com/juliotorresmoreno/odoo-backups/backup"
	"github.com/juliotorresmoreno/odoo-backups/config"
	"github.com/juliotorresmoreno/odoo-backups/db"
	"github.com/juliotorresmoreno/odoo-backups/storage"
)

type handler struct {
	storage *storage.StorageClient
	backup  *backup.OdooBackup
}

func ConfigureHandler() http.Handler {
	config := config.GetConfig()
	if config.AdminURL == "" || config.AdminPassword == "" {
		log.Fatal("AdminURL and AdminPassword must be set in the environment variables")
	}

	h := &handler{
		storage: storage.NewStorageClient(nil, config.Namespace),
		backup: backup.NewOdooBackup(backup.OdooBackupConfig{
			OdooURL:        config.AdminURL,
			MasterPassword: config.AdminPassword,
			Namespace:      config.Namespace,
			OutputDir:      "/data/odoo-backups",
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

type ConfigureRequest struct {
	DBName string `json:"dbname"`
	Size   int    `json:"size"`
}

func (h *handler) configureHandler(w http.ResponseWriter, r *http.Request) {
	var req ConfigureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.DBName == "" || req.Size <= 0 {
		http.Error(w, "Invalid request: DBName and Size must be provided", http.StatusBadRequest)
		return
	}

	err := h.storage.CreatePVC(context.TODO(), req.DBName, "longhorn", req.Size)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("PVC created for database %s with size %dGi", req.DBName, req.Size)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("PVC created for database %s with size %dGi", req.DBName, req.Size),
	})
}

func (h *handler) backupHandler(w http.ResponseWriter, r *http.Request) {
	backups, err := h.backup.AllDatabases()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

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

func (h *handler) scanBackupsHandler(w http.ResponseWriter, r *http.Request) {
	dbName := mux.Vars(r)["dbname"]
	if dbName == "" {
		http.Error(w, "Missing dbname", http.StatusBadRequest)
		return
	}

	base := os.Getenv("ODOO_BACKUP_PATH")
	files, err := os.ReadDir(path.Join(base, dbName))
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot read dir: %v", err), http.StatusInternalServerError)
		return
	}

	type Backup struct {
		Name      string `json:"name"`
		Size      int64  `json:"size"`
		CreatedAt string `json:"createdAt"`
	}
	var backups []Backup
	for _, f := range files {
		if f.IsDir() || f.Name() == "README.md" {
			continue
		}
		info, err := f.Info()
		if err != nil {
			log.Println("stat err:", err)
			continue
		}
		backups = append(backups, Backup{
			Name:      f.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime().Format(time.RFC3339),
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"files":  backups,
		"dbname": dbName,
		"output": base,
	})
}
