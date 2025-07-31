package config

import (
	"errors"
	"fmt"
	"math/rand"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Config holds all settings to start the server
type Config struct {
	WebListenAddr string `toml:"web_listen_addr"`
	Secret        string `toml:"secret"`
	BaseURL       string `toml:"base_url"`
	Debug         bool   `toml:"debug"`
	MailFrom      string `toml:"mail_from"`
}

// defaultConfig returns a config object with default values.
func defaultConfig() Config {
	return Config{
		WebListenAddr: "localhost:8080",
		Secret:        CreatePassword(32),
		BaseURL:       "http://localhost:8080",
		Debug:         true,
		MailFrom:      "mail@example.com",
	}
}

// LoadConfig loads the config from a toml file.
func LoadConfig(file string) (Config, error) {
	c := defaultConfig()

	f, err := os.Open(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// If an error happens, return the error and the default config. The
			// caller can deside, if he wants to use the config even when the
			// default could not be saved.
			err := saveConfig(file, c)
			return c, err
		}
		return Config{}, fmt.Errorf("open config file: %w", err)
	}

	if err := toml.NewDecoder(f).Decode(&c); err != nil {
		return Config{}, fmt.Errorf("reading config: %w", err)
	}
	return c, nil
}

func saveConfig(file string, config Config) (err error) {
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}
	defer func() {
		closeErr := f.Close()
		if closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if err := toml.NewEncoder(f).Encode(config); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// CreatePassword creates a random password string.
func CreatePassword(length int) string {
	chars := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

	result := make([]byte, length)
	for i := range length {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
