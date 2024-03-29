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

// ContentResponse Generic wrapper for content listings from images
type ContentResponse struct {
	ImageDigest *string `json:"imageDigest,omitempty"`
	ContentType *string `json:"content_type,omitempty"`
	Content []map[string]interface{} `json:"content,omitempty"`
}

// NewContentResponse instantiates a new ContentResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewContentResponse() *ContentResponse {
	this := ContentResponse{}
	return &this
}

// NewContentResponseWithDefaults instantiates a new ContentResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewContentResponseWithDefaults() *ContentResponse {
	this := ContentResponse{}
	return &this
}

// GetImageDigest returns the ImageDigest field value if set, zero value otherwise.
func (o *ContentResponse) GetImageDigest() string {
	if o == nil || o.ImageDigest == nil {
		var ret string
		return ret
	}
	return *o.ImageDigest
}

// GetImageDigestOk returns a tuple with the ImageDigest field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentResponse) GetImageDigestOk() (*string, bool) {
	if o == nil || o.ImageDigest == nil {
		return nil, false
	}
	return o.ImageDigest, true
}

// HasImageDigest returns a boolean if a field has been set.
func (o *ContentResponse) HasImageDigest() bool {
	if o != nil && o.ImageDigest != nil {
		return true
	}

	return false
}

// SetImageDigest gets a reference to the given string and assigns it to the ImageDigest field.
func (o *ContentResponse) SetImageDigest(v string) {
	o.ImageDigest = &v
}

// GetContentType returns the ContentType field value if set, zero value otherwise.
func (o *ContentResponse) GetContentType() string {
	if o == nil || o.ContentType == nil {
		var ret string
		return ret
	}
	return *o.ContentType
}

// GetContentTypeOk returns a tuple with the ContentType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentResponse) GetContentTypeOk() (*string, bool) {
	if o == nil || o.ContentType == nil {
		return nil, false
	}
	return o.ContentType, true
}

// HasContentType returns a boolean if a field has been set.
func (o *ContentResponse) HasContentType() bool {
	if o != nil && o.ContentType != nil {
		return true
	}

	return false
}

// SetContentType gets a reference to the given string and assigns it to the ContentType field.
func (o *ContentResponse) SetContentType(v string) {
	o.ContentType = &v
}

// GetContent returns the Content field value if set, zero value otherwise.
func (o *ContentResponse) GetContent() []map[string]interface{} {
	if o == nil || o.Content == nil {
		var ret []map[string]interface{}
		return ret
	}
	return o.Content
}

// GetContentOk returns a tuple with the Content field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentResponse) GetContentOk() ([]map[string]interface{}, bool) {
	if o == nil || o.Content == nil {
		return nil, false
	}
	return o.Content, true
}

// HasContent returns a boolean if a field has been set.
func (o *ContentResponse) HasContent() bool {
	if o != nil && o.Content != nil {
		return true
	}

	return false
}

// SetContent gets a reference to the given []map[string]interface{} and assigns it to the Content field.
func (o *ContentResponse) SetContent(v []map[string]interface{}) {
	o.Content = v
}

func (o ContentResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.ImageDigest != nil {
		toSerialize["imageDigest"] = o.ImageDigest
	}
	if o.ContentType != nil {
		toSerialize["content_type"] = o.ContentType
	}
	if o.Content != nil {
		toSerialize["content"] = o.Content
	}
	return json.Marshal(toSerialize)
}

type NullableContentResponse struct {
	value *ContentResponse
	isSet bool
}

func (v NullableContentResponse) Get() *ContentResponse {
	return v.value
}

func (v *NullableContentResponse) Set(val *ContentResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableContentResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableContentResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableContentResponse(val *ContentResponse) *NullableContentResponse {
	return &NullableContentResponse{value: val, isSet: true}
}

func (v NullableContentResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableContentResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


