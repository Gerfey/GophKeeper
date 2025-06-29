package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	DefaultJWTExpirationHours = 24
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Auth       AuthConfig
	Encryption EncryptionConfig
	Client     ClientConfig
}

type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	TLSCertFile  string
	TLSKeyFile   string
}

type EncryptionConfig struct {
	KeySize int
}

type DatabaseConfig struct {
	Driver   string
	Host     string
	Port     string
	Username string
	Password string
	DBName   string
	SSLMode  string
}

type AuthConfig struct {
	JWTSecret        string
	JWTExpirationHrs int
}

type ClientConfig struct {
	ServerURL string
	Token     string
	Username  string
}

func LoadConfig(path string) (*Config, error) {
	_ = godotenv.Load()

	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.read_timeout", "10s")
	viper.SetDefault("server.write_timeout", "10s")

	viper.SetDefault("database.driver", "postgres")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "5432")
	viper.SetDefault("database.sslmode", "disable")

	viper.SetDefault("auth.jwt_signing_key", "supersecretkey")
	viper.SetDefault("auth.jwt_expiration_hrs", DefaultJWTExpirationHours)

	viper.SetDefault("client.server_url", "https://localhost:8080")

	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("ошибка чтения конфигурационного файла: %w", err)
		}
	}

	var config Config

	readTimeout, err := time.ParseDuration(viper.GetString("server.read_timeout"))
	if err != nil {
		return nil, fmt.Errorf("неверный формат read_timeout: %w", err)
	}

	writeTimeout, err := time.ParseDuration(viper.GetString("server.write_timeout"))
	if err != nil {
		return nil, fmt.Errorf("неверный формат write_timeout: %w", err)
	}

	config.Server = ServerConfig{
		Host:         viper.GetString("server.host"),
		Port:         viper.GetString("server.port"),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		TLSCertFile:  viper.GetString("server.tls_cert_file"),
		TLSKeyFile:   viper.GetString("server.tls_key_file"),
	}

	config.Database = DatabaseConfig{
		Driver:   viper.GetString("database.driver"),
		Host:     viper.GetString("database.host"),
		Port:     viper.GetString("database.port"),
		Username: viper.GetString("database.username"),
		Password: viper.GetString("database.password"),
		DBName:   viper.GetString("database.dbname"),
		SSLMode:  viper.GetString("database.sslmode"),
	}

	config.Auth = AuthConfig{
		JWTSecret:        viper.GetString("auth.jwt_signing_key"),
		JWTExpirationHrs: viper.GetInt("auth.jwt_expiration_hrs"),
	}

	config.Encryption = EncryptionConfig{
		KeySize: viper.GetInt("encryption.key_size"),
	}

	config.Client = ClientConfig{
		ServerURL: viper.GetString("client.server_url"),
		Token:     "",
		Username:  "",
	}

	return &config, nil
}

func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s://%s:%s@%s:%s/%s?sslmode=%s",
		c.Driver, c.Username, c.Password, c.Host, c.Port, c.DBName, c.SSLMode)
}
