package database_mgmt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
)

type TursoDatabase struct {
	Name     string `json:"name"`
	DbId     string `json:"db_id"`
	Hostname string `json:"hostname"`
}

type TursoCreateResponse struct {
	Database TursoDatabase `json:"database"`
}

type TursoTokenResponse struct {
	JWT string `json:"jwt"`
}

// CreateTursoDatabase provisions a new database on Turso
func CreateTursoDatabase(name string) (*TursoDatabase, error) {
	url := fmt.Sprintf("https://api.turso.io/v1/organizations/%s/databases", config.AppConfig.TursoOrg)
	
	body, _ := json.Marshal(map[string]string{
		"name": name,
		"group": "default", // Default group for Turso
	})

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+config.AppConfig.TursoAPIToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("turso api error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result TursoCreateResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return &result.Database, nil
}

// CreateTursoToken generates an access token for a specific database
func CreateTursoToken(dbName string) (string, error) {
	url := fmt.Sprintf("https://api.turso.io/v1/organizations/%s/databases/%s/auth/tokens", config.AppConfig.TursoOrg, dbName)
	
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer "+config.AppConfig.TursoAPIToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("turso api error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result TursoTokenResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return result.JWT, nil
}

// ListTursoDatabases fetches all databases in the organization
func ListTursoDatabases() ([]TursoDatabase, error) {
	url := fmt.Sprintf("https://api.turso.io/v1/organizations/%s/databases", config.AppConfig.TursoOrg)
	
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+config.AppConfig.TursoAPIToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("turso api error (%d)", resp.StatusCode)
	}

	var result struct {
		Databases []TursoDatabase `json:"databases"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Databases, nil
}
