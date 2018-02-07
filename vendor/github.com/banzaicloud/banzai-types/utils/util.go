package utils

import (
	"reflect"
	"errors"
)

// IsDifferent compares x and y interfaces with deep equal
func IsDifferent(x interface{}, y interface{}, logTag string) error {
	if reflect.DeepEqual(x, y) {
		msg := "There is no change in data"
		LogInfo(logTag, msg)
		return errors.New(msg)
	}
	LogInfo(logTag, "Different interfaces")
	return nil
}
