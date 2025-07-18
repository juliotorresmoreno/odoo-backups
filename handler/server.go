package handler

import (
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
)

func HttpHandler() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		databases, err := db.ListDatabases()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(databases)
	}).Methods("GET")
	r.HandleFunc("/backup", BackupHandler()).Methods("POST")
	r.PathPrefix("/backup/{dbname}").Handler(http.StripPrefix("/backup/", http.HandlerFunc(ScanBackupDirectory))).Methods("GET")
	r.PathPrefix("/download/{dbname}/{file}").Handler(http.StripPrefix("/download/", http.HandlerFunc(DownloadBackup))).Methods("GET")

	return r
}

func DownloadBackup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dbName := vars["dbname"]
	fileName := vars["file"]

	if dbName == "" || fileName == "" {
		http.Error(w, "Database name and file name are required", http.StatusBadRequest)
		return
	}

	outputDir := os.Getenv("ODOO_BACKUP_PATH")
	filePath := path.Join(outputDir, dbName, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Backup file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, filePath)
}

type Backup struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"createdAt"`
}

func ScanBackupDirectory(w http.ResponseWriter, r *http.Request) {
	OutputDir := os.Getenv("ODOO_BACKUP_PATH")
	dbName := mux.Vars(r)["dbname"]
	if dbName == "" {
		http.Error(w, "Database name is required", http.StatusBadRequest)
		return
	}

	paths, err := os.ReadDir(path.Join(OutputDir, dbName))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading backup directory: %v", err), http.StatusInternalServerError)
		return
	}

	var backups []Backup
	for _, path := range paths {
		if !path.IsDir() && path.Name() != "README.md" {
			stats, err := path.Info()
			if err != nil {
				log.Printf("Error getting file info for %s: %v", path.Name(), err)
				continue
			}

			backups = append(backups, Backup{
				Name:      path.Name(),
				Size:      stats.Size(),
				CreatedAt: stats.ModTime().Format(time.RFC3339),
			})
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"files":  backups,
		"dbname": dbName,
		"output": OutputDir,
	})
}

func BackupHandler() http.HandlerFunc {
	config := config.GetConfig()
	if config == nil {
		log.Fatal("Failed to load configuration")
	}

	backup := backup.NewOdooBackup(backup.OdooBackupConfig{
		OdooURL:        config.AdminURL,
		MasterPassword: config.AdminPassword,
	})

	return func(w http.ResponseWriter, r *http.Request) {
		backups, err := backup.AllDatabases()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(backups)
	}
}
