package common

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type MasterConfig struct {
	Host     string   `mapstructure:"host"`
	HTTPPort int      `mapstructure:"http_port"`
	WSPort   int      `mapstructure:"ws_port"`
	DB       DBConfig `mapstructure:"db"`
}

type AgentConfig struct {
	MasterHost     string            `mapstructure:"master_host"`
	MasterPort     int               `mapstructure:"master_port"`
	MasterHTTPPort int               `mapstructure:"master_http_port"`
	MasterScheme   string            `mapstructure:"master_scheme"` // ws for local networks, wss behind TLS proxy
	UUID           string            `mapstructure:"uuid"`
	Token          string            `mapstructure:"token"`
	UUIDFile       string            `mapstructure:"uuid_file"`
	InstallerArgs  map[string]string `mapstructure:"installer_args"` // key: extension, value: arguments
}

func LoadConfig(configType, configPath string, config interface{}) error {
	viper.SetConfigType(configType)
	viper.SetConfigFile(configPath)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	if err := viper.Unmarshal(config); err != nil {
		return err
	}
	switch cfg := config.(type) {
	case *MasterConfig:
		setStringEnv("MASTER_HOST", &cfg.Host)
		if err := setIntEnv("HTTP_PORT", &cfg.HTTPPort); err != nil {
			return err
		}
		if err := setIntEnv("WS_PORT", &cfg.WSPort); err != nil {
			return err
		}
		setStringEnv("DB_HOST", &cfg.DB.Host)
		if err := setIntEnv("DB_PORT", &cfg.DB.Port); err != nil {
			return err
		}
		setStringEnv("DB_USER", &cfg.DB.User)
		setStringEnv("DB_PASSWORD", &cfg.DB.Password)
		setStringEnv("DB_NAME", &cfg.DB.DBName)
		setStringEnv("DB_SSLMODE", &cfg.DB.SSLMode)
	case *AgentConfig:
		setStringEnv("MASTER_HOST", &cfg.MasterHost)
		if err := setIntEnv("MASTER_PORT", &cfg.MasterPort); err != nil {
			return err
		}
		if err := setIntEnv("MASTER_HTTP_PORT", &cfg.MasterHTTPPort); err != nil {
			return err
		}
		setStringEnv("MASTER_SCHEME", &cfg.MasterScheme)
		setStringEnv("AGENT_UUID", &cfg.UUID)
		setStringEnv("AGENT_TOKEN", &cfg.Token)
		setStringEnv("UUID_FILE", &cfg.UUIDFile)
	}
	return nil
}

func setStringEnv(name string, target *string) {
	if value, ok := os.LookupEnv(name); ok && strings.TrimSpace(value) != "" {
		*target = value
	}
}

func setIntEnv(name string, target *int) error {
	value, ok := os.LookupEnv(name)
	if !ok || strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("environment variable %s must be an integer: %w", name, err)
	}
	*target = parsed
	return nil
}
