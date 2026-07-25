package common

import (
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
    MasterHost    string            `mapstructure:"master_host"`
    MasterPort    int               `mapstructure:"master_port"`
    UUIDFile      string            `mapstructure:"uuid_file"`
    InstallerArgs map[string]string `mapstructure:"installer_args"` // key: расширение, value: аргументы
}

func LoadConfig(configType, configPath string, config interface{}) error {
    viper.SetConfigType(configType)
    viper.SetConfigFile(configPath)
    viper.AutomaticEnv()
    if err := viper.ReadInConfig(); err != nil {
        return err
    }
    return viper.Unmarshal(config)
}