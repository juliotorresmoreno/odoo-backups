package db

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/juliotorresmoreno/odoo-backups/config"
)

type DatabaseListResponse struct {
	Jsonrpc string   `json:"jsonrpc"`
	ID      *int     `json:"id"`
	Result  []string `json:"result"`
}

func ListDatabases() ([]string, error) {
	config := config.GetConfig()
	if config.AdminURL == "" || config.AdminPassword == "" {
		return nil, fmt.Errorf("ADMIN_URL and ADMIN_PASSWORD must be set in the environment variables")
	}

	body := bytes.NewBufferString(`{}`)
	url := fmt.Sprintf("%s/web/database/list", config.AdminURL)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status: %s", response.Status)
	}

	var databases DatabaseListResponse
	if err := json.NewDecoder(response.Body).Decode(&databases); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return databases.Result, nil
}
