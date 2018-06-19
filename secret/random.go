package secret

import (
	"fmt"
	"github.com/aokoli/goutils"
)

//RandomString creates a random string whose length is the number of characters specified.
func RandomString(genType string, length int) (res string, err error) {
	switch genType {
	case "randAlphaNum":
		res, err = goutils.RandomAlphaNumeric(length)
	case "randAlpha":
		res, err = goutils.RandomAlphabetic(length)
	case "randNumeric":
		res, err = goutils.RandomNumeric(length)
	case "randAscii":
		res, err = goutils.RandomAscii(length)
	default:
		return res, fmt.Errorf("unsupported random type: %s", genType)
	}
	return

}
