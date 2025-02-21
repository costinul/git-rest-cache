package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TestConfigPrecedence checks if YAML file, env vars, and CLI flags
// override in the expected order.
func TestConfigPrecedence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFilePath := filepath.Join(tmpDir, "config-test.yaml")
	yamlContent := []byte(`port: 7777
log-level: "warn"
storage-folder: "/from-yaml"
`)

	if err := os.WriteFile(configFilePath, yamlContent, 0666); err != nil {
		t.Fatalf("Failed to write temp config file: %v", err)
	}

	t.Setenv("GIT_REST_CACHE_LOG_LEVEL", "debug")
	t.Setenv("GIT_REST_CACHE_STORAGE_FOLDER", "/from-env")

	viper.Reset()
	viper.SetConfigFile(configFilePath)

	rootCmd := &cobra.Command{
		Use: "test-cmd",
		Run: func(cmd *cobra.Command, args []string) {},
	}

	os.Args = []string{"test-cmd", "--port=9999"}
	rootCmd.SetArgs([]string{"--port=9999"})

	InitConfig(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to execute test-cmd: %v", err)
	}

	c := GetConfig()

	if c.Port != 9999 {
		t.Errorf("Expected port=9999 from CLI override, got %d", c.Port)
	}
	if c.LogLevel != "debug" {
		t.Errorf("Expected log-level=debug from ENV override, got %s", c.LogLevel)
	}
	if c.StorageFolder != "/from-env" {
		t.Errorf("Expected storage-folder=/from-env from ENV override, got %s", c.StorageFolder)
	}
}
