package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/juliotorresmoreno/odoo-backups/backup"
	"github.com/juliotorresmoreno/odoo-backups/config"
	"github.com/juliotorresmoreno/odoo-backups/handler"
	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
)

func main() {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	c := cron.New()
	config := config.GetConfig()
	if config == nil {
		log.Fatal("Failed to load configuration")
	}

	backup := backup.NewOdooBackup(backup.OdooBackupConfig{
		OdooURL:        config.AdminURL,
		MasterPassword: config.AdminPassword,
	})

	// Ejecutar a las 00:00 y 07:00
	c.AddFunc("0 0 * * *", func() {
		_, err := backup.AllDatabases()
		if err != nil {
			log.Printf("Error in backupAllDatabases: %v", err)
		}
	})
	c.AddFunc("0 7 * * *", func() {
		_, err := backup.AllDatabases()
		if err != nil {
			log.Printf("Error in backupAllDatabases: %v", err)
		}
	})
	c.Start()

	handler := handler.ConfigureHandler()

	httpServer := http.Server{
		Addr:    ":3050",
		Handler: handler,
	}

	log.Println(httpServer.ListenAndServe())
}
