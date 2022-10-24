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

// ContentPackageResponseContentInner struct for ContentPackageResponseContentInner
type ContentPackageResponseContentInner struct {
	Package *string `json:"package,omitempty"`
	Version *string `json:"version,omitempty"`
	Size *string `json:"size,omitempty"`
	Type *string `json:"type,omitempty"`
	Origin *string `json:"origin,omitempty"`
	// Deprecated in favor of the 'licenses' field\"
	License *string `json:"license,omitempty"`
	Licenses []string `json:"licenses,omitempty"`
	Location *string `json:"location,omitempty"`
	// A list of Common Platform Enumerations that may uniquely identify the package
	Cpes []string `json:"cpes,omitempty"`
}

// NewContentPackageResponseContentInner instantiates a new ContentPackageResponseContentInner object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewContentPackageResponseContentInner() *ContentPackageResponseContentInner {
	this := ContentPackageResponseContentInner{}
	return &this
}

// NewContentPackageResponseContentInnerWithDefaults instantiates a new ContentPackageResponseContentInner object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewContentPackageResponseContentInnerWithDefaults() *ContentPackageResponseContentInner {
	this := ContentPackageResponseContentInner{}
	return &this
}

// GetPackage returns the Package field value if set, zero value otherwise.
func (o *ContentPackageResponseContentInner) GetPackage() string {
	if o == nil || o.Package == nil {
		var ret string
		return ret
	}
	return *o.Package
}

// GetPackageOk returns a tuple with the Package field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentPackageResponseContentInner) GetPackageOk() (*string, bool) {
	if o == nil || o.Package == nil {
		return nil, false
	}
	return o.Package, true
}

// HasPackage returns a boolean if a field has been set.
func (o *ContentPackageResponseContentInner) HasPackage() bool {
	if o != nil && o.Package != nil {
		return true
	}

	return false
}

// SetPackage gets a reference to the given string and assigns it to the Package field.
func (o *ContentPackageResponseContentInner) SetPackage(v string) {
	o.Package = &v
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *ContentPackageResponseContentInner) GetVersion() string {
	if o == nil || o.Version == nil {
		var ret string
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentPackageResponseContentInner) GetVersionOk() (*string, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *ContentPackageResponseContentInner) HasVersion() bool {
	if o != nil && o.Version != nil {
		return true
	}

	return false
}

// SetVersion gets a reference to the given string and assigns it to the Version field.
func (o *ContentPackageResponseContentInner) SetVersion(v string) {
	o.Version = &v
}

// GetSize returns the Size field value if set, zero value otherwise.
func (o *ContentPackageResponseContentInner) GetSize() string {
	if o == nil || o.Size == nil {
		var ret string
		return ret
	}
	return *o.Size
}

// GetSizeOk returns a tuple with the Size field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentPackageResponseContentInner) GetSizeOk() (*string, bool) {
	if o == nil || o.Size == nil {
		return nil, false
	}
	return o.Size, true
}

// HasSize returns a boolean if a field has been set.
func (o *ContentPackageResponseContentInner) HasSize() bool {
	if o != nil && o.Size != nil {
		return true
	}

	return false
}

// SetSize gets a reference to the given string and assigns it to the Size field.
func (o *ContentPackageResponseContentInner) SetSize(v string) {
	o.Size = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *ContentPackageResponseContentInner) GetType() string {
	if o == nil || o.Type == nil {
		var ret string
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentPackageResponseContentInner) GetTypeOk() (*string, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *ContentPackageResponseContentInner) HasType() bool {
	if o != nil && o.Type != nil {
		return true
	}

	return false
}

// SetType gets a reference to the given string and assigns it to the Type field.
func (o *ContentPackageResponseContentInner) SetType(v string) {
	o.Type = &v
}

// GetOrigin returns the Origin field value if set, zero value otherwise.
func (o *ContentPackageResponseContentInner) GetOrigin() string {
	if o == nil || o.Origin == nil {
		var ret string
		return ret
	}
	return *o.Origin
}

// GetOriginOk returns a tuple with the Origin field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentPackageResponseContentInner) GetOriginOk() (*string, bool) {
	if o == nil || o.Origin == nil {
		return nil, false
	}
	return o.Origin, true
}

// HasOrigin returns a boolean if a field has been set.
func (o *ContentPackageResponseContentInner) HasOrigin() bool {
	if o != nil && o.Origin != nil {
		return true
	}

	return false
}

// SetOrigin gets a reference to the given string and assigns it to the Origin field.
func (o *ContentPackageResponseContentInner) SetOrigin(v string) {
	o.Origin = &v
}

// GetLicense returns the License field value if set, zero value otherwise.
func (o *ContentPackageResponseContentInner) GetLicense() string {
	if o == nil || o.License == nil {
		var ret string
		return ret
	}
	return *o.License
}

// GetLicenseOk returns a tuple with the License field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentPackageResponseContentInner) GetLicenseOk() (*string, bool) {
	if o == nil || o.License == nil {
		return nil, false
	}
	return o.License, true
}

// HasLicense returns a boolean if a field has been set.
func (o *ContentPackageResponseContentInner) HasLicense() bool {
	if o != nil && o.License != nil {
		return true
	}

	return false
}

// SetLicense gets a reference to the given string and assigns it to the License field.
func (o *ContentPackageResponseContentInner) SetLicense(v string) {
	o.License = &v
}

// GetLicenses returns the Licenses field value if set, zero value otherwise.
func (o *ContentPackageResponseContentInner) GetLicenses() []string {
	if o == nil || o.Licenses == nil {
		var ret []string
		return ret
	}
	return o.Licenses
}

// GetLicensesOk returns a tuple with the Licenses field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentPackageResponseContentInner) GetLicensesOk() ([]string, bool) {
	if o == nil || o.Licenses == nil {
		return nil, false
	}
	return o.Licenses, true
}

// HasLicenses returns a boolean if a field has been set.
func (o *ContentPackageResponseContentInner) HasLicenses() bool {
	if o != nil && o.Licenses != nil {
		return true
	}

	return false
}

// SetLicenses gets a reference to the given []string and assigns it to the Licenses field.
func (o *ContentPackageResponseContentInner) SetLicenses(v []string) {
	o.Licenses = v
}

// GetLocation returns the Location field value if set, zero value otherwise.
func (o *ContentPackageResponseContentInner) GetLocation() string {
	if o == nil || o.Location == nil {
		var ret string
		return ret
	}
	return *o.Location
}

// GetLocationOk returns a tuple with the Location field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentPackageResponseContentInner) GetLocationOk() (*string, bool) {
	if o == nil || o.Location == nil {
		return nil, false
	}
	return o.Location, true
}

// HasLocation returns a boolean if a field has been set.
func (o *ContentPackageResponseContentInner) HasLocation() bool {
	if o != nil && o.Location != nil {
		return true
	}

	return false
}

// SetLocation gets a reference to the given string and assigns it to the Location field.
func (o *ContentPackageResponseContentInner) SetLocation(v string) {
	o.Location = &v
}

// GetCpes returns the Cpes field value if set, zero value otherwise.
func (o *ContentPackageResponseContentInner) GetCpes() []string {
	if o == nil || o.Cpes == nil {
		var ret []string
		return ret
	}
	return o.Cpes
}

// GetCpesOk returns a tuple with the Cpes field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ContentPackageResponseContentInner) GetCpesOk() ([]string, bool) {
	if o == nil || o.Cpes == nil {
		return nil, false
	}
	return o.Cpes, true
}

// HasCpes returns a boolean if a field has been set.
func (o *ContentPackageResponseContentInner) HasCpes() bool {
	if o != nil && o.Cpes != nil {
		return true
	}

	return false
}

// SetCpes gets a reference to the given []string and assigns it to the Cpes field.
func (o *ContentPackageResponseContentInner) SetCpes(v []string) {
	o.Cpes = v
}

func (o ContentPackageResponseContentInner) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.Package != nil {
		toSerialize["package"] = o.Package
	}
	if o.Version != nil {
		toSerialize["version"] = o.Version
	}
	if o.Size != nil {
		toSerialize["size"] = o.Size
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}
	if o.Origin != nil {
		toSerialize["origin"] = o.Origin
	}
	if o.License != nil {
		toSerialize["license"] = o.License
	}
	if o.Licenses != nil {
		toSerialize["licenses"] = o.Licenses
	}
	if o.Location != nil {
		toSerialize["location"] = o.Location
	}
	if o.Cpes != nil {
		toSerialize["cpes"] = o.Cpes
	}
	return json.Marshal(toSerialize)
}

type NullableContentPackageResponseContentInner struct {
	value *ContentPackageResponseContentInner
	isSet bool
}

func (v NullableContentPackageResponseContentInner) Get() *ContentPackageResponseContentInner {
	return v.value
}

func (v *NullableContentPackageResponseContentInner) Set(val *ContentPackageResponseContentInner) {
	v.value = val
	v.isSet = true
}

func (v NullableContentPackageResponseContentInner) IsSet() bool {
	return v.isSet
}

func (v *NullableContentPackageResponseContentInner) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableContentPackageResponseContentInner(val *ContentPackageResponseContentInner) *NullableContentPackageResponseContentInner {
	return &NullableContentPackageResponseContentInner{value: val, isSet: true}
}

func (v NullableContentPackageResponseContentInner) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableContentPackageResponseContentInner) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


