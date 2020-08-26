package util

import (
	"fmt"
	"os"
	"strconv"
)

// Gets an integer value from the environment and parses it
func GetIntEnv(name string, varName string) (int, error) {
	value, err := GetEnv(name, varName)
	if err != nil {
		return 0, err
	}

	asInt, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("Environment variable value '%s' invalid for the %s ('%s')",
			value, name, varName)
	}

	return asInt, nil
}

// Gets a string value from the environment and parses it
func GetEnv(name string, varName string) (string, error) {
	value, exists := os.LookupEnv(varName)
	if !exists {
		return "", fmt.Errorf("No environment variable found for the %s ('%s')", name, varName)
	}

	return value, nil
}
