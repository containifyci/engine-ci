package utils

import (
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

func GetEnvWithDefault(key string, def func() string ) string {
	env := os.Getenv(key)
	if env == "" && def != nil {
		return def()
	}
	return env
}

func Getenv(key string, envType string) string {
	switch envType {
	case "local":
		return os.Getenv(key + "_LOCAL")
	case "build":
		return os.Getenv(key)
	case "production":
		return os.Getenv(key)
	}
	return ""
}

func GetEnvs(key []string, envType string) string {
	for _, k := range key {
		if v := Getenv(k, envType); v != "" {
			return v
		}
	}
	slog.Warn("No environment variable found", "keys", key)
	return ""
}



func GetEnv(key string, envType string) string {
	env := Getenv(key, envType)
	return GetValue(env, envType)
}

func GetValue(value string, envType string) string {
	if strings.HasPrefix(value, "env:") {
		return Getenv(strings.TrimPrefix(value, "env:"), envType)
	}

	if strings.HasPrefix(value, "cmd:") {
		cmd := strings.TrimPrefix(value, "cmd:")
		env2, err := RunCommand(cmd)
		if err != nil {
			slog.Error("Error running command", "error", err, "command", cmd)
			os.Exit(1)
		}
		slog.Info("Retrieved environment variable from command", "command", cmd)
		return *env2
	}
	return value
}

func RunCommand(cmd string) (*string, error) {
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		slog.Error("Error running command", "error", err, "command", cmd)
		return nil, err
	}
	res := strings.TrimSuffix(string(out), "\n")
	return &res, nil
}
