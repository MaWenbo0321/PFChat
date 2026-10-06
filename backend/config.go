package main

import (
	"fmt"
	"os"
	"strings"
)

func envValue(name string) string {
	return strings.TrimSpace(os.Getenv(name))
}

func envValueOrDefault(name, fallback string) string {
	if value := envValue(name); value != "" {
		return value
	}
	return fallback
}

func requireEnv(name string) (string, error) {
	value := envValue(name)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", name)
	}
	return value, nil
}

func validateRuntimeConfig() error {
	for _, name := range []string{
		"DASHSCOPE_API_KEY",
		"JWT_SECRET",
		"MYSQL_DATABASE",
		"MYSQL_USER",
		"MYSQL_PASSWORD",
	} {
		if _, err := requireEnv(name); err != nil {
			return err
		}
	}
	if len([]byte(envValue("JWT_SECRET"))) < 32 {
		return fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	return nil
}
