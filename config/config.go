package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Bot               BotConfig               `yaml:"bot"`
	LLM               LLMConfig               `yaml:"llm"`
	SillyTavern       SillyTavernConfig       `yaml:"silly_tavern"`
	ConversationAgent ConversationAgentConfig `yaml:"conversation_agent"`
	Log               LogConfig               `yaml:"log"`
	Qdrant            QdrantConfig            `yaml:"qdrant"`
	SQLite            SQLiteConfig            `yaml:"sqlite"`
	Embedding         EmbeddingConfig         `yaml:"embedding"`
}

type BotConfig struct {
	WSURL string `yaml:"ws_url"`
}

type LLMConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

type ConversationAgentConfig struct {
	MaxHistory   int    `yaml:"max_history"`
	SystemPrompt string `yaml:"system_prompt"`
}

type LogConfig struct {
	Level      string `yaml:"level"`
	Filename   string `yaml:"filename"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}

type QdrantConfig struct {
	Host           string  `yaml:"host"`
	Port           int     `yaml:"port"`
	Collection     string  `yaml:"collection"`
	TopK           int     `yaml:"top_k"`
	ScoreThreshold float64 `yaml:"score_threshold"`
}

type SQLiteConfig struct {
	DBPath string `yaml:"db_path"`
}

type EmbeddingConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

type SillyTavernConfig struct {
	APIURL string `yaml:"api_url"`
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

func Load(filename string) (*Config, error) {
	// 加载 .env 文件
	err := godotenv.Load()
	if err != nil {
		// 打印警告信息，但不返回错误
		// 因为在生产环境中，我们可能会使用环境变量而不是 .env 文件
		println("Warning: .env file not found, using environment variables")
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	expandEnvVars(&cfg)

	return &cfg, nil
}

func expandEnvVars(cfg *Config) {
	cfg.LLM.APIKey = expandEnv(cfg.LLM.APIKey)
	cfg.LLM.BaseURL = expandEnv(cfg.LLM.BaseURL)
	cfg.LLM.Model = expandEnv(cfg.LLM.Model)
	cfg.SillyTavern.APIURL = expandEnv(cfg.SillyTavern.APIURL)
	cfg.SillyTavern.APIKey = expandEnv(cfg.SillyTavern.APIKey)
	cfg.SillyTavern.Model = expandEnv(cfg.SillyTavern.Model)
	cfg.Embedding.APIKey = expandEnv(cfg.Embedding.APIKey)
	cfg.Embedding.BaseURL = expandEnv(cfg.Embedding.BaseURL)
	cfg.Embedding.Model = expandEnv(cfg.Embedding.Model)
}

func expandEnv(s string) string {
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		key := s[2 : len(s)-1]
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
	}
	return s
}
