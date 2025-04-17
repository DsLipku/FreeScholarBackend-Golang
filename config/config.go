package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/viper"
)

// Config holds all configuration for our application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	ES       ESConfig       `mapstructure:"elasticsearch"`
	Email    EmailConfig    `mapstructure:"email"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Media    MediaConfig    `mapstructure:"media"`
}

// ServerConfig holds all server related configuration
type ServerConfig struct {
	Port         int    `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	Debug        bool   `mapstructure:"debug"`
}

// DatabaseConfig holds all database related configuration
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Name     string `mapstructure:"name"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

// RedisConfig holds all redis related configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// ESConfig holds all elasticsearch related configuration
type ESConfig struct {
	URL string `mapstructure:"url"`
}

// EmailConfig holds email sending configuration
type EmailConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	UseTLS   bool   `mapstructure:"use_tls"`
}

// JWTConfig holds jwt token configuration
type JWTConfig struct {
	Secret string `mapstructure:"secret_key"`
}

// MediaConfig holds media file configuration
type MediaConfig struct {
	Root string `mapstructure:"root"`
	URL  string `mapstructure:"url"`
}

// Secrets structure for secrets.json
type Secrets struct {
	DatabasePassword string `json:"DATABASE_PASSWORD"`
	RedisPassword    string `json:"REDIS_PASSWORD"`
	EmailUser        string `json:"EMAIL_HOST_USER"`
	EmailPassword    string `json:"EMAIL_HOST_PASSWORD"`
	SecretKey        string `json:"SECRET_KEY"`
}

// LoadConfig loads configuration from config.yaml and secrets.json files
func LoadConfig(configPath string, secretsPath string) (*Config, error) {
	// Load secrets from secrets.json
	secrets, err := loadSecrets(secretsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load secrets: %w", err)
	}

	// Set up viper for YAML config
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	
	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Set defaults for configs not specified in YAML
	setDefaults()

	// Override values with environment variables if they exist
	viper.AutomaticEnv()

	// Inject secrets into configuration
	injectSecrets(secrets)

	// Parse the config into our Config struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// loadSecrets loads sensitive configuration from secrets.json
func loadSecrets(path string) (*Secrets, error) {
	// Read secrets.json
	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open secrets file: %w", err)
	}
	defer jsonFile.Close()

	jsonData, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read secrets file: %w", err)
	}

	// Parse JSON into Secrets struct
	var secrets Secrets
	if err := json.Unmarshal(jsonData, &secrets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secrets: %w", err)
	}

	return &secrets, nil
}

// setDefaults sets default values for configuration
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8000)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", 60)
	viper.SetDefault("server.write_timeout", 60)
	viper.SetDefault("server.debug", false)

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	
	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)

	// Elasticsearch defaults
	viper.SetDefault("elasticsearch.url", "http://localhost:9200")

	// Email defaults
	viper.SetDefault("email.host", "smtp.qq.com")
	viper.SetDefault("email.port", 25)
	viper.SetDefault("email.use_tls", false)

	// Media defaults
	viper.SetDefault("media.root", "./media")
	viper.SetDefault("media.url", "/media/")
}

// injectSecrets injects sensitive configuration from secrets into viper
func injectSecrets(secrets *Secrets) {
	viper.Set("database.password", secrets.DatabasePassword)
	viper.Set("redis.password", secrets.RedisPassword)
	viper.Set("email.user", secrets.EmailUser)
	viper.Set("email.password", secrets.EmailPassword)
	viper.Set("jwt.secret_key", secrets.SecretKey)
}