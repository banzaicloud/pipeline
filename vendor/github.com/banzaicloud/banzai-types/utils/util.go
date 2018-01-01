package utils

import "reflect"

// IsDifferent compares x and y interfaces with deep equal
func IsDifferent(x interface{}, y interface{}, logTag string) (bool, string) {
	if reflect.DeepEqual(x, y) {
		msg := "There is no change in data"
		LogInfo(logTag, msg)
		return false, msg
	}
	LogInfo(logTag, "Different interfaces")
	return true, ""
}
