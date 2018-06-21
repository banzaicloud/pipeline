package utils

import (
	"encoding/json"
	"fmt"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"net/http"
)

// NewErr creates new AKS error
// Params:
// 1 - Message string
// 2 - StatusCode int
func NewErr(params ...interface{}) *AKSError {
	switch len(params) {
	case 1:
		return &AKSError{
			Message: params[0].(string),
		}
	case 2:
		return &AKSError{
			Message:    params[0].(string),
			StatusCode: params[1].(int),
		}
	default:
		return &AKSError{
			Message: "unknown error happened",
		}
	}
}

// AKSError describes a client error
type AKSError struct {
	StatusCode int
	Message    string
}

func (e *AKSError) Error() string {
	return e.Message
}

// ToJSON returns the passed item as a pretty-printed JSON string. If any JSON error occurs,
// it returns the empty string.
func ToJSON(v interface{}) (string, error) {
	j, err := json.MarshalIndent(v, "", "  ")
	return string(j), err
}

// NewServicePrincipalTokenFromCredentials creates a new ServicePrincipalToken using values of the
// passed credentials map.
func NewServicePrincipalTokenFromCredentials(c map[string]string, scope string) (*adal.ServicePrincipalToken, error) {
	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, c["AZURE_TENANT_ID"])
	if err != nil {
		panic(err)
	}
	return adal.NewServicePrincipalToken(*oauthConfig, c["AZURE_CLIENT_ID"], c["AZURE_CLIENT_SECRET"], scope)
}

func ensureValueStrings(mapOfInterface map[string]interface{}) map[string]string {
	mapOfStrings := make(map[string]string)
	for key, value := range mapOfInterface {
		mapOfStrings[key] = ensureValueString(value)
	}
	return mapOfStrings
}

func ensureValueString(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// S converts string to string pointer
func S(input string) *string {
	s := input
	return &s
}

// AzureServerError describes an Azure error
type AzureServerError struct {
	Message string `json:"message"`
}

// CreateErrorFromValue creates error from azure response
func CreateErrorFromValue(statusCode int, v []byte) error {
	if statusCode == http.StatusBadRequest {
		ase := AzureServerError{}
		json.Unmarshal([]byte(v), &ase)
		if len(ase.Message) != 0 {
			return NewErr(ase.Message, statusCode)
		}
	}

	type TempError struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	tempError := TempError{}
	json.Unmarshal([]byte(v), &tempError)
	return NewErr(tempError.Error.Message, statusCode)
}

// ToI converts integer pointer to integer
func ToI(pointer *int32) int {
	if pointer != nil {
		return int(*pointer)
	}
	return 0
}

// ToS converts stirng pointer to string
func ToS(pointer *string) string {
	if pointer != nil {
		return *pointer
	}
	return ""
}

// FromBToS converts byte pointer to string
func FromBToS(pointer *[]byte) string {
	if pointer != nil {
		return string(*pointer)
	}
	return ""
}

// AppendIfMissing appends string to a slice if it's not contains it
func AppendIfMissing(slice []string, s string) []string {
	for _, e := range slice {
		if e == s {
			return slice
		}
	}
	return append(slice, s)
}
