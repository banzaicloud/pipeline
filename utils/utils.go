package utils

import "os"

//GetEnv retrieves ENV variable, fallback if not set
func GetEnv(envKey, defaultValue string) string {
	value, exists := os.LookupEnv(envKey)
	if !exists {
		value = defaultValue
	}
	return value
}

//GetHomeDir retrieves Home on Linux
func GetHomeDir() string {
	//Linux
	return os.Getenv("HOME")
}
