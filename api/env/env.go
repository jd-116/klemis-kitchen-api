package env

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// GetIntEnv gets an integer value from the environment and parses it
func GetIntEnv(name string, varName string) (int, error) {
	value, err := GetEnv(name, varName)
	if err != nil {
		return 0, err
	}

	asInt, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("Environment variable value '%s' invalid for the %s ('%s'):\n%s",
			value, name, varName, err)
	}

	return asInt, nil
}

// GetDurationEnv gets a duration value from the environment and parses it
func GetDurationEnv(name string, varName string) (time.Duration, error) {
	value, err := GetEnv(name, varName)
	if err != nil {
		return 0, err
	}

	asDuration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("Environment variable value '%s' invalid for the %s ('%s'):\n%s",
			value, name, varName, err)
	}

	return asDuration, nil
}

// GetEnv gets a string value from the environment and parses it
func GetEnv(name string, varName string) (string, error) {
	value, exists := os.LookupEnv(varName)
	if !exists {
		return "", fmt.Errorf("No environment variable found for the %s ('%s')", name, varName)
	}

	return value, nil
}
