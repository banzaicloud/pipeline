package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"github.com/banzaicloud/banzai-types/constants"
)

func Convert2Json(i interface{}) string {
	var res string
	jsonResponse, err := json.Marshal(i)
	if err != nil {
		LogInfo(constants.TagFormat, "Convert to json failed: ", err.Error())
		res = fmt.Sprintf("%#v", i)
	} else {
		res = fmt.Sprintf("%s", jsonResponse)
	}
	return res
}

// convertString2Uint converts a string to uint
func ConvertString2Uint(s string) uint {
	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		LogInfo(constants.TagFormat, "Convert string to uint failed: ", err.Error())
		panic(err)
	}
	return uint(i)
}
