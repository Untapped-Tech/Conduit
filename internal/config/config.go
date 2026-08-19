package config

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/untappedtech/conduit/internal/domain"
	"gopkg.in/yaml.v3"
)

func Load(customConfigPath string) (*domain.ServerConfig, error) {
	if customConfigPath != "" {
		return loadConfigFile(customConfigPath)
	}

	fallbackPaths := []string{
		"config.json",
		"config.yaml",
		"config.yml",
		"config.toml",
		"config.xml",
	}

	for _, filePath := range fallbackPaths {
		if _, err := os.Stat(filePath); err == nil {
			return loadConfigFile(filePath)
		}
	}

	return nil, fmt.Errorf("no config file found in fallback order (json, yaml, toml, xml)")
}

func loadConfigFile(filePath string) (*domain.ServerConfig, error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	serverConfig := &domain.ServerConfig{}
	lowerFilePath := strings.ToLower(filePath)

	switch {
	case strings.HasSuffix(lowerFilePath, ".json"):
		err = json.Unmarshal(fileData, serverConfig)
	case strings.HasSuffix(lowerFilePath, ".yaml") || strings.HasSuffix(lowerFilePath, ".yml"):
		err = yaml.Unmarshal(fileData, serverConfig)
	case strings.HasSuffix(lowerFilePath, ".toml"):
		err = toml.Unmarshal(fileData, serverConfig)
	case strings.HasSuffix(lowerFilePath, ".xml"):
		err = xml.Unmarshal(fileData, serverConfig)
	default:
		return nil, fmt.Errorf("unsupported config file extension: %s", filePath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config %s: %w", filePath, err)
	}

	if serverConfig.Server.DefaultLimit <= 0 {
		serverConfig.Server.DefaultLimit = 50
	}

	return serverConfig, nil
}

func GenerateDefaultConfig(formatType string, outputFilePath string) error {
	defaultConfig := &domain.ServerConfig{}
	defaultConfig.Server.Host = "0.0.0.0"
	defaultConfig.Server.Port = 8080
	defaultConfig.Server.DefaultLimit = 50

	defaultConfig.Database.Driver = "sqlite"
	defaultConfig.Database.DSN = "./app.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=page_size(16384)"

	defaultConfig.Policy.PublicReads = true
	defaultConfig.Policy.PublicWrites = true
	defaultConfig.Policy.PublicMutation = true

	defaultConfig.Auth.EnvironmentEnabled = false
	defaultConfig.Auth.DBEnabled = false
	defaultConfig.Auth.DBAuth.Driver = "sqlite"
	defaultConfig.Auth.DBAuth.DSN = "./auth.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	defaultConfig.Auth.DBAuth.Table = "users"
	defaultConfig.Auth.DBAuth.TokenColumn = "token"
	defaultConfig.Auth.DBAuth.RoleColumn = "role"
	defaultConfig.Auth.DBAuth.Roles = map[string]string{
		"admin":     "admin",
		"readWrite": "rw",
		"readOnly":  "ro",
	}
	defaultConfig.Auth.DBAuth.Cache.Capacity = 10000
	defaultConfig.Auth.DBAuth.Cache.TTLSeconds = 300

	var encodedBytes []byte
	var marshalError error

	switch strings.ToLower(formatType) {
	case "json":
		encodedBytes, marshalError = json.MarshalIndent(defaultConfig, "", "  ")
	case "yaml", "yml":
		encodedBytes, marshalError = yaml.Marshal(defaultConfig)
	case "toml":
		encodedBytes, marshalError = toml.Marshal(defaultConfig)
	case "xml":
		encodedBytes, marshalError = xml.MarshalIndent(defaultConfig, "", "  ")
		if marshalError == nil {
			encodedBytes = append([]byte(xml.Header), encodedBytes...)
		}
	default:
		return fmt.Errorf("unsupported format for config generation: %s", formatType)
	}

	if marshalError != nil {
		return fmt.Errorf("failed to marshal default config: %w", marshalError)
	}

	return os.WriteFile(outputFilePath, encodedBytes, 0644)
}
