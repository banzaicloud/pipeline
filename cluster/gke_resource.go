package cluster

import (
	"net/http"

	"google.golang.org/api/googleapi"
)

type resourceChecker interface {
	getType() string
	list() ([]string, error)
	isResourceDeleted(string) error
	forceDelete(string) error
}

type resourceCheckers []resourceChecker

const (
	firewall       = "firewall"
	forwardingRule = "forwardingRule"
	targetPool     = "targetPool"
)

// isResourceNotFound transforms an error into googleapi.Error
func isResourceNotFound(err error) error {
	apiError, isOk := err.(*googleapi.Error)
	if isOk {
		if apiError.Code == http.StatusNotFound {
			return nil
		}
	}
	return err
}
