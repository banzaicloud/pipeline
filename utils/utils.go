package utils

import (
	"encoding/json"
	"github.com/banzaicloud/banzai-types/constants"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
)

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

//NopHandler is an empty handler to help net/http -> Gin conversions
type NopHandler struct{}

func (h NopHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {}

//WriteToFile write the []byte to the given file
func WriteToFile(data []byte, file string) error {
	if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
		return err
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return ioutil.WriteFile(file, data, 0644)
	}

	tmpfi, err := ioutil.TempFile(filepath.Dir(file), "file.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfi.Name())

	if err = ioutil.WriteFile(tmpfi.Name(), data, 0644); err != nil {
		return err
	}

	if err = tmpfi.Close(); err != nil {
		return err
	}

	if err = os.Remove(file); err != nil {
		return err
	}

	err = os.Rename(tmpfi.Name(), file)
	return err
}

// IsDifferent compares x and y interfaces with deep equal
func IsDifferent(x interface{}, y interface{}) error {
	if reflect.DeepEqual(x, y) {
		return constants.ErrorNotDifferentInterfaces
	}

	return nil
}

// ConvertJson2Map converts []byte to map[string]string
func ConvertJson2Map(js []byte) (map[string]string, error) {
	var result map[string]string
	err := json.Unmarshal(js, &result)
	return result, err
}

// Contains checks slice contains `s` string
func Contains(slice []string, s string) bool {
	for _, sl := range slice {
		if sl == s {
			return true
		}
	}
	return false
}

// ValidateCloudType validates if the passed cloudType is supported.
// If a not supported cloud type is passed in than returns ErrorNotSupportedCloudType otherwise nil
func ValidateCloudType(cloudType string) error {
	switch cloudType {
	case constants.Amazon:
		return nil
	case constants.Google:
		return nil
	case constants.Azure:
		return nil
	default:
		return constants.ErrorNotSupportedCloudType
	}
}
