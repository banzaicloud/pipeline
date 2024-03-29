/*
Product Info.

The product info application uses the cloud provider APIs to asynchronously fetch and parse instance type attributes and prices, while storing the results in an in memory cache and making it available as structured data through a REST API.

API version: 0.9.5
Contact: info@banzaicloud.com
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package cloudinfo

import (
	"encoding/json"
)

// AttributeResponse AttributeResponse holds attribute values
type AttributeResponse struct {
	AttributeName *string `json:"attributeName,omitempty"`
	AttributeValues []float64 `json:"attributeValues,omitempty"`
}

// NewAttributeResponse instantiates a new AttributeResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewAttributeResponse() *AttributeResponse {
	this := AttributeResponse{}
	return &this
}

// NewAttributeResponseWithDefaults instantiates a new AttributeResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewAttributeResponseWithDefaults() *AttributeResponse {
	this := AttributeResponse{}
	return &this
}

// GetAttributeName returns the AttributeName field value if set, zero value otherwise.
func (o *AttributeResponse) GetAttributeName() string {
	if o == nil || o.AttributeName == nil {
		var ret string
		return ret
	}
	return *o.AttributeName
}

// GetAttributeNameOk returns a tuple with the AttributeName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AttributeResponse) GetAttributeNameOk() (*string, bool) {
	if o == nil || o.AttributeName == nil {
		return nil, false
	}
	return o.AttributeName, true
}

// HasAttributeName returns a boolean if a field has been set.
func (o *AttributeResponse) HasAttributeName() bool {
	if o != nil && o.AttributeName != nil {
		return true
	}

	return false
}

// SetAttributeName gets a reference to the given string and assigns it to the AttributeName field.
func (o *AttributeResponse) SetAttributeName(v string) {
	o.AttributeName = &v
}

// GetAttributeValues returns the AttributeValues field value if set, zero value otherwise.
func (o *AttributeResponse) GetAttributeValues() []float64 {
	if o == nil || o.AttributeValues == nil {
		var ret []float64
		return ret
	}
	return o.AttributeValues
}

// GetAttributeValuesOk returns a tuple with the AttributeValues field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *AttributeResponse) GetAttributeValuesOk() ([]float64, bool) {
	if o == nil || o.AttributeValues == nil {
		return nil, false
	}
	return o.AttributeValues, true
}

// HasAttributeValues returns a boolean if a field has been set.
func (o *AttributeResponse) HasAttributeValues() bool {
	if o != nil && o.AttributeValues != nil {
		return true
	}

	return false
}

// SetAttributeValues gets a reference to the given []float64 and assigns it to the AttributeValues field.
func (o *AttributeResponse) SetAttributeValues(v []float64) {
	o.AttributeValues = v
}

func (o AttributeResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.AttributeName != nil {
		toSerialize["attributeName"] = o.AttributeName
	}
	if o.AttributeValues != nil {
		toSerialize["attributeValues"] = o.AttributeValues
	}
	return json.Marshal(toSerialize)
}

type NullableAttributeResponse struct {
	value *AttributeResponse
	isSet bool
}

func (v NullableAttributeResponse) Get() *AttributeResponse {
	return v.value
}

func (v *NullableAttributeResponse) Set(val *AttributeResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableAttributeResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableAttributeResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableAttributeResponse(val *AttributeResponse) *NullableAttributeResponse {
	return &NullableAttributeResponse{value: val, isSet: true}
}

func (v NullableAttributeResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableAttributeResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


