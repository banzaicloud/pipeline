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

// RegistryTagSource An image reference using a tag in a registry, this is the most common source type.
type RegistryTagSource struct {
	// A docker pull string (e.g. docker.io/nginx:latest, or docker.io/nginx@sha256:abd) to retrieve the image
	Pullstring string `json:"pullstring"`
	// Base64 encoded content of the dockerfile used to build the image, if available.
	Dockerfile *string `json:"dockerfile,omitempty"`
}

// NewRegistryTagSource instantiates a new RegistryTagSource object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewRegistryTagSource(pullstring string) *RegistryTagSource {
	this := RegistryTagSource{}
	this.Pullstring = pullstring
	return &this
}

// NewRegistryTagSourceWithDefaults instantiates a new RegistryTagSource object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewRegistryTagSourceWithDefaults() *RegistryTagSource {
	this := RegistryTagSource{}
	return &this
}

// GetPullstring returns the Pullstring field value
func (o *RegistryTagSource) GetPullstring() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Pullstring
}

// GetPullstringOk returns a tuple with the Pullstring field value
// and a boolean to check if the value has been set.
func (o *RegistryTagSource) GetPullstringOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Pullstring, true
}

// SetPullstring sets field value
func (o *RegistryTagSource) SetPullstring(v string) {
	o.Pullstring = v
}

// GetDockerfile returns the Dockerfile field value if set, zero value otherwise.
func (o *RegistryTagSource) GetDockerfile() string {
	if o == nil || o.Dockerfile == nil {
		var ret string
		return ret
	}
	return *o.Dockerfile
}

// GetDockerfileOk returns a tuple with the Dockerfile field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RegistryTagSource) GetDockerfileOk() (*string, bool) {
	if o == nil || o.Dockerfile == nil {
		return nil, false
	}
	return o.Dockerfile, true
}

// HasDockerfile returns a boolean if a field has been set.
func (o *RegistryTagSource) HasDockerfile() bool {
	if o != nil && o.Dockerfile != nil {
		return true
	}

	return false
}

// SetDockerfile gets a reference to the given string and assigns it to the Dockerfile field.
func (o *RegistryTagSource) SetDockerfile(v string) {
	o.Dockerfile = &v
}

func (o RegistryTagSource) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["pullstring"] = o.Pullstring
	}
	if o.Dockerfile != nil {
		toSerialize["dockerfile"] = o.Dockerfile
	}
	return json.Marshal(toSerialize)
}

type NullableRegistryTagSource struct {
	value *RegistryTagSource
	isSet bool
}

func (v NullableRegistryTagSource) Get() *RegistryTagSource {
	return v.value
}

func (v *NullableRegistryTagSource) Set(val *RegistryTagSource) {
	v.value = val
	v.isSet = true
}

func (v NullableRegistryTagSource) IsSet() bool {
	return v.isSet
}

func (v *NullableRegistryTagSource) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableRegistryTagSource(val *RegistryTagSource) *NullableRegistryTagSource {
	return &NullableRegistryTagSource{value: val, isSet: true}
}

func (v NullableRegistryTagSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableRegistryTagSource) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


