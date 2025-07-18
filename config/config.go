package config

import (
	"fmt"
	"os"
)

type Config struct {
	AdminURL      string
	AdminPassword string
	Namespace     string
}

func New() *Config {
	return &Config{}
}

func (c *Config) Load() error {
	var err error
	c.AdminURL, err = c.Get("ADMIN_URL")
	if err != nil {
		return fmt.Errorf("error loading ADMIN_URL: %v", err)
	}
	c.AdminPassword, err = c.Get("ADMIN_PASSWORD")
	if err != nil {
		return fmt.Errorf("error loading ADMIN_PASSWORD: %v", err)
	}

	return nil
}

func (c *Config) Get(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("environment variable %s is not set", key)
	}
	return value, nil
}

func GetConfig() *Config {
	config := New()
	if err := config.Load(); err != nil {
		panic("Failed to load configuration: " + err.Error())
	}
	return config
}
