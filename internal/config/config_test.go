package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveFileAndLoadEncryptedFields(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv(configEncryptionEnv, "super-secret-passphrase")

	savedPath, err := SaveFile(&Config{
		DBDriver:               "postgres",
		DBUrl:                  "postgres://user:pass@localhost:5432/tango?sslmode=disable",
		APIKey:                 "sk-test-secret",
		BaseURL:                "http://localhost:8080",
		ChatChannel:            "discord",
		ChatModel:              "gpt-4.1-mini",
		LLMConfigEncryptionKey: "12345678901234567890123456789012",
	})
	if err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	data, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	if strings.Contains(text, "postgres://user:pass@localhost:5432/tango?sslmode=disable") {
		t.Fatalf("database url should be encrypted in file")
	}
	if strings.Contains(text, "sk-test-secret") {
		t.Fatalf("api key should be encrypted in file")
	}

	cfg := Load()
	if cfg.DBDriver != "postgres" {
		t.Fatalf("DBDriver = %q, want %q", cfg.DBDriver, "postgres")
	}
	if cfg.DBUrl != "postgres://user:pass@localhost:5432/tango?sslmode=disable" {
		t.Fatalf("DBUrl = %q", cfg.DBUrl)
	}
	if cfg.APIKey != "sk-test-secret" {
		t.Fatalf("APIKey = %q", cfg.APIKey)
	}
	if cfg.ChatChannel != "discord" {
		t.Fatalf("ChatChannel = %q", cfg.ChatChannel)
	}
	if cfg.LLMConfigEncryptionKey != "12345678901234567890123456789012" {
		t.Fatalf("LLMConfigEncryptionKey = %q", cfg.LLMConfigEncryptionKey)
	}
}

func TestSaveFilePreservesExistingLLMKey(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv(configEncryptionEnv, "")

	_, err := SaveFile(&Config{
		DBDriver:               "sqlite",
		DBUrl:                  "file:tango.db?_foreign_keys=on",
		LLMConfigEncryptionKey: "12345678901234567890123456789012",
	})
	if err != nil {
		t.Fatalf("SaveFile() initial error = %v", err)
	}

	_, err = SaveFile(&Config{
		DBDriver: "postgres",
		DBUrl:    "postgres://override:override@localhost:5432/override?sslmode=disable",
	})
	if err != nil {
		t.Fatalf("SaveFile() overwrite error = %v", err)
	}

	cfg := Load()
	if cfg.LLMConfigEncryptionKey != "12345678901234567890123456789012" {
		t.Fatalf("LLMConfigEncryptionKey = %q", cfg.LLMConfigEncryptionKey)
	}
}

func TestLoadEnvOverridesFile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv(configEncryptionEnv, "")

	_, err := SaveFile(&Config{
		DBDriver: "sqlite",
		DBUrl:    "file:local.db?_foreign_keys=on",
		BaseURL:  "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DATABASE_URL", "postgres://override:override@localhost:5432/override?sslmode=disable")

	cfg := Load()
	if cfg.DBDriver != "postgres" {
		t.Fatalf("DBDriver = %q, want postgres", cfg.DBDriver)
	}
	if cfg.DBUrl != "postgres://override:override@localhost:5432/override?sslmode=disable" {
		t.Fatalf("DBUrl = %q", cfg.DBUrl)
	}

	expectedPath := filepath.Join(tempHome, ".config", configDirName, configFileName)
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected config file at %s: %v", expectedPath, err)
	}
}
