package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	configDirName              = "demo"
	defaultBuildKitHost        = "tcp://buildkitd:1234"
	defaultBuildWorkspaceDir   = "/workspace/jobs"
	configFileName          = "config.json"
	encryptedValuePrefix    = "enc:v1:"
	configEncryptionEnv     = "CONFIG_FILE_ENCRYPTION_KEY"
	defaultPostgresDBURL    = "postgres://postgres:postgres@localhost:5432/tango?sslmode=disable"
	defaultSQLiteDBURL      = "file:tango.db?_foreign_keys=on"
	defaultAPIBaseURL       = "http://localhost:8080"
	defaultChatModel        = "gpt-4.1-mini"
	defaultDBDriver         = "postgres"
	defaultPort             = "8080"
	defaultCacheDriver      = "memory"
	defaultExecutionEngine  = "custom"
	defaultSkillsStorageDir = "data/skills"
	defaultLogFormat        = "text"
	defaultLogOutput        = "both"
	defaultLogFilePath      = "logs/tango.log"
	defaultDiscordMention   = true
	defaultDiscordTyping    = true
	defaultTelegramTyping   = true
)

type Config struct {
	Port                string
	DBDriver            string
	DBUrl               string
	APIKey              string
	BaseURL             string
	ChatChannel         string
	ChatModel           string
	CacheDriver         string
	CacheDefaultTTL     time.Duration
	SkillsStorageDir    string
	LogFormat           string
	LogOutput           string
	LogFilePath         string
	LogMaxSizeMB        int
	LogMaxBackups       int
	LogMaxAgeDays       int
	LogCompress         bool
	OrchestrationEngine string
	WorkflowEngine      string

	LLMConfigEncryptionKey string

	DiscordToken                      string
	DiscordRequireMention             bool
	DiscordEnableTyping               bool
	DiscordEnableMessageContentIntent bool
	DiscordAllowedUserIDs             map[string]bool

	TelegramToken          string
	TelegramEnableTyping   bool
	TelegramAllowedUserIDs map[string]bool

	// Build service
	BuildKitHost         string
	BuildWorkspaceDir    string
	BuildRegistryHost    string
	BuildRegistryUser    string
	BuildRegistryPass    string
}

type fileConfig struct {
	Port                   string `json:"port,omitempty"`
	DBDriver               string `json:"db_driver,omitempty"`
	DBURL                  string `json:"database_url,omitempty"`
	APIKey                 string `json:"api_key,omitempty"`
	BaseURL                string `json:"base_url,omitempty"`
	ChatChannel            string `json:"chat_channel,omitempty"`
	ChatModel              string `json:"chat_model,omitempty"`
	OrchestrationEngine    string `json:"orchestration_engine,omitempty"`
	WorkflowEngine         string `json:"workflow_engine,omitempty"`
	LLMConfigEncryptionKey string `json:"llm_config_encryption_key,omitempty"`
}

func Load() *Config {
	cfg := &Config{
		Port:                              defaultPort,
		DBDriver:                          defaultDBDriver,
		DBUrl:                             defaultPostgresDBURL,
		BaseURL:                           defaultAPIBaseURL,
		ChatModel:                         defaultChatModel,
		CacheDriver:                       defaultCacheDriver,
		CacheDefaultTTL:                   time.Minute,
		SkillsStorageDir:                  defaultSkillsStorageDir,
		LogFormat:                         defaultLogFormat,
		LogOutput:                         defaultLogOutput,
		LogFilePath:                       defaultLogFilePath,
		LogMaxSizeMB:                      20,
		LogMaxBackups:                     10,
		LogMaxAgeDays:                     7,
		LogCompress:                       true,
		OrchestrationEngine:               defaultExecutionEngine,
		WorkflowEngine:                    defaultExecutionEngine,
		LLMConfigEncryptionKey:            getEnv("LLM_CONFIG_ENCRYPTION_KEY", ""),
		DiscordRequireMention:             defaultDiscordMention,
		DiscordEnableTyping:               defaultDiscordTyping,
		DiscordEnableMessageContentIntent: false,
		TelegramEnableTyping:              defaultTelegramTyping,
	}

	if fileCfg, err := loadFileConfig(); err == nil && fileCfg != nil {
		applyFileConfig(cfg, fileCfg)
	}

	cfg.Port = getEnv("PORT", cfg.Port)
	cfg.DBDriver = getEnv("DB_DRIVER", cfg.DBDriver)
	cfg.DBUrl = getEnv("DATABASE_URL", cfg.DBUrl)
	cfg.APIKey = getEnv("API_KEY", cfg.APIKey)
	cfg.BaseURL = getEnv("API_BASE_URL", cfg.BaseURL)
	cfg.ChatChannel = getEnv("CHAT_CHANNEL", cfg.ChatChannel)
	cfg.ChatModel = getEnv("CHAT_MODEL", cfg.ChatModel)
	cfg.CacheDriver = getEnv("CACHE_DRIVER", cfg.CacheDriver)
	cfg.CacheDefaultTTL = getEnvDuration("CACHE_DEFAULT_TTL", cfg.CacheDefaultTTL)
	cfg.SkillsStorageDir = getEnv("SKILLS_STORAGE_DIR", cfg.SkillsStorageDir)
	cfg.LogFormat = getEnv("LOG_FORMAT", cfg.LogFormat)
	cfg.LogOutput = getEnv("LOG_OUTPUT", cfg.LogOutput)
	cfg.LogFilePath = getEnv("LOG_FILE_PATH", cfg.LogFilePath)
	cfg.LogMaxSizeMB = getEnvInt("LOG_MAX_SIZE_MB", cfg.LogMaxSizeMB)
	cfg.LogMaxBackups = getEnvInt("LOG_MAX_BACKUPS", cfg.LogMaxBackups)
	cfg.LogMaxAgeDays = getEnvInt("LOG_MAX_AGE_DAYS", cfg.LogMaxAgeDays)
	cfg.LogCompress = getEnvBool("LOG_COMPRESS", cfg.LogCompress)
	cfg.OrchestrationEngine = normalizeExecutionEngine(getEnv("ORCHESTRATION_ENGINE", cfg.OrchestrationEngine))
	cfg.WorkflowEngine = normalizeExecutionEngine(getEnv("WORKFLOW_ENGINE", cfg.WorkflowEngine))
	cfg.LLMConfigEncryptionKey = getEnv("LLM_CONFIG_ENCRYPTION_KEY", cfg.LLMConfigEncryptionKey)
	cfg.DiscordToken = getEnv("DISCORD_BOT_TOKEN", "")
	cfg.DiscordRequireMention = getEnvBool("DISCORD_REQUIRE_MENTION", cfg.DiscordRequireMention)
	cfg.DiscordEnableTyping = getEnvBool("DISCORD_ENABLE_TYPING", cfg.DiscordEnableTyping)
	cfg.DiscordEnableMessageContentIntent = getEnvBool("DISCORD_ENABLE_MESSAGE_CONTENT_INTENT", cfg.DiscordEnableMessageContentIntent)
	cfg.DiscordAllowedUserIDs = getEnvIDSet("DISCORD_ALLOWED_USER_IDS")
	cfg.TelegramToken = getEnv("TELEGRAM_BOT_TOKEN", "")
	cfg.TelegramEnableTyping = getEnvBool("TELEGRAM_ENABLE_TYPING", cfg.TelegramEnableTyping)
	cfg.TelegramAllowedUserIDs = getEnvIDSet("TELEGRAM_ALLOWED_USER_IDS")
	cfg.BuildKitHost = getEnv("BUILDKIT_HOST", defaultBuildKitHost)
	cfg.BuildWorkspaceDir = getEnv("BUILD_WORKSPACE_DIR", defaultBuildWorkspaceDir)
	cfg.BuildRegistryHost = getEnv("BUILD_REGISTRY_HOST", "")
	cfg.BuildRegistryUser = getEnv("BUILD_REGISTRY_USER", "")
	cfg.BuildRegistryPass = getEnv("BUILD_REGISTRY_PASS", "")

	if cfg.DBDriver == "sqlite" && cfg.DBUrl == defaultPostgresDBURL {
		cfg.DBUrl = defaultSQLiteDBURL
	}

	return cfg
}

func SaveFile(cfg *Config) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config is nil")
	}

	path, err := Path()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}

	fileCfg := fileConfig{
		Port:                   cfg.Port,
		DBDriver:               cfg.DBDriver,
		DBURL:                  cfg.DBUrl,
		APIKey:                 cfg.APIKey,
		BaseURL:                cfg.BaseURL,
		ChatChannel:            cfg.ChatChannel,
		ChatModel:              cfg.ChatModel,
		OrchestrationEngine:    cfg.OrchestrationEngine,
		WorkflowEngine:         cfg.WorkflowEngine,
		LLMConfigEncryptionKey: cfg.LLMConfigEncryptionKey,
	}

	if existing, err := loadFileConfig(); err == nil && existing != nil && fileCfg.LLMConfigEncryptionKey == "" {
		fileCfg.LLMConfigEncryptionKey = existing.LLMConfigEncryptionKey
	}

	key := strings.TrimSpace(os.Getenv(configEncryptionEnv))
	if key != "" {
		if fileCfg.DBURL != "" {
			encrypted, err := encryptString(key, fileCfg.DBURL)
			if err != nil {
				return "", fmt.Errorf("encrypt database url: %w", err)
			}
			fileCfg.DBURL = encrypted
		}
		if fileCfg.APIKey != "" {
			encrypted, err := encryptString(key, fileCfg.APIKey)
			if err != nil {
				return "", fmt.Errorf("encrypt api key: %w", err)
			}
			fileCfg.APIKey = encrypted
		}
	}

	data, err := json.MarshalIndent(fileCfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write config file: %w", err)
	}
	return path, nil
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", configDirName, configFileName), nil
}

func loadFileConfig() (*fileConfig, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg fileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("decode config file: %w", err)
	}

	key := strings.TrimSpace(os.Getenv(configEncryptionEnv))
	if strings.HasPrefix(cfg.DBURL, encryptedValuePrefix) {
		if key == "" {
			return nil, fmt.Errorf("database_url is encrypted but %s is empty", configEncryptionEnv)
		}
		value, err := decryptString(key, cfg.DBURL)
		if err != nil {
			return nil, fmt.Errorf("decrypt database_url: %w", err)
		}
		cfg.DBURL = value
	}
	if strings.HasPrefix(cfg.APIKey, encryptedValuePrefix) {
		if key == "" {
			return nil, fmt.Errorf("api_key is encrypted but %s is empty", configEncryptionEnv)
		}
		value, err := decryptString(key, cfg.APIKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt api_key: %w", err)
		}
		cfg.APIKey = value
	}

	return &cfg, nil
}

func applyFileConfig(dst *Config, src *fileConfig) {
	if src == nil {
		return
	}
	if src.Port != "" {
		dst.Port = src.Port
	}
	if src.DBDriver != "" {
		dst.DBDriver = src.DBDriver
	}
	if src.DBURL != "" {
		dst.DBUrl = src.DBURL
	}
	if src.APIKey != "" {
		dst.APIKey = src.APIKey
	}
	if src.BaseURL != "" {
		dst.BaseURL = src.BaseURL
	}
	if src.ChatChannel != "" {
		dst.ChatChannel = src.ChatChannel
	}
	if src.ChatModel != "" {
		dst.ChatModel = src.ChatModel
	}
	if src.OrchestrationEngine != "" {
		dst.OrchestrationEngine = normalizeExecutionEngine(src.OrchestrationEngine)
	}
	if src.WorkflowEngine != "" {
		dst.WorkflowEngine = normalizeExecutionEngine(src.WorkflowEngine)
	}
	if src.LLMConfigEncryptionKey != "" {
		dst.LLMConfigEncryptionKey = src.LLMConfigEncryptionKey
	}
}

func normalizeExecutionEngine(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "custom":
		return defaultExecutionEngine
	case "eino":
		return "eino"
	default:
		return defaultExecutionEngine
	}
}

func encryptString(passphrase, plaintext string) (string, error) {
	block, err := aes.NewCipher(deriveKey(passphrase))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return encryptedValuePrefix + base64.StdEncoding.EncodeToString(nonce) + ":" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptString(passphrase, encoded string) (string, error) {
	raw := strings.TrimPrefix(encoded, encryptedValuePrefix)
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid encrypted value format")
	}
	nonce, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("decode nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(deriveKey(passphrase))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func deriveKey(passphrase string) []byte {
	sum := sha256.Sum256([]byte(passphrase))
	return sum[:]
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}

	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvIDSet(key string) map[string]bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}

	out := make(map[string]bool)
	for _, part := range strings.Split(raw, ",") {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		out[id] = true
	}

	if len(out) == 0 {
		return nil
	}
	return out
}
