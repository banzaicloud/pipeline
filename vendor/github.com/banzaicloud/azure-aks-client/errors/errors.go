package errors

import "errors"

// constants for errors
var (
	ErrClusterNameRegexp  = errors.New("only numbers, lowercase letters and underscores are allowed under name property. In addition, the value cannot end with an underscore, and must also be less than 32 characters long")
	ErrClusterNameEmpty   = errors.New("the name should not be empty")
	ErrClusterNameTooLong = errors.New("cluster name is greater than or equal 32")
	ErrClusterStageFailed = errors.New("cluster stage is 'Failed'")
	ErrNoInfrastructureRG = errors.New("no infrastructure resource group found")
)
