package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
)


// Create new AKS error
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
			Message: params[0].(string),
			StatusCode: params[1].(int),
		}
	default:
		return &AKSError{
			Message: "unknown error happend",
		}
	}
}

type AKSError struct {
	StatusCode int
	Message string
}

func (e *AKSError) Error() string{
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

func ReadPubRSA(filename string) string {
	b, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/" + filename)
	if err != nil {
		fmt.Print(err)
	}
	return string(b)
}

func S(input string) *string {
	s := input
	return &s
}

type AzureServerError struct {
	Message string `json:"message"`
}

func CreateErrorFromValue(statusCode int, v []byte) error {
	if statusCode == banzaiConstants.BadRequest {
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
