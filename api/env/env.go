package env

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/c2h5oh/datasize"
)

// GetEnv gets a string value from the environment and parses it
func GetEnv(name string, varName string) (string, error) {
	value, exists := os.LookupEnv(varName)
	if !exists {
		return "", fmt.Errorf("No environment variable found for the %s ('%s')", name, varName)
	}

	return value, nil
}

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

// GetBytesEnv gets a datasize.ByteSize value from the environment and parses it
func GetBytesEnv(name string, varName string) (datasize.ByteSize, error) {
	sizeStr, err := GetEnv(name, varName)
	if err != nil {
		return 0, err
	}

	// Parse the bytes into bytes
	var size datasize.ByteSize
	err = size.UnmarshalText([]byte(sizeStr))
	if err != nil {
		return 0, err
	}

	return size, nil
}
