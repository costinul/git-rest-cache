package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Port              int           `mapstructure:"port"`
	LogLevel          string        `mapstructure:"log-level"`
	StorageFolder     string        `mapstructure:"storage-folder"`
	RepoTTL           time.Duration `mapstructure:"repo-ttl"`
	TokenTTL          time.Duration `mapstructure:"token-ttl"`
	RepoCheckInterval time.Duration `mapstructure:"repo-check-interval"`
}

var cfg Config

func GetConfig() *Config {
	return &cfg
}

func InitConfig(cmd *cobra.Command) {
	viper.SetDefault("port", 8080)
	viper.SetDefault("log-level", "info")
	viper.SetDefault("storage-folder", "./cached-repos")
	viper.SetDefault("repo-ttl", "24h")
	viper.SetDefault("token-ttl", "24h")
	viper.SetDefault("repo-check-interval", "5m")

	if viper.ConfigFileUsed() == "" {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("../")
		viper.AddConfigPath("../config")
		viper.AddConfigPath("./config")
	}

	if err := viper.ReadInConfig(); err == nil {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	} else {
		fmt.Printf("Config file not loaded: %v\n", err)
	}

	viper.SetEnvPrefix("GIT_REST_CACHE")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	cmd.PersistentFlags().Int("port", 8080, "HTTP port to listen on")
	cmd.PersistentFlags().String("log-level", "info", "Logging level (debug, info, warn, error)")
	cmd.PersistentFlags().String("storage-folder", "./cached-repos", "Folder to store cached repos")
	cmd.PersistentFlags().String("repo-ttl", "24h", "Time a repo remains in cache since last access")
	cmd.PersistentFlags().String("token-ttl", "24h", "Time a token remains valid in memory after last use")
	cmd.PersistentFlags().String("repo-check-interval", "5m", "Interval to fetch changes in cached repos")

	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		_ = viper.BindPFlag(f.Name, f)
	})

	cmd.ParseFlags(os.Args[1:])

	if err := viper.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("error unmarshaling config: %w", err))
	}
}
