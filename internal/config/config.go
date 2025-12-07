// Package config provides configuration loading and management for the application.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Logger   LoggerConfig   `yaml:"logger"`
	JWT      JWTConfig      `yaml:"jwt"`
	UserAPI  UserAPIConfig  `yaml:"user_api"`
	CORS     CORSConfig     `yaml:"cors"`
	Redis    RedisConfig    `mapstructure:"redis" yaml:"redis"` // ← Redis 추가
	S3       S3Config       `yaml:"s3"`                          // ← S3 추가
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port            string        `yaml:"port"`
	Mode            string        `yaml:"mode"` // debug, release
	BasePath        string        `yaml:"base_path"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            string        `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	DBName          string        `yaml:"dbname"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level      string `yaml:"level"` // debug, info, warn, error
	OutputPath string `yaml:"output_path"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret     string        `yaml:"secret"`
	ExpireTime time.Duration `yaml:"expire_time"`
}

// UserAPIConfig holds User API configuration
type UserAPIConfig struct {
	BaseURL string        `yaml:"base_url"`
	Timeout time.Duration `yaml:"timeout"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins string `yaml:"allowed_origins"`
}

type RedisConfig struct {
	Password string `mapstructure:"password" yaml:"password"`
	DB       int    `mapstructure:"db" yaml:"db"`
	TLS      bool   `mapstructure:"tls" yaml:"tls"`
	URL      string `mapstructure:"url" yaml:"url"` // redis:// 형식 지원
}

// S3Config holds S3 configuration
type S3Config struct {
	Bucket    string `yaml:"bucket"`
	Region    string `yaml:"region"`
	AccessKey string `yaml:"access_key"` // MinIO용만 필요 (선택적)
	SecretKey string `yaml:"secret_key"` // MinIO용만 필요 (선택적)
	Endpoint  string `yaml:"endpoint"`   // 로컬 MinIO용 (선택적)
}

// Load loads configuration from file and environment variables
// If config file doesn't exist, loads from environment variables only
func Load(configPath string) (*Config, error) {
	var cfg Config

	// Try to read config file (optional)
	data, err := os.ReadFile(configPath)
	if err == nil {
		// Parse YAML if file exists
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	} else {
		// Config file doesn't exist - use defaults
		fmt.Fprintf(os.Stderr, "Config file not found, using environment variables and defaults\n")
		cfg = getDefaultConfig()
	}

	// Override with environment variables
	cfg.overrideFromEnv()

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// getDefaultConfig returns default configuration values
func getDefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Port:            "8000",
			Mode:            "debug",
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			ShutdownTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            "5432",
			User:            "postgres",
			Password:        "",
			DBName:          "project_board",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Logger: LoggerConfig{
			Level:      "info",
			OutputPath: "stdout",
		},
		JWT: JWTConfig{
			Secret:     "",
			ExpireTime: 24 * time.Hour,
		},
		UserAPI: UserAPIConfig{
			BaseURL: "http://localhost:8080",
			Timeout: 5 * time.Second,
		},
		CORS: CORSConfig{
			AllowedOrigins: "*",
		},
	}
}

// overrideFromEnv overrides configuration with environment variables
// Supports both original format (wealist-project) and current format
// Original format takes precedence when both are provided
func (c *Config) overrideFromEnv() {
	// Server
	if port := os.Getenv("SERVER_PORT"); port != "" {
		c.Server.Port = port
	}

	// ENV alias for SERVER_MODE (original format takes precedence)
	// Maps: dev→debug, prod→release
	if env := os.Getenv("ENV"); env != "" {
		switch env {
		case "dev":
			c.Server.Mode = "debug"
		case "prod":
			c.Server.Mode = "release"
		default:
			c.Server.Mode = env
		}
	}
	// Current format can override if ENV not set
	if mode := os.Getenv("SERVER_MODE"); mode != "" && os.Getenv("ENV") == "" {
		c.Server.Mode = mode
	}

	// Base path for ALB routing
	if basePath := os.Getenv("SERVER_BASE_PATH"); basePath != "" {
		c.Server.BasePath = basePath
	}

	// Database - DATABASE_URL takes precedence (original format)
	// Parse DATABASE_URL first if provided
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		host, port, user, password, dbname, err := parseDatabaseURL(databaseURL)
		if err != nil {
			// Log error but continue - individual variables might be set
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse DATABASE_URL: %v\n", err)
		} else {
			// Populate database fields from parsed URL
			c.Database.Host = host
			c.Database.Port = port
			c.Database.User = user
			c.Database.Password = password
			c.Database.DBName = dbname
		}
	}

	// Individual DB_* variables can override DATABASE_URL if provided
	if host := os.Getenv("DB_HOST"); host != "" {
		c.Database.Host = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		c.Database.Port = port
	}
	if user := os.Getenv("DB_USER"); user != "" {
		c.Database.User = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		c.Database.Password = password
	}
	if dbname := os.Getenv("DB_NAME"); dbname != "" {
		c.Database.DBName = dbname
	}

	// Logger
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.Logger.Level = level
	}
	if outputPath := os.Getenv("LOG_OUTPUT_PATH"); outputPath != "" {
		c.Logger.OutputPath = outputPath
	}

	// JWT - SECRET_KEY alias (original format takes precedence)
	if secret := os.Getenv("SECRET_KEY"); secret != "" {
		c.JWT.Secret = secret
	}
	// Current format can override if SECRET_KEY not set
	if secret := os.Getenv("JWT_SECRET"); secret != "" && os.Getenv("SECRET_KEY") == "" {
		c.JWT.Secret = secret
	}

	// User API - USER_SERVICE_URL alias (original format takes precedence)
	if baseURL := os.Getenv("USER_SERVICE_URL"); baseURL != "" {
		c.UserAPI.BaseURL = baseURL
	}
	// Current format can override if USER_SERVICE_URL not set
	if baseURL := os.Getenv("USER_API_BASE_URL"); baseURL != "" && os.Getenv("USER_SERVICE_URL") == "" {
		c.UserAPI.BaseURL = baseURL
	}
	if timeout := os.Getenv("USER_API_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			c.UserAPI.Timeout = d
		}
	}

	// CORS - CORS_ORIGINS alias (original format takes precedence)
	if origins := os.Getenv("CORS_ORIGINS"); origins != "" {
		c.CORS.AllowedOrigins = origins
	}
	// Current format can override if CORS_ORIGINS not set
	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" && os.Getenv("CORS_ORIGINS") == "" {
		c.CORS.AllowedOrigins = origins
	}
	// Redis 환경변수 오버라이드 (이게 핵심!)
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		c.Redis.Password = redisPassword
	}
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		c.Redis.URL = redisURL
	}

	// S3 환경변수 오버라이드
	if s3Bucket := os.Getenv("S3_BUCKET"); s3Bucket != "" {
		c.S3.Bucket = s3Bucket
	}
	if s3Region := os.Getenv("S3_REGION"); s3Region != "" {
		c.S3.Region = s3Region
	}
	// AccessKey/SecretKey는 MinIO 사용 시에만 필요 (선택적)
	if s3AccessKey := os.Getenv("S3_ACCESS_KEY"); s3AccessKey != "" {
		c.S3.AccessKey = s3AccessKey
	}
	if s3SecretKey := os.Getenv("S3_SECRET_KEY"); s3SecretKey != "" {
		c.S3.SecretKey = s3SecretKey
	}
	if s3Endpoint := os.Getenv("S3_ENDPOINT"); s3Endpoint != "" {
		c.S3.Endpoint = s3Endpoint
	}
}

// validate validates the configuration
func (c *Config) validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Port == "" {
		return fmt.Errorf("database port is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("jwt secret is required")
	}
	if c.UserAPI.BaseURL == "" {
		return fmt.Errorf("user api base url is required")
	}
	if c.UserAPI.Timeout == 0 {
		return fmt.Errorf("user api timeout is required")
	}

	// Validate and normalize User API Base URL
	if err := c.validateUserAPIBaseURL(); err != nil {
		return err
	}

	return nil
}

// validateUserAPIBaseURL validates and normalizes the User API base URL
func (c *Config) validateUserAPIBaseURL() error {
	baseURL := c.UserAPI.BaseURL

	// Check for trailing slash and remove it
	if strings.HasSuffix(baseURL, "/") {
		fmt.Fprintf(os.Stderr, "Warning: User API base URL has trailing slash, removing it: %s\n", baseURL)
		c.UserAPI.BaseURL = strings.TrimSuffix(baseURL, "/")
		baseURL = c.UserAPI.BaseURL
	}

	// Parse URL to validate format
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid user api base url format '%s': %w", baseURL, err)
	}

	// Validate scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("user api base url must use http or https scheme, got: %s", parsedURL.Scheme)
	}

	// Validate host is present
	if parsedURL.Host == "" {
		return fmt.Errorf("user api base url missing host: %s", baseURL)
	}

	// Log configuration details for debugging
	fmt.Fprintf(os.Stderr, "User API Configuration validated:\n")
	fmt.Fprintf(os.Stderr, "  - Base URL: %s\n", c.UserAPI.BaseURL)
	fmt.Fprintf(os.Stderr, "  - Scheme: %s\n", parsedURL.Scheme)
	fmt.Fprintf(os.Stderr, "  - Host: %s\n", parsedURL.Host)
	fmt.Fprintf(os.Stderr, "  - Timeout: %s\n", c.UserAPI.Timeout)

	// Check environment variable sources
	if userServiceURL := os.Getenv("USER_SERVICE_URL"); userServiceURL != "" {
		fmt.Fprintf(os.Stderr, "  - Source: USER_SERVICE_URL environment variable\n")
	} else if userAPIBaseURL := os.Getenv("USER_API_BASE_URL"); userAPIBaseURL != "" {
		fmt.Fprintf(os.Stderr, "  - Source: USER_API_BASE_URL environment variable\n")
	} else {
		fmt.Fprintf(os.Stderr, "  - Source: config.yaml file\n")
	}

	return nil
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		c.Host, c.Port, c.User, c.Password, c.DBName,
	)
}

// parseDatabaseURL parses a PostgreSQL connection URL and extracts connection components
// Expected format: postgresql://user:password@host:port/dbname?sslmode=disable
func parseDatabaseURL(databaseURL string) (host, port, user, password, dbname string, err error) {
	if databaseURL == "" {
		return "", "", "", "", "", fmt.Errorf("DATABASE_URL is empty")
	}

	// Parse the URL
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("invalid DATABASE_URL format: %w\nExpected format: postgresql://user:password@host:port/dbname?sslmode=disable", err)
	}

	// Validate scheme
	if u.Scheme != "postgresql" && u.Scheme != "postgres" {
		return "", "", "", "", "", fmt.Errorf("invalid DATABASE_URL scheme '%s': must be 'postgresql' or 'postgres'\nExpected format: postgresql://user:password@host:port/dbname?sslmode=disable", u.Scheme)
	}

	// Extract user and password
	if u.User == nil {
		return "", "", "", "", "", fmt.Errorf("DATABASE_URL missing user credentials\nExpected format: postgresql://user:password@host:port/dbname?sslmode=disable")
	}
	user = u.User.Username()
	password, _ = u.User.Password()

	// Extract host and port
	if u.Host == "" {
		return "", "", "", "", "", fmt.Errorf("DATABASE_URL missing host\nExpected format: postgresql://user:password@host:port/dbname?sslmode=disable")
	}

	// Split host and port
	hostPort := u.Host
	if strings.Contains(hostPort, ":") {
		parts := strings.Split(hostPort, ":")
		host = parts[0]
		port = parts[1]
	} else {
		host = hostPort
		port = "5432" // Default PostgreSQL port
	}

	// Extract database name
	dbname = strings.TrimPrefix(u.Path, "/")
	if dbname == "" {
		return "", "", "", "", "", fmt.Errorf("DATABASE_URL missing database name\nExpected format: postgresql://user:password@host:port/dbname?sslmode=disable")
	}

	return host, port, user, password, dbname, nil
}
