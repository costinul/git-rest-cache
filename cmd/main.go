package main

import (
	"context"
	"fmt"
	"os"

	"github.com/costinul/git-rest-cache/api"
	"github.com/costinul/git-rest-cache/config"
	"github.com/costinul/git-rest-cache/gitcache"
	"github.com/costinul/git-rest-cache/logger"
	"github.com/costinul/git-rest-cache/provider"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "git-rest-cache",
		Short: "A minimal service that clones & caches Git repos, then exposes them via REST.",
		Run: func(cmd *cobra.Command, args []string) {
			startApp()
		},
	}

	config.InitConfig(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func startApp() {
	cfg := config.GetConfig()

	logger.SetLevel(cfg.LogLevel)

	logger.Info("Starting app...")
	ctx := context.Background()

	gitCache := gitcache.NewGitCache(cfg, ctx, &gitcache.DefaultGitManager{})
	err := gitCache.Start()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to start git cache: %v", err))
		os.Exit(1)
	}

	providerManager := provider.NewDefaultProviderManager()
	api := api.NewCacheAPI(cfg, gitCache, providerManager)
	err = api.Run()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to start API: %v", err))
		os.Exit(1)
	}

	logger.Info("App stopped")
}
