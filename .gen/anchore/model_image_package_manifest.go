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

// ImagePackageManifest struct for ImagePackageManifest
type ImagePackageManifest struct {
	Artifacts []ImportPackage `json:"artifacts"`
	Source ImportSource `json:"source"`
	Distro ImportDistribution `json:"distro"`
	Descriptor *ImportDescriptor `json:"descriptor,omitempty"`
	Schema *ImportSchema `json:"schema,omitempty"`
	ArtifactRelationships []ImportPackageRelationship `json:"artifactRelationships,omitempty"`
}

// NewImagePackageManifest instantiates a new ImagePackageManifest object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewImagePackageManifest(artifacts []ImportPackage, source ImportSource, distro ImportDistribution) *ImagePackageManifest {
	this := ImagePackageManifest{}
	this.Artifacts = artifacts
	this.Source = source
	this.Distro = distro
	return &this
}

// NewImagePackageManifestWithDefaults instantiates a new ImagePackageManifest object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewImagePackageManifestWithDefaults() *ImagePackageManifest {
	this := ImagePackageManifest{}
	return &this
}

// GetArtifacts returns the Artifacts field value
func (o *ImagePackageManifest) GetArtifacts() []ImportPackage {
	if o == nil {
		var ret []ImportPackage
		return ret
	}

	return o.Artifacts
}

// GetArtifactsOk returns a tuple with the Artifacts field value
// and a boolean to check if the value has been set.
func (o *ImagePackageManifest) GetArtifactsOk() ([]ImportPackage, bool) {
	if o == nil {
		return nil, false
	}
	return o.Artifacts, true
}

// SetArtifacts sets field value
func (o *ImagePackageManifest) SetArtifacts(v []ImportPackage) {
	o.Artifacts = v
}

// GetSource returns the Source field value
func (o *ImagePackageManifest) GetSource() ImportSource {
	if o == nil {
		var ret ImportSource
		return ret
	}

	return o.Source
}

// GetSourceOk returns a tuple with the Source field value
// and a boolean to check if the value has been set.
func (o *ImagePackageManifest) GetSourceOk() (*ImportSource, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Source, true
}

// SetSource sets field value
func (o *ImagePackageManifest) SetSource(v ImportSource) {
	o.Source = v
}

// GetDistro returns the Distro field value
func (o *ImagePackageManifest) GetDistro() ImportDistribution {
	if o == nil {
		var ret ImportDistribution
		return ret
	}

	return o.Distro
}

// GetDistroOk returns a tuple with the Distro field value
// and a boolean to check if the value has been set.
func (o *ImagePackageManifest) GetDistroOk() (*ImportDistribution, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Distro, true
}

// SetDistro sets field value
func (o *ImagePackageManifest) SetDistro(v ImportDistribution) {
	o.Distro = v
}

// GetDescriptor returns the Descriptor field value if set, zero value otherwise.
func (o *ImagePackageManifest) GetDescriptor() ImportDescriptor {
	if o == nil || o.Descriptor == nil {
		var ret ImportDescriptor
		return ret
	}
	return *o.Descriptor
}

// GetDescriptorOk returns a tuple with the Descriptor field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImagePackageManifest) GetDescriptorOk() (*ImportDescriptor, bool) {
	if o == nil || o.Descriptor == nil {
		return nil, false
	}
	return o.Descriptor, true
}

// HasDescriptor returns a boolean if a field has been set.
func (o *ImagePackageManifest) HasDescriptor() bool {
	if o != nil && o.Descriptor != nil {
		return true
	}

	return false
}

// SetDescriptor gets a reference to the given ImportDescriptor and assigns it to the Descriptor field.
func (o *ImagePackageManifest) SetDescriptor(v ImportDescriptor) {
	o.Descriptor = &v
}

// GetSchema returns the Schema field value if set, zero value otherwise.
func (o *ImagePackageManifest) GetSchema() ImportSchema {
	if o == nil || o.Schema == nil {
		var ret ImportSchema
		return ret
	}
	return *o.Schema
}

// GetSchemaOk returns a tuple with the Schema field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImagePackageManifest) GetSchemaOk() (*ImportSchema, bool) {
	if o == nil || o.Schema == nil {
		return nil, false
	}
	return o.Schema, true
}

// HasSchema returns a boolean if a field has been set.
func (o *ImagePackageManifest) HasSchema() bool {
	if o != nil && o.Schema != nil {
		return true
	}

	return false
}

// SetSchema gets a reference to the given ImportSchema and assigns it to the Schema field.
func (o *ImagePackageManifest) SetSchema(v ImportSchema) {
	o.Schema = &v
}

// GetArtifactRelationships returns the ArtifactRelationships field value if set, zero value otherwise.
func (o *ImagePackageManifest) GetArtifactRelationships() []ImportPackageRelationship {
	if o == nil || o.ArtifactRelationships == nil {
		var ret []ImportPackageRelationship
		return ret
	}
	return o.ArtifactRelationships
}

// GetArtifactRelationshipsOk returns a tuple with the ArtifactRelationships field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ImagePackageManifest) GetArtifactRelationshipsOk() ([]ImportPackageRelationship, bool) {
	if o == nil || o.ArtifactRelationships == nil {
		return nil, false
	}
	return o.ArtifactRelationships, true
}

// HasArtifactRelationships returns a boolean if a field has been set.
func (o *ImagePackageManifest) HasArtifactRelationships() bool {
	if o != nil && o.ArtifactRelationships != nil {
		return true
	}

	return false
}

// SetArtifactRelationships gets a reference to the given []ImportPackageRelationship and assigns it to the ArtifactRelationships field.
func (o *ImagePackageManifest) SetArtifactRelationships(v []ImportPackageRelationship) {
	o.ArtifactRelationships = v
}

func (o ImagePackageManifest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["artifacts"] = o.Artifacts
	}
	if true {
		toSerialize["source"] = o.Source
	}
	if true {
		toSerialize["distro"] = o.Distro
	}
	if o.Descriptor != nil {
		toSerialize["descriptor"] = o.Descriptor
	}
	if o.Schema != nil {
		toSerialize["schema"] = o.Schema
	}
	if o.ArtifactRelationships != nil {
		toSerialize["artifactRelationships"] = o.ArtifactRelationships
	}
	return json.Marshal(toSerialize)
}

type NullableImagePackageManifest struct {
	value *ImagePackageManifest
	isSet bool
}

func (v NullableImagePackageManifest) Get() *ImagePackageManifest {
	return v.value
}

func (v *NullableImagePackageManifest) Set(val *ImagePackageManifest) {
	v.value = val
	v.isSet = true
}

func (v NullableImagePackageManifest) IsSet() bool {
	return v.isSet
}

func (v *NullableImagePackageManifest) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableImagePackageManifest(val *ImagePackageManifest) *NullableImagePackageManifest {
	return &NullableImagePackageManifest{value: val, isSet: true}
}

func (v NullableImagePackageManifest) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableImagePackageManifest) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


