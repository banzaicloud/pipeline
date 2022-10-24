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
	"time"
)

// TagEntry A docker-pullable tag value as well as deconstructed components
type TagEntry struct {
	// The pullable string for the tag. E.g. \"docker.io/library/node:latest\"
	Pullstring *string `json:"pullstring,omitempty"`
	// The registry hostname:port section of the pull string
	Registry *string `json:"registry,omitempty"`
	// The repository section of the pull string
	Repository *string `json:"repository,omitempty"`
	// The tag-only section of the pull string
	Tag *string `json:"tag,omitempty"`
	// The timestamp at which the Anchore Engine detected this tag was mapped to the image digest. Does not necessarily indicate when the tag was actually pushed to the registry.
	DetectedAt *time.Time `json:"detected_at,omitempty"`
}

// NewTagEntry instantiates a new TagEntry object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewTagEntry() *TagEntry {
	this := TagEntry{}
	return &this
}

// NewTagEntryWithDefaults instantiates a new TagEntry object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewTagEntryWithDefaults() *TagEntry {
	this := TagEntry{}
	return &this
}

// GetPullstring returns the Pullstring field value if set, zero value otherwise.
func (o *TagEntry) GetPullstring() string {
	if o == nil || o.Pullstring == nil {
		var ret string
		return ret
	}
	return *o.Pullstring
}

// GetPullstringOk returns a tuple with the Pullstring field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TagEntry) GetPullstringOk() (*string, bool) {
	if o == nil || o.Pullstring == nil {
		return nil, false
	}
	return o.Pullstring, true
}

// HasPullstring returns a boolean if a field has been set.
func (o *TagEntry) HasPullstring() bool {
	if o != nil && o.Pullstring != nil {
		return true
	}

	return false
}

// SetPullstring gets a reference to the given string and assigns it to the Pullstring field.
func (o *TagEntry) SetPullstring(v string) {
	o.Pullstring = &v
}

// GetRegistry returns the Registry field value if set, zero value otherwise.
func (o *TagEntry) GetRegistry() string {
	if o == nil || o.Registry == nil {
		var ret string
		return ret
	}
	return *o.Registry
}

// GetRegistryOk returns a tuple with the Registry field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TagEntry) GetRegistryOk() (*string, bool) {
	if o == nil || o.Registry == nil {
		return nil, false
	}
	return o.Registry, true
}

// HasRegistry returns a boolean if a field has been set.
func (o *TagEntry) HasRegistry() bool {
	if o != nil && o.Registry != nil {
		return true
	}

	return false
}

// SetRegistry gets a reference to the given string and assigns it to the Registry field.
func (o *TagEntry) SetRegistry(v string) {
	o.Registry = &v
}

// GetRepository returns the Repository field value if set, zero value otherwise.
func (o *TagEntry) GetRepository() string {
	if o == nil || o.Repository == nil {
		var ret string
		return ret
	}
	return *o.Repository
}

// GetRepositoryOk returns a tuple with the Repository field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TagEntry) GetRepositoryOk() (*string, bool) {
	if o == nil || o.Repository == nil {
		return nil, false
	}
	return o.Repository, true
}

// HasRepository returns a boolean if a field has been set.
func (o *TagEntry) HasRepository() bool {
	if o != nil && o.Repository != nil {
		return true
	}

	return false
}

// SetRepository gets a reference to the given string and assigns it to the Repository field.
func (o *TagEntry) SetRepository(v string) {
	o.Repository = &v
}

// GetTag returns the Tag field value if set, zero value otherwise.
func (o *TagEntry) GetTag() string {
	if o == nil || o.Tag == nil {
		var ret string
		return ret
	}
	return *o.Tag
}

// GetTagOk returns a tuple with the Tag field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TagEntry) GetTagOk() (*string, bool) {
	if o == nil || o.Tag == nil {
		return nil, false
	}
	return o.Tag, true
}

// HasTag returns a boolean if a field has been set.
func (o *TagEntry) HasTag() bool {
	if o != nil && o.Tag != nil {
		return true
	}

	return false
}

// SetTag gets a reference to the given string and assigns it to the Tag field.
func (o *TagEntry) SetTag(v string) {
	o.Tag = &v
}

// GetDetectedAt returns the DetectedAt field value if set, zero value otherwise.
func (o *TagEntry) GetDetectedAt() time.Time {
	if o == nil || o.DetectedAt == nil {
		var ret time.Time
		return ret
	}
	return *o.DetectedAt
}

// GetDetectedAtOk returns a tuple with the DetectedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TagEntry) GetDetectedAtOk() (*time.Time, bool) {
	if o == nil || o.DetectedAt == nil {
		return nil, false
	}
	return o.DetectedAt, true
}

// HasDetectedAt returns a boolean if a field has been set.
func (o *TagEntry) HasDetectedAt() bool {
	if o != nil && o.DetectedAt != nil {
		return true
	}

	return false
}

// SetDetectedAt gets a reference to the given time.Time and assigns it to the DetectedAt field.
func (o *TagEntry) SetDetectedAt(v time.Time) {
	o.DetectedAt = &v
}

func (o TagEntry) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.Pullstring != nil {
		toSerialize["pullstring"] = o.Pullstring
	}
	if o.Registry != nil {
		toSerialize["registry"] = o.Registry
	}
	if o.Repository != nil {
		toSerialize["repository"] = o.Repository
	}
	if o.Tag != nil {
		toSerialize["tag"] = o.Tag
	}
	if o.DetectedAt != nil {
		toSerialize["detected_at"] = o.DetectedAt
	}
	return json.Marshal(toSerialize)
}

type NullableTagEntry struct {
	value *TagEntry
	isSet bool
}

func (v NullableTagEntry) Get() *TagEntry {
	return v.value
}

func (v *NullableTagEntry) Set(val *TagEntry) {
	v.value = val
	v.isSet = true
}

func (v NullableTagEntry) IsSet() bool {
	return v.isSet
}

func (v *NullableTagEntry) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableTagEntry(val *TagEntry) *NullableTagEntry {
	return &NullableTagEntry{value: val, isSet: true}
}

func (v NullableTagEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableTagEntry) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


