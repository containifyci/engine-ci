package utils

import (
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

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
	if strings.HasPrefix(env, "env:") {
		return Getenv(strings.TrimPrefix(env, "env:"), envType)
	}

	if strings.HasPrefix(env, "cmd:") {
		cmd := strings.TrimPrefix(env, "cmd:")
		env2, err := RunCommand(cmd)
		if err != nil {
			slog.Error("Error running command", "error", err)
			os.Exit(1)
		}
		slog.Info("Retrieved environment variable from command", "command", cmd, "key", key)
		return *env2
	}
	return env
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
