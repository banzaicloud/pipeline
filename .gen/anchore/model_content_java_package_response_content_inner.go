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

// ContentJAVAPackageResponseContentInner struct for ContentJAVAPackageResponseContentInner
type ContentJAVAPackageResponseContentInner struct {
	Package *string `json:"package,omitempty"`
	ImplementationVersion *string `json:"implementation-version,omitempty"`
	SpecificationVersion *string `json:"specification-version,omitempty"`
	MavenVersion *string `json:"maven-version,omitempty"`
	Location *string `json:"location,omitempty"`
	Type *string `json:"type,omitempty"`
	Origin *string `json:"origin,omitempty"`
	// A list of Common Platform Enumerations that may uniquely identify the package
	Cpes []string `json:"cpes,omitempty"`
}

// NewContentJAVAPackageResponseContentInner instantiates a new ContentJAVAPackageResponseContentInner object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewContentJAVAPackageResponseContentInner() *ContentJAVAPackageResponseContentInner {
	this := ContentJAVAPackageResponseContentInner{}
	return &this
}

// NewContentJAVAPackageResponseContentInnerWithDefaults instantiates a new ContentJAVAPackageResponseContentInner object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewContentJAVAPackageResponseContentInnerWithDefaults() *ContentJAVAPackageResponseContentInner {
	this := ContentJAVAPackageResponseContentInner{}
	return &this
}

// GetPackage returns the Package field value if set, zero value otherwise.
func (o *ContentJAVAPackageResponseContentInner) GetPackage() string {
	if o == nil || o.Package == nil {
		var ret string
		return ret
	}
	return *o.Package
}

// GetPackageOk returns a tuple with the Package field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentJAVAPackageResponseContentInner) GetPackageOk() (*string, bool) {
	if o == nil || o.Package == nil {
		return nil, false
	}
	return o.Package, true
}

// HasPackage returns a boolean if a field has been set.
func (o *ContentJAVAPackageResponseContentInner) HasPackage() bool {
	if o != nil && o.Package != nil {
		return true
	}

	return false
}

// SetPackage gets a reference to the given string and assigns it to the Package field.
func (o *ContentJAVAPackageResponseContentInner) SetPackage(v string) {
	o.Package = &v
}

// GetImplementationVersion returns the ImplementationVersion field value if set, zero value otherwise.
func (o *ContentJAVAPackageResponseContentInner) GetImplementationVersion() string {
	if o == nil || o.ImplementationVersion == nil {
		var ret string
		return ret
	}
	return *o.ImplementationVersion
}

// GetImplementationVersionOk returns a tuple with the ImplementationVersion field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentJAVAPackageResponseContentInner) GetImplementationVersionOk() (*string, bool) {
	if o == nil || o.ImplementationVersion == nil {
		return nil, false
	}
	return o.ImplementationVersion, true
}

// HasImplementationVersion returns a boolean if a field has been set.
func (o *ContentJAVAPackageResponseContentInner) HasImplementationVersion() bool {
	if o != nil && o.ImplementationVersion != nil {
		return true
	}

	return false
}

// SetImplementationVersion gets a reference to the given string and assigns it to the ImplementationVersion field.
func (o *ContentJAVAPackageResponseContentInner) SetImplementationVersion(v string) {
	o.ImplementationVersion = &v
}

// GetSpecificationVersion returns the SpecificationVersion field value if set, zero value otherwise.
func (o *ContentJAVAPackageResponseContentInner) GetSpecificationVersion() string {
	if o == nil || o.SpecificationVersion == nil {
		var ret string
		return ret
	}
	return *o.SpecificationVersion
}

// GetSpecificationVersionOk returns a tuple with the SpecificationVersion field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentJAVAPackageResponseContentInner) GetSpecificationVersionOk() (*string, bool) {
	if o == nil || o.SpecificationVersion == nil {
		return nil, false
	}
	return o.SpecificationVersion, true
}

// HasSpecificationVersion returns a boolean if a field has been set.
func (o *ContentJAVAPackageResponseContentInner) HasSpecificationVersion() bool {
	if o != nil && o.SpecificationVersion != nil {
		return true
	}

	return false
}

// SetSpecificationVersion gets a reference to the given string and assigns it to the SpecificationVersion field.
func (o *ContentJAVAPackageResponseContentInner) SetSpecificationVersion(v string) {
	o.SpecificationVersion = &v
}

// GetMavenVersion returns the MavenVersion field value if set, zero value otherwise.
func (o *ContentJAVAPackageResponseContentInner) GetMavenVersion() string {
	if o == nil || o.MavenVersion == nil {
		var ret string
		return ret
	}
	return *o.MavenVersion
}

// GetMavenVersionOk returns a tuple with the MavenVersion field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentJAVAPackageResponseContentInner) GetMavenVersionOk() (*string, bool) {
	if o == nil || o.MavenVersion == nil {
		return nil, false
	}
	return o.MavenVersion, true
}

// HasMavenVersion returns a boolean if a field has been set.
func (o *ContentJAVAPackageResponseContentInner) HasMavenVersion() bool {
	if o != nil && o.MavenVersion != nil {
		return true
	}

	return false
}

// SetMavenVersion gets a reference to the given string and assigns it to the MavenVersion field.
func (o *ContentJAVAPackageResponseContentInner) SetMavenVersion(v string) {
	o.MavenVersion = &v
}

// GetLocation returns the Location field value if set, zero value otherwise.
func (o *ContentJAVAPackageResponseContentInner) GetLocation() string {
	if o == nil || o.Location == nil {
		var ret string
		return ret
	}
	return *o.Location
}

// GetLocationOk returns a tuple with the Location field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentJAVAPackageResponseContentInner) GetLocationOk() (*string, bool) {
	if o == nil || o.Location == nil {
		return nil, false
	}
	return o.Location, true
}

// HasLocation returns a boolean if a field has been set.
func (o *ContentJAVAPackageResponseContentInner) HasLocation() bool {
	if o != nil && o.Location != nil {
		return true
	}

	return false
}

// SetLocation gets a reference to the given string and assigns it to the Location field.
func (o *ContentJAVAPackageResponseContentInner) SetLocation(v string) {
	o.Location = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *ContentJAVAPackageResponseContentInner) GetType() string {
	if o == nil || o.Type == nil {
		var ret string
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentJAVAPackageResponseContentInner) GetTypeOk() (*string, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *ContentJAVAPackageResponseContentInner) HasType() bool {
	if o != nil && o.Type != nil {
		return true
	}

	return false
}

// SetType gets a reference to the given string and assigns it to the Type field.
func (o *ContentJAVAPackageResponseContentInner) SetType(v string) {
	o.Type = &v
}

// GetOrigin returns the Origin field value if set, zero value otherwise.
func (o *ContentJAVAPackageResponseContentInner) GetOrigin() string {
	if o == nil || o.Origin == nil {
		var ret string
		return ret
	}
	return *o.Origin
}

// GetOriginOk returns a tuple with the Origin field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentJAVAPackageResponseContentInner) GetOriginOk() (*string, bool) {
	if o == nil || o.Origin == nil {
		return nil, false
	}
	return o.Origin, true
}

// HasOrigin returns a boolean if a field has been set.
func (o *ContentJAVAPackageResponseContentInner) HasOrigin() bool {
	if o != nil && o.Origin != nil {
		return true
	}

	return false
}

// SetOrigin gets a reference to the given string and assigns it to the Origin field.
func (o *ContentJAVAPackageResponseContentInner) SetOrigin(v string) {
	o.Origin = &v
}

// GetCpes returns the Cpes field value if set, zero value otherwise.
func (o *ContentJAVAPackageResponseContentInner) GetCpes() []string {
	if o == nil || o.Cpes == nil {
		var ret []string
		return ret
	}
	return o.Cpes
}

// GetCpesOk returns a tuple with the Cpes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentJAVAPackageResponseContentInner) GetCpesOk() ([]string, bool) {
	if o == nil || o.Cpes == nil {
		return nil, false
	}
	return o.Cpes, true
}

// HasCpes returns a boolean if a field has been set.
func (o *ContentJAVAPackageResponseContentInner) HasCpes() bool {
	if o != nil && o.Cpes != nil {
		return true
	}

	return false
}

// SetCpes gets a reference to the given []string and assigns it to the Cpes field.
func (o *ContentJAVAPackageResponseContentInner) SetCpes(v []string) {
	o.Cpes = v
}

func (o ContentJAVAPackageResponseContentInner) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.Package != nil {
		toSerialize["package"] = o.Package
	}
	if o.ImplementationVersion != nil {
		toSerialize["implementation-version"] = o.ImplementationVersion
	}
	if o.SpecificationVersion != nil {
		toSerialize["specification-version"] = o.SpecificationVersion
	}
	if o.MavenVersion != nil {
		toSerialize["maven-version"] = o.MavenVersion
	}
	if o.Location != nil {
		toSerialize["location"] = o.Location
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}
	if o.Origin != nil {
		toSerialize["origin"] = o.Origin
	}
	if o.Cpes != nil {
		toSerialize["cpes"] = o.Cpes
	}
	return json.Marshal(toSerialize)
}

type NullableContentJAVAPackageResponseContentInner struct {
	value *ContentJAVAPackageResponseContentInner
	isSet bool
}

func (v NullableContentJAVAPackageResponseContentInner) Get() *ContentJAVAPackageResponseContentInner {
	return v.value
}

func (v *NullableContentJAVAPackageResponseContentInner) Set(val *ContentJAVAPackageResponseContentInner) {
	v.value = val
	v.isSet = true
}

func (v NullableContentJAVAPackageResponseContentInner) IsSet() bool {
	return v.isSet
}

func (v *NullableContentJAVAPackageResponseContentInner) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableContentJAVAPackageResponseContentInner(val *ContentJAVAPackageResponseContentInner) *NullableContentJAVAPackageResponseContentInner {
	return &NullableContentJAVAPackageResponseContentInner{value: val, isSet: true}
}

func (v NullableContentJAVAPackageResponseContentInner) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableContentJAVAPackageResponseContentInner) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


