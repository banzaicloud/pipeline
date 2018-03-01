package utils

import (
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
