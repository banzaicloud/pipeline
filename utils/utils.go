// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

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

// EncodeStringToBase64 first checks if the string is encoded if yes returns it if no than encodes it.
func EncodeStringToBase64(s string) string {
	if _, err := base64.StdEncoding.DecodeString(s); err != nil {
		return base64.StdEncoding.EncodeToString([]byte(s))
	}
	return s
}

// ConvertSecondsToTime returns string format of seconds
func ConvertSecondsToTime(t time.Time) string {
	return t.Format(time.RFC3339)
}
