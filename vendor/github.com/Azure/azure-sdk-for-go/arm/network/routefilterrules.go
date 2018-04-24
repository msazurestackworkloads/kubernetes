package network

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
// Code generated by Microsoft (R) AutoRest Code Generator 1.0.1.0
// Changes may cause incorrect behavior and will be lost if the code is
// regenerated.

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"net/http"
)

// RouteFilterRulesClient is the composite Swagger for Network Client
type RouteFilterRulesClient struct {
	ManagementClient
}

// NewRouteFilterRulesClient creates an instance of the RouteFilterRulesClient
// client.
func NewRouteFilterRulesClient(subscriptionID string) RouteFilterRulesClient {
	return NewRouteFilterRulesClientWithBaseURI(DefaultBaseURI, subscriptionID)
}

// NewRouteFilterRulesClientWithBaseURI creates an instance of the
// RouteFilterRulesClient client.
func NewRouteFilterRulesClientWithBaseURI(baseURI string, subscriptionID string) RouteFilterRulesClient {
	return RouteFilterRulesClient{NewWithBaseURI(baseURI, subscriptionID)}
}

// CreateOrUpdate creates or updates a route in the specified route filter.
// This method may poll for completion. Polling can be canceled by passing the
// cancel channel argument. The channel will be used to cancel polling and any
// outstanding HTTP requests.
//
// resourceGroupName is the name of the resource group. routeFilterName is the
// name of the route filter. ruleName is the name of the route filter rule.
// routeFilterRuleParameters is parameters supplied to the create or update
// route filter rule operation.
func (client RouteFilterRulesClient) CreateOrUpdate(resourceGroupName string, routeFilterName string, ruleName string, routeFilterRuleParameters RouteFilterRule, cancel <-chan struct{}) (<-chan RouteFilterRule, <-chan error) {
	resultChan := make(chan RouteFilterRule, 1)
	errChan := make(chan error, 1)
	if err := validation.Validate([]validation.Validation{
		{TargetValue: routeFilterRuleParameters,
			Constraints: []validation.Constraint{{Target: "routeFilterRuleParameters.RouteFilterRulePropertiesFormat", Name: validation.Null, Rule: false,
				Chain: []validation.Constraint{{Target: "routeFilterRuleParameters.RouteFilterRulePropertiesFormat.RouteFilterRuleType", Name: validation.Null, Rule: true, Chain: nil},
					{Target: "routeFilterRuleParameters.RouteFilterRulePropertiesFormat.Communities", Name: validation.Null, Rule: true, Chain: nil},
				}}}}}); err != nil {
		errChan <- validation.NewErrorWithValidationError(err, "network.RouteFilterRulesClient", "CreateOrUpdate")
		close(errChan)
		close(resultChan)
		return resultChan, errChan
	}

	go func() {
		var err error
		var result RouteFilterRule
		defer func() {
			resultChan <- result
			errChan <- err
			close(resultChan)
			close(errChan)
		}()
		req, err := client.CreateOrUpdatePreparer(resourceGroupName, routeFilterName, ruleName, routeFilterRuleParameters, cancel)
		if err != nil {
			err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "CreateOrUpdate", nil, "Failure preparing request")
			return
		}

		resp, err := client.CreateOrUpdateSender(req)
		if err != nil {
			result.Response = autorest.Response{Response: resp}
			err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "CreateOrUpdate", resp, "Failure sending request")
			return
		}

		result, err = client.CreateOrUpdateResponder(resp)
		if err != nil {
			err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "CreateOrUpdate", resp, "Failure responding to request")
		}
	}()
	return resultChan, errChan
}

// CreateOrUpdatePreparer prepares the CreateOrUpdate request.
func (client RouteFilterRulesClient) CreateOrUpdatePreparer(resourceGroupName string, routeFilterName string, ruleName string, routeFilterRuleParameters RouteFilterRule, cancel <-chan struct{}) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"routeFilterName":   autorest.Encode("path", routeFilterName),
		"ruleName":          autorest.Encode("path", ruleName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2015-06-15"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsJSON(),
		autorest.AsPut(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}/routeFilterRules/{ruleName}", pathParameters),
		autorest.WithJSON(routeFilterRuleParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{Cancel: cancel})
}

// CreateOrUpdateSender sends the CreateOrUpdate request. The method will close the
// http.Response Body if it receives an error.
func (client RouteFilterRulesClient) CreateOrUpdateSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client,
		req,
		azure.DoPollForAsynchronous(client.PollingDelay))
}

// CreateOrUpdateResponder handles the response to the CreateOrUpdate request. The method always
// closes the http.Response Body.
func (client RouteFilterRulesClient) CreateOrUpdateResponder(resp *http.Response) (result RouteFilterRule, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusCreated),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// Delete deletes the specified rule from a route filter. This method may poll
// for completion. Polling can be canceled by passing the cancel channel
// argument. The channel will be used to cancel polling and any outstanding
// HTTP requests.
//
// resourceGroupName is the name of the resource group. routeFilterName is the
// name of the route filter. ruleName is the name of the rule.
func (client RouteFilterRulesClient) Delete(resourceGroupName string, routeFilterName string, ruleName string, cancel <-chan struct{}) (<-chan autorest.Response, <-chan error) {
	resultChan := make(chan autorest.Response, 1)
	errChan := make(chan error, 1)
	go func() {
		var err error
		var result autorest.Response
		defer func() {
			resultChan <- result
			errChan <- err
			close(resultChan)
			close(errChan)
		}()
		req, err := client.DeletePreparer(resourceGroupName, routeFilterName, ruleName, cancel)
		if err != nil {
			err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Delete", nil, "Failure preparing request")
			return
		}

		resp, err := client.DeleteSender(req)
		if err != nil {
			result.Response = resp
			err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Delete", resp, "Failure sending request")
			return
		}

		result, err = client.DeleteResponder(resp)
		if err != nil {
			err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Delete", resp, "Failure responding to request")
		}
	}()
	return resultChan, errChan
}

// DeletePreparer prepares the Delete request.
func (client RouteFilterRulesClient) DeletePreparer(resourceGroupName string, routeFilterName string, ruleName string, cancel <-chan struct{}) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"routeFilterName":   autorest.Encode("path", routeFilterName),
		"ruleName":          autorest.Encode("path", ruleName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2015-06-15"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsDelete(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}/routeFilterRules/{ruleName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{Cancel: cancel})
}

// DeleteSender sends the Delete request. The method will close the
// http.Response Body if it receives an error.
func (client RouteFilterRulesClient) DeleteSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client,
		req,
		azure.DoPollForAsynchronous(client.PollingDelay))
}

// DeleteResponder handles the response to the Delete request. The method always
// closes the http.Response Body.
func (client RouteFilterRulesClient) DeleteResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusAccepted, http.StatusOK, http.StatusNoContent),
		autorest.ByClosing())
	result.Response = resp
	return
}

// Get gets the specified rule from a route filter.
//
// resourceGroupName is the name of the resource group. routeFilterName is the
// name of the route filter. ruleName is the name of the rule.
func (client RouteFilterRulesClient) Get(resourceGroupName string, routeFilterName string, ruleName string) (result RouteFilterRule, err error) {
	req, err := client.GetPreparer(resourceGroupName, routeFilterName, ruleName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Get", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Get", resp, "Failure sending request")
		return
	}

	result, err = client.GetResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Get", resp, "Failure responding to request")
	}

	return
}

// GetPreparer prepares the Get request.
func (client RouteFilterRulesClient) GetPreparer(resourceGroupName string, routeFilterName string, ruleName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"routeFilterName":   autorest.Encode("path", routeFilterName),
		"ruleName":          autorest.Encode("path", ruleName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2015-06-15"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}/routeFilterRules/{ruleName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{})
}

// GetSender sends the Get request. The method will close the
// http.Response Body if it receives an error.
func (client RouteFilterRulesClient) GetSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req)
}

// GetResponder handles the response to the Get request. The method always
// closes the http.Response Body.
func (client RouteFilterRulesClient) GetResponder(resp *http.Response) (result RouteFilterRule, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// ListByRouteFilter gets all RouteFilterRules in a route filter.
//
// resourceGroupName is the name of the resource group. routeFilterName is the
// name of the route filter.
func (client RouteFilterRulesClient) ListByRouteFilter(resourceGroupName string, routeFilterName string) (result RouteFilterRuleListResult, err error) {
	req, err := client.ListByRouteFilterPreparer(resourceGroupName, routeFilterName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "ListByRouteFilter", nil, "Failure preparing request")
		return
	}

	resp, err := client.ListByRouteFilterSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "ListByRouteFilter", resp, "Failure sending request")
		return
	}

	result, err = client.ListByRouteFilterResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "ListByRouteFilter", resp, "Failure responding to request")
	}

	return
}

// ListByRouteFilterPreparer prepares the ListByRouteFilter request.
func (client RouteFilterRulesClient) ListByRouteFilterPreparer(resourceGroupName string, routeFilterName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"routeFilterName":   autorest.Encode("path", routeFilterName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2015-06-15"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}/routeFilterRules", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{})
}

// ListByRouteFilterSender sends the ListByRouteFilter request. The method will close the
// http.Response Body if it receives an error.
func (client RouteFilterRulesClient) ListByRouteFilterSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req)
}

// ListByRouteFilterResponder handles the response to the ListByRouteFilter request. The method always
// closes the http.Response Body.
func (client RouteFilterRulesClient) ListByRouteFilterResponder(resp *http.Response) (result RouteFilterRuleListResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// ListByRouteFilterNextResults retrieves the next set of results, if any.
func (client RouteFilterRulesClient) ListByRouteFilterNextResults(lastResults RouteFilterRuleListResult) (result RouteFilterRuleListResult, err error) {
	req, err := lastResults.RouteFilterRuleListResultPreparer()
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "ListByRouteFilter", nil, "Failure preparing next results request")
	}
	if req == nil {
		return
	}

	resp, err := client.ListByRouteFilterSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "ListByRouteFilter", resp, "Failure sending next results request")
	}

	result, err = client.ListByRouteFilterResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "ListByRouteFilter", resp, "Failure responding to next results request")
	}

	return
}

// Update updates a route in the specified route filter. This method may poll
// for completion. Polling can be canceled by passing the cancel channel
// argument. The channel will be used to cancel polling and any outstanding
// HTTP requests.
//
// resourceGroupName is the name of the resource group. routeFilterName is the
// name of the route filter. ruleName is the name of the route filter rule.
// routeFilterRuleParameters is parameters supplied to the update route filter
// rule operation.
func (client RouteFilterRulesClient) Update(resourceGroupName string, routeFilterName string, ruleName string, routeFilterRuleParameters PatchRouteFilterRule, cancel <-chan struct{}) (<-chan RouteFilterRule, <-chan error) {
	resultChan := make(chan RouteFilterRule, 1)
	errChan := make(chan error, 1)
	go func() {
		var err error
		var result RouteFilterRule
		defer func() {
			resultChan <- result
			errChan <- err
			close(resultChan)
			close(errChan)
		}()
		req, err := client.UpdatePreparer(resourceGroupName, routeFilterName, ruleName, routeFilterRuleParameters, cancel)
		if err != nil {
			err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Update", nil, "Failure preparing request")
			return
		}

		resp, err := client.UpdateSender(req)
		if err != nil {
			result.Response = autorest.Response{Response: resp}
			err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Update", resp, "Failure sending request")
			return
		}

		result, err = client.UpdateResponder(resp)
		if err != nil {
			err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Update", resp, "Failure responding to request")
		}
	}()
	return resultChan, errChan
}

// UpdatePreparer prepares the Update request.
func (client RouteFilterRulesClient) UpdatePreparer(resourceGroupName string, routeFilterName string, ruleName string, routeFilterRuleParameters PatchRouteFilterRule, cancel <-chan struct{}) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"routeFilterName":   autorest.Encode("path", routeFilterName),
		"ruleName":          autorest.Encode("path", ruleName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2015-06-15"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsJSON(),
		autorest.AsPatch(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}/routeFilterRules/{ruleName}", pathParameters),
		autorest.WithJSON(routeFilterRuleParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{Cancel: cancel})
}

// UpdateSender sends the Update request. The method will close the
// http.Response Body if it receives an error.
func (client RouteFilterRulesClient) UpdateSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client,
		req,
		azure.DoPollForAsynchronous(client.PollingDelay))
}

// UpdateResponder handles the response to the Update request. The method always
// closes the http.Response Body.
func (client RouteFilterRulesClient) UpdateResponder(resp *http.Response) (result RouteFilterRule, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}
