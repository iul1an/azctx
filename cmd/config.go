package cmd

import (
	"fmt"
	"os"
)

// configFileEnv overrides the config file location. It cannot go through
// viper's AutomaticEnv: it is needed before the config is loaded.
const configFileEnv = "AZCTX_CONFIG_FILE"

// resolveConfigFile returns the config file named by AZCTX_CONFIG_FILE, or
// "" to use the default search path. Naming a file that does not exist is an
// error: silently falling back would run with settings the user did not ask
// for.
func resolveConfigFile() (string, error) {
	path := os.Getenv(configFileEnv)
	if path == "" {
		return "", nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("%s=%s: %w", configFileEnv, path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s=%s is a directory", configFileEnv, path)
	}
	return path, nil
}
