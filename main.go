// Main file

package main

import (
	"encoding/json"
	"github.com/tailscale/hujson"
	"os"
	"path/filepath"
)

func main() {
	config, err := ParseConfig()
	if err != nil {
		Logf("Error: Parsing config failed: %v", ErrorToStr(err))
		os.Exit(1)
	}

	bot, err := NewBot(config)
	if err != nil {
		Logf("Error: Bot initialization failed: %v", ErrorToStr(err))
		os.Exit(1)
	}

	err = bot.Run()
	if err != nil {
		Logf("Error: Bot run failed: %v", ErrorToStr(err))
		os.Exit(1)
	}
}

func ParseConfig() (*Config, error) {
	jsonBytes, err := os.ReadFile(filepath.FromSlash("./config.jsonc"))
	if err != nil {
		return nil, WrapError(err)
	}

	config := &Config{}

	ast, err := hujson.Parse(jsonBytes)
	if err != nil {
		return nil, WrapError(err)
	}
	ast.Standardize()
	jsonBytes = ast.Pack()

	err = json.Unmarshal(jsonBytes, config)
	if err != nil {
		return nil, WrapError(err)
	}

	return config, nil
}
