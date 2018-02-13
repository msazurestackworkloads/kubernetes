package commerce

// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Code generated by Microsoft (R) AutoRest Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"encoding/json"
	"errors"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
	"net/http"
)

// AggregationGranularity enumerates the values for aggregation granularity.
type AggregationGranularity string

const (
	// Daily specifies the daily state for aggregation granularity.
	Daily AggregationGranularity = "Daily"
	// Hourly specifies the hourly state for aggregation granularity.
	Hourly AggregationGranularity = "Hourly"
)

// Name enumerates the values for name.
type Name string

const (
	// NameMonetaryCommitment specifies the name monetary commitment state for name.
	NameMonetaryCommitment Name = "Monetary Commitment"
	// NameMonetaryCredit specifies the name monetary credit state for name.
	NameMonetaryCredit Name = "Monetary Credit"
	// NameRecurringCharge specifies the name recurring charge state for name.
	NameRecurringCharge Name = "Recurring Charge"
)

// ErrorResponse is describes the format of Error response.
type ErrorResponse struct {
	Code    *string `json:"code,omitempty"`
	Message *string `json:"message,omitempty"`
}

// InfoField is key-value pairs of instance details in the legacy format.
type InfoField struct {
	Project *string `json:"project,omitempty"`
}

// MeterInfo is detailed information about the meter.
type MeterInfo struct {
	MeterID          *uuid.UUID           `json:"MeterId,omitempty"`
	MeterName        *string              `json:"MeterName,omitempty"`
	MeterCategory    *string              `json:"MeterCategory,omitempty"`
	MeterSubCategory *string              `json:"MeterSubCategory,omitempty"`
	Unit             *string              `json:"Unit,omitempty"`
	MeterTags        *[]string            `json:"MeterTags,omitempty"`
	MeterRegion      *string              `json:"MeterRegion,omitempty"`
	MeterRates       *map[string]*float64 `json:"MeterRates,omitempty"`
	EffectiveDate    *date.Time           `json:"EffectiveDate,omitempty"`
	IncludedQuantity *float64             `json:"IncludedQuantity,omitempty"`
}

// MonetaryCommitment is indicates that a monetary commitment is required for this offer
type MonetaryCommitment struct {
	EffectiveDate    *date.Time                   `json:"EffectiveDate,omitempty"`
	Name             Name                         `json:"Name,omitempty"`
	TieredDiscount   *map[string]*decimal.Decimal `json:"TieredDiscount,omitempty"`
	ExcludedMeterIds *[]uuid.UUID                 `json:"ExcludedMeterIds,omitempty"`
}

// MarshalJSON is the custom marshaler for MonetaryCommitment.
func (mc MonetaryCommitment) MarshalJSON() ([]byte, error) {
	mc.Name = NameMonetaryCommitment
	type Alias MonetaryCommitment
	return json.Marshal(&struct {
		Alias
	}{
		Alias: (Alias)(mc),
	})
}

// AsMonetaryCredit is the OfferTermInfo implementation for MonetaryCommitment.
func (mc MonetaryCommitment) AsMonetaryCredit() (*MonetaryCredit, bool) {
	return nil, false
}

// AsMonetaryCommitment is the OfferTermInfo implementation for MonetaryCommitment.
func (mc MonetaryCommitment) AsMonetaryCommitment() (*MonetaryCommitment, bool) {
	return &mc, true
}

// AsRecurringCharge is the OfferTermInfo implementation for MonetaryCommitment.
func (mc MonetaryCommitment) AsRecurringCharge() (*RecurringCharge, bool) {
	return nil, false
}

// MonetaryCredit is indicates that this is a monetary credit offer.
type MonetaryCredit struct {
	EffectiveDate    *date.Time       `json:"EffectiveDate,omitempty"`
	Name             Name             `json:"Name,omitempty"`
	Credit           *decimal.Decimal `json:"Credit,omitempty"`
	ExcludedMeterIds *[]uuid.UUID     `json:"ExcludedMeterIds,omitempty"`
}

// MarshalJSON is the custom marshaler for MonetaryCredit.
func (mc MonetaryCredit) MarshalJSON() ([]byte, error) {
	mc.Name = NameMonetaryCredit
	type Alias MonetaryCredit
	return json.Marshal(&struct {
		Alias
	}{
		Alias: (Alias)(mc),
	})
}

// AsMonetaryCredit is the OfferTermInfo implementation for MonetaryCredit.
func (mc MonetaryCredit) AsMonetaryCredit() (*MonetaryCredit, bool) {
	return &mc, true
}

// AsMonetaryCommitment is the OfferTermInfo implementation for MonetaryCredit.
func (mc MonetaryCredit) AsMonetaryCommitment() (*MonetaryCommitment, bool) {
	return nil, false
}

// AsRecurringCharge is the OfferTermInfo implementation for MonetaryCredit.
func (mc MonetaryCredit) AsRecurringCharge() (*RecurringCharge, bool) {
	return nil, false
}

// OfferTermInfo is describes the offer term.
type OfferTermInfo interface {
	AsMonetaryCredit() (*MonetaryCredit, bool)
	AsMonetaryCommitment() (*MonetaryCommitment, bool)
	AsRecurringCharge() (*RecurringCharge, bool)
}

func unmarshalOfferTermInfo(body []byte) (OfferTermInfo, error) {
	var m map[string]interface{}
	err := json.Unmarshal(body, &m)
	if err != nil {
		return nil, err
	}

	switch m["Name"] {
	case string(NameMonetaryCredit):
		var mc MonetaryCredit
		err := json.Unmarshal(body, &mc)
		return mc, err
	case string(NameMonetaryCommitment):
		var mc MonetaryCommitment
		err := json.Unmarshal(body, &mc)
		return mc, err
	case string(NameRecurringCharge):
		var rc RecurringCharge
		err := json.Unmarshal(body, &rc)
		return rc, err
	default:
		return nil, errors.New("Unsupported type")
	}
}
func unmarshalOfferTermInfoArray(body []byte) ([]OfferTermInfo, error) {
	var rawMessages []*json.RawMessage
	err := json.Unmarshal(body, &rawMessages)
	if err != nil {
		return nil, err
	}

	otiArray := make([]OfferTermInfo, len(rawMessages))

	for index, rawMessage := range rawMessages {
		oti, err := unmarshalOfferTermInfo(*rawMessage)
		if err != nil {
			return nil, err
		}
		otiArray[index] = oti
	}
	return otiArray, nil
}

// RateCardQueryParameters is parameters that are used in the odata $filter query parameter for providing RateCard
// information.
type RateCardQueryParameters struct {
	OfferDurableID *string `json:"OfferDurableId,omitempty"`
	Currency       *string `json:"Currency,omitempty"`
	Locale         *string `json:"Locale,omitempty"`
	RegionInfo     *string `json:"RegionInfo,omitempty"`
}

// RecurringCharge is indicates a recurring charge is present for this offer.
type RecurringCharge struct {
	EffectiveDate   *date.Time `json:"EffectiveDate,omitempty"`
	Name            Name       `json:"Name,omitempty"`
	RecurringCharge *int32     `json:"RecurringCharge,omitempty"`
}

// MarshalJSON is the custom marshaler for RecurringCharge.
func (rc RecurringCharge) MarshalJSON() ([]byte, error) {
	rc.Name = NameRecurringCharge
	type Alias RecurringCharge
	return json.Marshal(&struct {
		Alias
	}{
		Alias: (Alias)(rc),
	})
}

// AsMonetaryCredit is the OfferTermInfo implementation for RecurringCharge.
func (rc RecurringCharge) AsMonetaryCredit() (*MonetaryCredit, bool) {
	return nil, false
}

// AsMonetaryCommitment is the OfferTermInfo implementation for RecurringCharge.
func (rc RecurringCharge) AsMonetaryCommitment() (*MonetaryCommitment, bool) {
	return nil, false
}

// AsRecurringCharge is the OfferTermInfo implementation for RecurringCharge.
func (rc RecurringCharge) AsRecurringCharge() (*RecurringCharge, bool) {
	return &rc, true
}

// ResourceRateCardInfo is price and Metadata information for resources
type ResourceRateCardInfo struct {
	autorest.Response `json:"-"`
	Currency          *string          `json:"Currency,omitempty"`
	Locale            *string          `json:"Locale,omitempty"`
	IsTaxIncluded     *bool            `json:"IsTaxIncluded,omitempty"`
	OfferTerms        *[]OfferTermInfo `json:"OfferTerms,omitempty"`
	Meters            *[]MeterInfo     `json:"Meters,omitempty"`
}

// UnmarshalJSON is the custom unmarshaler for ResourceRateCardInfo struct.
func (rrci *ResourceRateCardInfo) UnmarshalJSON(body []byte) error {
	var m map[string]*json.RawMessage
	err := json.Unmarshal(body, &m)
	if err != nil {
		return err
	}
	var v *json.RawMessage

	v = m["Currency"]
	if v != nil {
		var currency string
		err = json.Unmarshal(*m["Currency"], &currency)
		if err != nil {
			return err
		}
		rrci.Currency = &currency
	}

	v = m["Locale"]
	if v != nil {
		var locale string
		err = json.Unmarshal(*m["Locale"], &locale)
		if err != nil {
			return err
		}
		rrci.Locale = &locale
	}

	v = m["IsTaxIncluded"]
	if v != nil {
		var isTaxIncluded bool
		err = json.Unmarshal(*m["IsTaxIncluded"], &isTaxIncluded)
		if err != nil {
			return err
		}
		rrci.IsTaxIncluded = &isTaxIncluded
	}

	v = m["OfferTerms"]
	if v != nil {
		offerTerms, err := unmarshalOfferTermInfoArray(*m["OfferTerms"])
		if err != nil {
			return err
		}
		rrci.OfferTerms = &offerTerms
	}

	v = m["Meters"]
	if v != nil {
		var meters []MeterInfo
		err = json.Unmarshal(*m["Meters"], &meters)
		if err != nil {
			return err
		}
		rrci.Meters = &meters
	}

	return nil
}

// UsageAggregation is describes the usageAggregation.
type UsageAggregation struct {
	ID           *string `json:"id,omitempty"`
	Name         *string `json:"name,omitempty"`
	Type         *string `json:"type,omitempty"`
	*UsageSample `json:"properties,omitempty"`
}

// UsageAggregationListResult is the Get UsageAggregates operation response.
type UsageAggregationListResult struct {
	autorest.Response `json:"-"`
	Value             *[]UsageAggregation `json:"value,omitempty"`
	NextLink          *string             `json:"nextLink,omitempty"`
}

// UsageAggregationListResultPreparer prepares a request to retrieve the next set of results. It returns
// nil if no more results exist.
func (client UsageAggregationListResult) UsageAggregationListResultPreparer() (*http.Request, error) {
	if client.NextLink == nil || len(to.String(client.NextLink)) <= 0 {
		return nil, nil
	}
	return autorest.Prepare(&http.Request{},
		autorest.AsJSON(),
		autorest.AsGet(),
		autorest.WithBaseURL(to.String(client.NextLink)))
}

// UsageSample is describes a sample of the usageAggregation.
type UsageSample struct {
	SubscriptionID   *uuid.UUID              `json:"subscriptionId,omitempty"`
	MeterID          *string                 `json:"meterId,omitempty"`
	UsageStartTime   *date.Time              `json:"usageStartTime,omitempty"`
	UsageEndTime     *date.Time              `json:"usageEndTime,omitempty"`
	Quantity         *map[string]interface{} `json:"quantity,omitempty"`
	Unit             *string                 `json:"unit,omitempty"`
	MeterName        *string                 `json:"meterName,omitempty"`
	MeterCategory    *string                 `json:"meterCategory,omitempty"`
	MeterSubCategory *string                 `json:"meterSubCategory,omitempty"`
	MeterRegion      *string                 `json:"meterRegion,omitempty"`
	InfoFields       *InfoField              `json:"infoFields,omitempty"`
	InstanceData     *string                 `json:"instanceData,omitempty"`
}
