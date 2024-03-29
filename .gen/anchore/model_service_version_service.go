/*
Anchore Engine API Server

This is the Anchore Engine API. Provides the primary external API for users of the service.

API version: 0.1.20
Contact: nurmi@anchore.com
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package anchore

import (
	"encoding/json"
)

// ServiceVersionService struct for ServiceVersionService
type ServiceVersionService struct {
	// Semantic Version string of the service implementation
	Version *string `json:"version,omitempty"`
}

// NewServiceVersionService instantiates a new ServiceVersionService object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewServiceVersionService() *ServiceVersionService {
	this := ServiceVersionService{}
	return &this
}

// NewServiceVersionServiceWithDefaults instantiates a new ServiceVersionService object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewServiceVersionServiceWithDefaults() *ServiceVersionService {
	this := ServiceVersionService{}
	return &this
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *ServiceVersionService) GetVersion() string {
	if o == nil || o.Version == nil {
		var ret string
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceVersionService) GetVersionOk() (*string, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *ServiceVersionService) HasVersion() bool {
	if o != nil && o.Version != nil {
		return true
	}

	return false
}

// SetVersion gets a reference to the given string and assigns it to the Version field.
func (o *ServiceVersionService) SetVersion(v string) {
	o.Version = &v
}

func (o ServiceVersionService) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.Version != nil {
		toSerialize["version"] = o.Version
	}
	return json.Marshal(toSerialize)
}

type NullableServiceVersionService struct {
	value *ServiceVersionService
	isSet bool
}

func (v NullableServiceVersionService) Get() *ServiceVersionService {
	return v.value
}

func (v *NullableServiceVersionService) Set(val *ServiceVersionService) {
	v.value = val
	v.isSet = true
}

func (v NullableServiceVersionService) IsSet() bool {
	return v.isSet
}

func (v *NullableServiceVersionService) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableServiceVersionService(val *ServiceVersionService) *NullableServiceVersionService {
	return &NullableServiceVersionService{value: val, isSet: true}
}

func (v NullableServiceVersionService) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableServiceVersionService) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


