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
	ConversationAgent ConversationAgentConfig `yaml:"conversation_agent"`
	Log               LogConfig               `yaml:"log"`
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