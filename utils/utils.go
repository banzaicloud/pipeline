package utils

import "os"

//Retrieve ENV variable, fallback if not set
func GetEnv(envKey, defaultValue string) string {
	value, exists := os.LookupEnv(envKey)
	if !exists {
		value = defaultValue
	}
	return value
}

//Retrieve Home on Linux
func GetHomeDir() string {
	//Linux
	return os.Getenv("HOME")
}
