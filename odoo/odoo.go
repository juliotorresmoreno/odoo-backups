package odoo

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/juliotorresmoreno/odoo-backups/db"
	"github.com/juliotorresmoreno/odoo-backups/storage"
)

type OdooAdminConfig struct {
	OdooURL        string
	MasterPassword string
	Namespace      string
	BackupFormat   string
}

type OdooAdmin struct {
	OdooURL        string
	MasterPassword string
	Namespace      string
	BackupFormat   string
	storageClient  *storage.StorageClient
}

func NewOdooAdmin(config OdooAdminConfig) *OdooAdmin {
	if config.OdooURL == "" || config.MasterPassword == "" {
		log.Fatal("OdooURL and MasterPassword must be set")
	}
	if config.BackupFormat == "" {
		config.BackupFormat = "zip"
	}

	storageClient := storage.NewStorageClient(nil, config.Namespace)

	return &OdooAdmin{
		OdooURL:        config.OdooURL,
		MasterPassword: config.MasterPassword,
		Namespace:      config.Namespace,
		BackupFormat:   config.BackupFormat,
		storageClient:  storageClient,
	}
}

func (o *OdooAdmin) AllDatabases() ([]string, error) {
	if o.OdooURL == "" || o.MasterPassword == "" {
		return nil, fmt.Errorf("OdooURL and MasterPassword must be set")
	}

	list, err := db.ListDatabases()
	if err != nil {
		return nil, fmt.Errorf("error al listar bases de datos: %v", err)
	}

	var backups = make([]string, 0)

	for _, dbName := range list {
		if dbName == "postgres" || dbName == "template0" || dbName == "template1" {
			continue
		}

		log.Printf("Iniciando backup de la base de datos: %s", dbName)

		ok, err := o.Backup(dbName)
		if err != nil {
			log.Printf("Error al hacer backup de la base de datos '%s': %v", dbName, err)
			continue
		}
		if ok {
			backups = append(backups, dbName)
		}
	}

	return backups, nil
}

func (o *OdooAdmin) Backup(dbName string) (bool, error) {
	url := fmt.Sprintf("http://executor-%s.%s:4080/backup/%s", dbName, o.Namespace, dbName)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return false, fmt.Errorf("error creating request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)

	return resp.StatusCode == http.StatusOK, nil
}

type Backup struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"createdAt"`
}

func (o *OdooAdmin) ListBackups(dbName string) ([]Backup, error) {
	var backups []Backup

	return backups, nil
}
