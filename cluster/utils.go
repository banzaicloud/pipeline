package cluster

import (
	"reflect"

	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// isDifferent compares x and y interfaces with deep equal
func isDifferent(x interface{}, y interface{}) error {
	if reflect.DeepEqual(x, y) {
		return pkgErrors.ErrorNotDifferentInterfaces
	}

	return nil
}
