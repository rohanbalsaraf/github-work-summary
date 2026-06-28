package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	defaultProfile   = "default"
	keyActiveProfile = "active_profile"
	keyAIProvider    = "ai_provider"
)

func initConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	viper.AddConfigPath(home)
	viper.SetConfigName(".gws")
	viper.SetConfigType("yaml")

	viper.SetDefault(keyActiveProfile, defaultProfile)
	viper.SetDefault(keyAIProvider, "gemini")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// Create empty config file if it doesn't exist
		configPath := filepath.Join(home, ".gws.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			f, _ := os.Create(configPath)
			if f != nil { _ = f.Close() }
		}
	}

	// Migration: Move top-level repositories/branches to default profile if they exist
	migrateToProfiles()
}

func migrateToProfiles() {
	repos := viper.GetStringSlice("repositories")
	branches := viper.GetStringSlice("branches")

	modified := false
	if len(repos) > 0 {
		viper.Set(getProfileKey(defaultProfile, "repositories"), repos)
		viper.Set("repositories", nil)
		modified = true
	}
	if len(branches) > 0 {
		viper.Set(getProfileKey(defaultProfile, "branches"), branches)
		viper.Set("branches", nil)
		modified = true
	}

	if modified {
		_ = saveConfig()
	}
}

func getActiveProfileName() string {
	// 1. Check flag override (will be set in root.go)
	if rootProfileOverride != "" {
		return rootProfileOverride
	}
	// 2. Check config
	name := viper.GetString(keyActiveProfile)
	if name == "" {
		return defaultProfile
	}
	return name
}

func getProfileKey(profileName, suffix string) string {
	return fmt.Sprintf("profiles.%s.%s", profileName, suffix)
}

func saveConfig() error {
	return viper.WriteConfig()
}
