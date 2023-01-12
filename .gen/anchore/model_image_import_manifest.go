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

// ImageImportManifest struct for ImageImportManifest
type ImageImportManifest struct {
	Contents *ImportContentDigests `json:"contents,omitempty"`
	Tags []string `json:"tags,omitempty"`
	Digest *string `json:"digest,omitempty"`
	// The digest of the images's manifest-list parent if it was accessed from a multi-arch tag where the tag pointed to a manifest-list. This allows preservation of that relationship in the data
	ParentDigest *string `json:"parent_digest,omitempty"`
	// An \"imageId\" as used by Docker if available
	LocalImageId *string `json:"local_image_id,omitempty"`
	OperationUuid *string `json:"operation_uuid,omitempty"`
}

// NewImageImportManifest instantiates a new ImageImportManifest object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewImageImportManifest() *ImageImportManifest {
	this := ImageImportManifest{}
	return &this
}

// NewImageImportManifestWithDefaults instantiates a new ImageImportManifest object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewImageImportManifestWithDefaults() *ImageImportManifest {
	this := ImageImportManifest{}
	return &this
}

// GetContents returns the Contents field value if set, zero value otherwise.
func (o *ImageImportManifest) GetContents() ImportContentDigests {
	if o == nil || o.Contents == nil {
		var ret ImportContentDigests
		return ret
	}
	return *o.Contents
}

// GetContentsOk returns a tuple with the Contents field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImageImportManifest) GetContentsOk() (*ImportContentDigests, bool) {
	if o == nil || o.Contents == nil {
		return nil, false
	}
	return o.Contents, true
}

// HasContents returns a boolean if a field has been set.
func (o *ImageImportManifest) HasContents() bool {
	if o != nil && o.Contents != nil {
		return true
	}

	return false
}

// SetContents gets a reference to the given ImportContentDigests and assigns it to the Contents field.
func (o *ImageImportManifest) SetContents(v ImportContentDigests) {
	o.Contents = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *ImageImportManifest) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImageImportManifest) GetTagsOk() ([]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *ImageImportManifest) HasTags() bool {
	if o != nil && o.Tags != nil {
		return true
	}

	return false
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *ImageImportManifest) SetTags(v []string) {
	o.Tags = v
}

// GetDigest returns the Digest field value if set, zero value otherwise.
func (o *ImageImportManifest) GetDigest() string {
	if o == nil || o.Digest == nil {
		var ret string
		return ret
	}
	return *o.Digest
}

// GetDigestOk returns a tuple with the Digest field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImageImportManifest) GetDigestOk() (*string, bool) {
	if o == nil || o.Digest == nil {
		return nil, false
	}
	return o.Digest, true
}

// HasDigest returns a boolean if a field has been set.
func (o *ImageImportManifest) HasDigest() bool {
	if o != nil && o.Digest != nil {
		return true
	}

	return false
}

// SetDigest gets a reference to the given string and assigns it to the Digest field.
func (o *ImageImportManifest) SetDigest(v string) {
	o.Digest = &v
}

// GetParentDigest returns the ParentDigest field value if set, zero value otherwise.
func (o *ImageImportManifest) GetParentDigest() string {
	if o == nil || o.ParentDigest == nil {
		var ret string
		return ret
	}
	return *o.ParentDigest
}

// GetParentDigestOk returns a tuple with the ParentDigest field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImageImportManifest) GetParentDigestOk() (*string, bool) {
	if o == nil || o.ParentDigest == nil {
		return nil, false
	}
	return o.ParentDigest, true
}

// HasParentDigest returns a boolean if a field has been set.
func (o *ImageImportManifest) HasParentDigest() bool {
	if o != nil && o.ParentDigest != nil {
		return true
	}

	return false
}

// SetParentDigest gets a reference to the given string and assigns it to the ParentDigest field.
func (o *ImageImportManifest) SetParentDigest(v string) {
	o.ParentDigest = &v
}

// GetLocalImageId returns the LocalImageId field value if set, zero value otherwise.
func (o *ImageImportManifest) GetLocalImageId() string {
	if o == nil || o.LocalImageId == nil {
		var ret string
		return ret
	}
	return *o.LocalImageId
}

// GetLocalImageIdOk returns a tuple with the LocalImageId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImageImportManifest) GetLocalImageIdOk() (*string, bool) {
	if o == nil || o.LocalImageId == nil {
		return nil, false
	}
	return o.LocalImageId, true
}

// HasLocalImageId returns a boolean if a field has been set.
func (o *ImageImportManifest) HasLocalImageId() bool {
	if o != nil && o.LocalImageId != nil {
		return true
	}

	return false
}

// SetLocalImageId gets a reference to the given string and assigns it to the LocalImageId field.
func (o *ImageImportManifest) SetLocalImageId(v string) {
	o.LocalImageId = &v
}

// GetOperationUuid returns the OperationUuid field value if set, zero value otherwise.
func (o *ImageImportManifest) GetOperationUuid() string {
	if o == nil || o.OperationUuid == nil {
		var ret string
		return ret
	}
	return *o.OperationUuid
}

// GetOperationUuidOk returns a tuple with the OperationUuid field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImageImportManifest) GetOperationUuidOk() (*string, bool) {
	if o == nil || o.OperationUuid == nil {
		return nil, false
	}
	return o.OperationUuid, true
}

// HasOperationUuid returns a boolean if a field has been set.
func (o *ImageImportManifest) HasOperationUuid() bool {
	if o != nil && o.OperationUuid != nil {
		return true
	}

	return false
}

// SetOperationUuid gets a reference to the given string and assigns it to the OperationUuid field.
func (o *ImageImportManifest) SetOperationUuid(v string) {
	o.OperationUuid = &v
}

func (o ImageImportManifest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.Contents != nil {
		toSerialize["contents"] = o.Contents
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}
	if o.Digest != nil {
		toSerialize["digest"] = o.Digest
	}
	if o.ParentDigest != nil {
		toSerialize["parent_digest"] = o.ParentDigest
	}
	if o.LocalImageId != nil {
		toSerialize["local_image_id"] = o.LocalImageId
	}
	if o.OperationUuid != nil {
		toSerialize["operation_uuid"] = o.OperationUuid
	}
	return json.Marshal(toSerialize)
}

type NullableImageImportManifest struct {
	value *ImageImportManifest
	isSet bool
}

func (v NullableImageImportManifest) Get() *ImageImportManifest {
	return v.value
}

func (v *NullableImageImportManifest) Set(val *ImageImportManifest) {
	v.value = val
	v.isSet = true
}

func (v NullableImageImportManifest) IsSet() bool {
	return v.isSet
}

func (v *NullableImageImportManifest) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableImageImportManifest(val *ImageImportManifest) *NullableImageImportManifest {
	return &NullableImageImportManifest{value: val, isSet: true}
}

func (v NullableImageImportManifest) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableImageImportManifest) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

