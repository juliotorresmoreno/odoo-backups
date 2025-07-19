package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/juliotorresmoreno/odoo-backups/db"
	"github.com/juliotorresmoreno/odoo-backups/storage"
)

type OdooBackupConfig struct {
	OdooURL        string
	MasterPassword string
	Namespace      string
	BackupFormat   string
	OutputDir      string
}

type OdooBackup struct {
	OdooURL        string
	MasterPassword string
	BackupFormat   string
	OutputDir      string

	storageClient *storage.StorageClient
}

func NewOdooBackup(config OdooBackupConfig) *OdooBackup {
	if config.OdooURL == "" || config.MasterPassword == "" {
		log.Fatal("OdooURL and MasterPassword must be set")
	}
	if config.BackupFormat == "" {
		config.BackupFormat = "zip"
	}
	if config.OutputDir == "" {
		config.OutputDir = os.TempDir()
		log.Printf("Warning: OutputDir not specified, using: %s", config.OutputDir)
	}

	storageClient := storage.NewStorageClient(nil, config.Namespace)

	return &OdooBackup{
		OdooURL:        config.OdooURL,
		MasterPassword: config.MasterPassword,
		BackupFormat:   config.BackupFormat,
		OutputDir:      config.OutputDir,
		storageClient:  storageClient,
	}
}

func (o *OdooBackup) AllDatabases() ([]string, error) {
	if o.OdooURL == "" || o.MasterPassword == "" {
		return nil, fmt.Errorf("OdooURL and MasterPassword must be set")
	}

	list, err := db.ListDatabases()
	if err != nil {
		return nil, fmt.Errorf("error al listar bases de datos: %v", err)
	}

	var backups []string

	for _, dbName := range list {
		if dbName == "postgres" || dbName == "template0" || dbName == "template1" {
			continue
		}

		log.Printf("Iniciando backup de la base de datos: %s", dbName)

		backupPath, err := o.OdooDatabase(dbName)
		if err != nil {
			log.Printf("Error al hacer backup de la base de datos '%s': %v", dbName, err)
			continue
		}
		backups = append(backups, backupPath)
	}

	return backups, nil
}

func (o *OdooBackup) OdooDatabase(dbName string) (string, error) {
	if o.OdooURL == "" || o.MasterPassword == "" || dbName == "" {
		return "", fmt.Errorf("OdooURL, MasterPassword y dbName no pueden estar vacíos")
	}

	exists, err := o.storageClient.ExistsPVC(context.TODO(), dbName)
	if !exists || err != nil {
		return "", fmt.Errorf("PVC for database '%s' does not exist", dbName)
	}

	backupFileName := fmt.Sprintf("%s_%s.%s", dbName, time.Now().Format("20060102_150405"), o.BackupFormat)
	fullPath := filepath.Join(o.OutputDir, backupFileName)

	backupCmd := fmt.Sprintf("/app/main -backup %s", dbName)

	commands := []string{"mkdir -p " + o.OutputDir, backupCmd}
	_, err = o.storageClient.ExecuteWithPVC(context.TODO(), "executor", dbName, commands)

	return fullPath, err
}

type Backup struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"createdAt"`
}

func (o *OdooBackup) ListBackups(dbName string) ([]Backup, error) {
	var backups []Backup
	if o.OdooURL == "" || o.MasterPassword == "" || dbName == "" {
		return backups, fmt.Errorf("OdooURL, MasterPassword y dbName no pueden estar vacíos")
	}

	exists, err := o.storageClient.ExistsPVC(context.TODO(), dbName)
	if !exists || err != nil {
		return backups, fmt.Errorf("PVC for database '%s' does not exist", dbName)
	}

	listCmd := "/app/main -list"

	commands := []string{"mkdir -p " + o.OutputDir, listCmd}
	response, err := o.storageClient.ExecuteWithPVC(context.TODO(), "executor", dbName, commands)

	if err != nil {
		return backups, fmt.Errorf("error al listar backups: %v", err)
	}

	fmt.Println(response)

	json.Unmarshal([]byte(response), &backups)

	return backups, nil
}
