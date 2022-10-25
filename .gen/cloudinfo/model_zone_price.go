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

// ZonePrice ZonePrice struct for displaying price information per zone
type ZonePrice struct {
	Price *float64 `json:"price,omitempty"`
	Zone *string `json:"zone,omitempty"`
}

// NewZonePrice instantiates a new ZonePrice object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewZonePrice() *ZonePrice {
	this := ZonePrice{}
	return &this
}

// NewZonePriceWithDefaults instantiates a new ZonePrice object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewZonePriceWithDefaults() *ZonePrice {
	this := ZonePrice{}
	return &this
}

// GetPrice returns the Price field value if set, zero value otherwise.
func (o *ZonePrice) GetPrice() float64 {
	if o == nil || o.Price == nil {
		var ret float64
		return ret
	}
	return *o.Price
}

// GetPriceOk returns a tuple with the Price field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ZonePrice) GetPriceOk() (*float64, bool) {
	if o == nil || o.Price == nil {
		return nil, false
	}
	return o.Price, true
}

// HasPrice returns a boolean if a field has been set.
func (o *ZonePrice) HasPrice() bool {
	if o != nil && o.Price != nil {
		return true
	}

	return false
}

// SetPrice gets a reference to the given float64 and assigns it to the Price field.
func (o *ZonePrice) SetPrice(v float64) {
	o.Price = &v
}

// GetZone returns the Zone field value if set, zero value otherwise.
func (o *ZonePrice) GetZone() string {
	if o == nil || o.Zone == nil {
		var ret string
		return ret
	}
	return *o.Zone
}

// GetZoneOk returns a tuple with the Zone field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ZonePrice) GetZoneOk() (*string, bool) {
	if o == nil || o.Zone == nil {
		return nil, false
	}
	return o.Zone, true
}

// HasZone returns a boolean if a field has been set.
func (o *ZonePrice) HasZone() bool {
	if o != nil && o.Zone != nil {
		return true
	}

	return false
}

// SetZone gets a reference to the given string and assigns it to the Zone field.
func (o *ZonePrice) SetZone(v string) {
	o.Zone = &v
}

func (o ZonePrice) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.Price != nil {
		toSerialize["price"] = o.Price
	}
	if o.Zone != nil {
		toSerialize["zone"] = o.Zone
	}
	return json.Marshal(toSerialize)
}

type NullableZonePrice struct {
	value *ZonePrice
	isSet bool
}

func (v NullableZonePrice) Get() *ZonePrice {
	return v.value
}

func (v *NullableZonePrice) Set(val *ZonePrice) {
	v.value = val
	v.isSet = true
}

func (v NullableZonePrice) IsSet() bool {
	return v.isSet
}

func (v *NullableZonePrice) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableZonePrice(val *ZonePrice) *NullableZonePrice {
	return &NullableZonePrice{value: val, isSet: true}
}

func (v NullableZonePrice) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableZonePrice) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


