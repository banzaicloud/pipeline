package secret

import (
	"fmt"
	"github.com/aokoli/goutils"
)

func Generator(genType string, length int) (string, error) {
	switch genType {
	case "randAlphaNum":
		res, err := goutils.RandomAlphaNumeric(length)
		if err != nil {
			return "", err
		}
		return res, nil
	case "randAlpha":
		res, err := goutils.RandomAlphabetic(length)
		if err != nil {
			return "", err
		}
		return res, nil
	case "randNumeric":
		res, err := goutils.RandomNumeric(length)
		if err != nil {
			return "", err
		}
		return res, nil
	case "randAscii":
		res, err := goutils.RandomAscii(length)
		if err != nil {
			return "", err
		}
		return res, nil
	default:
		return "", fmt.Errorf("unsupported random type: %s", genType)
	}

}
