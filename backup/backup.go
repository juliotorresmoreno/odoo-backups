package backup

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/juliotorresmoreno/odoo-backups/db"
	"github.com/juliotorresmoreno/odoo-backups/storage"
)

type OdooBackupConfig struct {
	OdooURL        string
	MasterPassword string
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

	storageClient := storage.NewStorageClient(nil, "default")

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

	if o.BackupFormat == "" {
		o.BackupFormat = "zip"
	}

	if o.OutputDir == "" {
		o.OutputDir = os.TempDir()
		log.Printf("Advertencia: OutputDir no especificado, usando: %s", o.OutputDir)
	}

	backupEndpoint := fmt.Sprintf("%s/web/database/backup", o.OdooURL)

	form := fmt.Sprintf("master_pwd=%s&name=%s&backup_format=%s", o.MasterPassword, dbName, o.BackupFormat)
	log.Printf("Iniciando backup para la DB '%s' en %s", dbName, o.OdooURL)

	req, err := http.NewRequest("POST", backupEndpoint, bytes.NewBufferString(form))
	if err != nil {
		return "", fmt.Errorf("error al crear la petición HTTP: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/octet-stream")

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("no se pudo hacer el backup de la base de datos '%s': %w", dbName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("error en la respuesta del servidor: %s - %s", resp.Status, string(bodyBytes))
	}

	backupFileName := fmt.Sprintf("%s_%s.%s", dbName, time.Now().Format("20060102_150405"), o.BackupFormat)
	fullPath := filepath.Join(o.OutputDir, dbName, backupFileName)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("error al crear directorio '%s': %w", filepath.Dir(fullPath), err)
	}

	outFile, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("error al crear archivo '%s': %w", fullPath, err)
	}
	defer outFile.Close()

	n, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("error al guardar backup: %w", err)
	}

	log.Printf("Backup exitoso: '%s' (%d bytes)", fullPath, n)
	return fullPath, nil
}
