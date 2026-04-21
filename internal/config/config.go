package config

import (
    "github.com/spf13/viper"
)

type Config struct {
    Valkey       ValkeyConfig     `mapstructure:"valkey"`
    WPCLI        WPCLIConfig      `mapstructure:"wp_cli"`
    Paths        PathsConfig      `mapstructure:"paths"`
    Queues       []string         `mapstructure:"queues"`
    Logging      LoggingConfig    `mapstructure:"logging"`
    DomainOwners map[string]string `mapstructure:"domain_owners"`
}

type ValkeyConfig struct {
    Addr     string `mapstructure:"addr"`
    Password string `mapstructure:"password"`
    DB       int    `mapstructure:"db"`
}

type WPCLIConfig struct {
    Path string `mapstructure:"path"`
}

type PathsConfig struct {
    BasePath       string `mapstructure:"base_path"`
    DocumentFolder string `mapstructure:"document_folder"`
}

type LoggingConfig struct {
    Level string `mapstructure:"level"`
    File  string `mapstructure:"file"`
}

func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("/etc/wp-task-runner/")
    viper.AddConfigPath(".")
    viper.AutomaticEnv() // support env vars

    // Set defaults
    viper.SetDefault("valkey.addr", "127.0.0.1:6379")
    viper.SetDefault("wp_cli.path", "/usr/local/bin/wp")
    viper.SetDefault("paths.base_path", "/var/www")
    viper.SetDefault("paths.document_folder", "public")
    viper.SetDefault("queues", []string{"wp_plugin_queue", "wp_theme_queue", "wp_core_queue"})

    if err := viper.ReadInConfig(); err != nil {
        // continue with defaults/env if config file missing
    }

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
