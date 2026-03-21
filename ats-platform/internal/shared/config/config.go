// internal/shared/config/config.go
package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Elastic  ElasticConfig
	Minio    MinioConfig
}

type ServerConfig struct {
	Name         string        `mapstructure:"name"`
	HTTPPort     int           `mapstructure:"http_port"`
	GRPCPort     int           `mapstructure:"grpc_port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type ElasticConfig struct {
	Hosts    []string `mapstructure:"hosts"`
	Username string   `mapstructure:"username"`
	Password string   `mapstructure:"password"`
}

type MinioConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	UseSSL    bool   `mapstructure:"use_ssl"`
	Bucket    string `mapstructure:"bucket"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file
	v.SetConfigFile(configPath)

	// Read from environment variables
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
