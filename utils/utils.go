package utils

import (
	"os"
	"strings"
)

func Ptr[T any](v T) *T {
	return &v
}

func GetEnv(key string) string {
	// 1. Normal env
	if val := os.Getenv(key); val != "" {
		return val
	}

	// 2. File-based secret
	if file := os.Getenv(key + "_FILE"); file != "" {
		data, err := os.ReadFile(file)
		if err == nil {
			return strings.TrimSpace(string(data))
		} else {
			panic(err)
		}
	}

	return ""
}
