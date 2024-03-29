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

// DeleteImageResponse Image deletion response containing status and details
type DeleteImageResponse struct {
	Digest string `json:"digest"`
	// Current status of the image deletion
	Status string `json:"status"`
	Detail *string `json:"detail,omitempty"`
}

// NewDeleteImageResponse instantiates a new DeleteImageResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewDeleteImageResponse(digest string, status string) *DeleteImageResponse {
	this := DeleteImageResponse{}
	this.Digest = digest
	this.Status = status
	return &this
}

// NewDeleteImageResponseWithDefaults instantiates a new DeleteImageResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewDeleteImageResponseWithDefaults() *DeleteImageResponse {
	this := DeleteImageResponse{}
	return &this
}

// GetDigest returns the Digest field value
func (o *DeleteImageResponse) GetDigest() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Digest
}

// GetDigestOk returns a tuple with the Digest field value
// and a boolean to check if the value has been set.
func (o *DeleteImageResponse) GetDigestOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Digest, true
}

// SetDigest sets field value
func (o *DeleteImageResponse) SetDigest(v string) {
	o.Digest = v
}

// GetStatus returns the Status field value
func (o *DeleteImageResponse) GetStatus() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Status
}

// GetStatusOk returns a tuple with the Status field value
// and a boolean to check if the value has been set.
func (o *DeleteImageResponse) GetStatusOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Status, true
}

// SetStatus sets field value
func (o *DeleteImageResponse) SetStatus(v string) {
	o.Status = v
}

// GetDetail returns the Detail field value if set, zero value otherwise.
func (o *DeleteImageResponse) GetDetail() string {
	if o == nil || o.Detail == nil {
		var ret string
		return ret
	}
	return *o.Detail
}

// GetDetailOk returns a tuple with the Detail field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *DeleteImageResponse) GetDetailOk() (*string, bool) {
	if o == nil || o.Detail == nil {
		return nil, false
	}
	return o.Detail, true
}

// HasDetail returns a boolean if a field has been set.
func (o *DeleteImageResponse) HasDetail() bool {
	if o != nil && o.Detail != nil {
		return true
	}

	return false
}

// SetDetail gets a reference to the given string and assigns it to the Detail field.
func (o *DeleteImageResponse) SetDetail(v string) {
	o.Detail = &v
}

func (o DeleteImageResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["digest"] = o.Digest
	}
	if true {
		toSerialize["status"] = o.Status
	}
	if o.Detail != nil {
		toSerialize["detail"] = o.Detail
	}
	return json.Marshal(toSerialize)
}

type NullableDeleteImageResponse struct {
	value *DeleteImageResponse
	isSet bool
}

func (v NullableDeleteImageResponse) Get() *DeleteImageResponse {
	return v.value
}

func (v *NullableDeleteImageResponse) Set(val *DeleteImageResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableDeleteImageResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableDeleteImageResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableDeleteImageResponse(val *DeleteImageResponse) *NullableDeleteImageResponse {
	return &NullableDeleteImageResponse{value: val, isSet: true}
}

func (v NullableDeleteImageResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableDeleteImageResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


