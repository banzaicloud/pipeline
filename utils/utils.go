package utils

import "os"

func GetEnv(envKey, defaultValue string) string {
	value, exists := os.LookupEnv(envKey)
	if !exists {
		value = defaultValue
	}
	return value
}

func GetHomeDir() string {
	//Linux
	return os.Getenv("HOME")
}
