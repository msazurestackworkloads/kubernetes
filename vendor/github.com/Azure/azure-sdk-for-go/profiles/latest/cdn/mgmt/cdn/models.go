// +build go1.9

// Copyright 2017 Microsoft Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This code was auto-generated by:
// github.com/Azure/azure-sdk-for-go/tools/profileBuilder
// commit ID: 2014fbbf031942474ad27a5a66dffaed5347f3fb

package cdn

import original "github.com/Azure/azure-sdk-for-go/services/cdn/mgmt/2017-04-02/cdn"

const (
	DefaultBaseURI = original.DefaultBaseURI
)

type BaseClient = original.BaseClient
type CustomDomainsClient = original.CustomDomainsClient
type EdgeNodesClient = original.EdgeNodesClient
type EndpointsClient = original.EndpointsClient
type CustomDomainResourceState = original.CustomDomainResourceState

const (
	Active   CustomDomainResourceState = original.Active
	Creating CustomDomainResourceState = original.Creating
	Deleting CustomDomainResourceState = original.Deleting
)

type CustomHTTPSProvisioningState = original.CustomHTTPSProvisioningState

const (
	Disabled  CustomHTTPSProvisioningState = original.Disabled
	Disabling CustomHTTPSProvisioningState = original.Disabling
	Enabled   CustomHTTPSProvisioningState = original.Enabled
	Enabling  CustomHTTPSProvisioningState = original.Enabling
	Failed    CustomHTTPSProvisioningState = original.Failed
)

type CustomHTTPSProvisioningSubstate = original.CustomHTTPSProvisioningSubstate

const (
	CertificateDeleted                            CustomHTTPSProvisioningSubstate = original.CertificateDeleted
	CertificateDeployed                           CustomHTTPSProvisioningSubstate = original.CertificateDeployed
	DeletingCertificate                           CustomHTTPSProvisioningSubstate = original.DeletingCertificate
	DeployingCertificate                          CustomHTTPSProvisioningSubstate = original.DeployingCertificate
	DomainControlValidationRequestApproved        CustomHTTPSProvisioningSubstate = original.DomainControlValidationRequestApproved
	DomainControlValidationRequestRejected        CustomHTTPSProvisioningSubstate = original.DomainControlValidationRequestRejected
	DomainControlValidationRequestTimedOut        CustomHTTPSProvisioningSubstate = original.DomainControlValidationRequestTimedOut
	IssuingCertificate                            CustomHTTPSProvisioningSubstate = original.IssuingCertificate
	PendingDomainControlValidationREquestApproval CustomHTTPSProvisioningSubstate = original.PendingDomainControlValidationREquestApproval
	SubmittingDomainControlValidationRequest      CustomHTTPSProvisioningSubstate = original.SubmittingDomainControlValidationRequest
)

type EndpointResourceState = original.EndpointResourceState

const (
	EndpointResourceStateCreating EndpointResourceState = original.EndpointResourceStateCreating
	EndpointResourceStateDeleting EndpointResourceState = original.EndpointResourceStateDeleting
	EndpointResourceStateRunning  EndpointResourceState = original.EndpointResourceStateRunning
	EndpointResourceStateStarting EndpointResourceState = original.EndpointResourceStateStarting
	EndpointResourceStateStopped  EndpointResourceState = original.EndpointResourceStateStopped
	EndpointResourceStateStopping EndpointResourceState = original.EndpointResourceStateStopping
)

type GeoFilterActions = original.GeoFilterActions

const (
	Allow GeoFilterActions = original.Allow
	Block GeoFilterActions = original.Block
)

type OptimizationType = original.OptimizationType

const (
	DynamicSiteAcceleration     OptimizationType = original.DynamicSiteAcceleration
	GeneralMediaStreaming       OptimizationType = original.GeneralMediaStreaming
	GeneralWebDelivery          OptimizationType = original.GeneralWebDelivery
	LargeFileDownload           OptimizationType = original.LargeFileDownload
	VideoOnDemandMediaStreaming OptimizationType = original.VideoOnDemandMediaStreaming
)

type OriginResourceState = original.OriginResourceState

const (
	OriginResourceStateActive   OriginResourceState = original.OriginResourceStateActive
	OriginResourceStateCreating OriginResourceState = original.OriginResourceStateCreating
	OriginResourceStateDeleting OriginResourceState = original.OriginResourceStateDeleting
)

type ProfileResourceState = original.ProfileResourceState

const (
	ProfileResourceStateActive   ProfileResourceState = original.ProfileResourceStateActive
	ProfileResourceStateCreating ProfileResourceState = original.ProfileResourceStateCreating
	ProfileResourceStateDeleting ProfileResourceState = original.ProfileResourceStateDeleting
	ProfileResourceStateDisabled ProfileResourceState = original.ProfileResourceStateDisabled
)

type QueryStringCachingBehavior = original.QueryStringCachingBehavior

const (
	BypassCaching     QueryStringCachingBehavior = original.BypassCaching
	IgnoreQueryString QueryStringCachingBehavior = original.IgnoreQueryString
	NotSet            QueryStringCachingBehavior = original.NotSet
	UseQueryString    QueryStringCachingBehavior = original.UseQueryString
)

type ResourceType = original.ResourceType

const (
	MicrosoftCdnProfilesEndpoints ResourceType = original.MicrosoftCdnProfilesEndpoints
)

type SkuName = original.SkuName

const (
	CustomVerizon    SkuName = original.CustomVerizon
	PremiumVerizon   SkuName = original.PremiumVerizon
	StandardAkamai   SkuName = original.StandardAkamai
	StandardChinaCdn SkuName = original.StandardChinaCdn
	StandardVerizon  SkuName = original.StandardVerizon
)

type CheckNameAvailabilityInput = original.CheckNameAvailabilityInput
type CheckNameAvailabilityOutput = original.CheckNameAvailabilityOutput
type CidrIPAddress = original.CidrIPAddress
type CustomDomain = original.CustomDomain
type CustomDomainListResult = original.CustomDomainListResult
type CustomDomainListResultIterator = original.CustomDomainListResultIterator
type CustomDomainListResultPage = original.CustomDomainListResultPage
type CustomDomainParameters = original.CustomDomainParameters
type CustomDomainProperties = original.CustomDomainProperties
type CustomDomainPropertiesParameters = original.CustomDomainPropertiesParameters
type CustomDomainsCreateFuture = original.CustomDomainsCreateFuture
type CustomDomainsDeleteFuture = original.CustomDomainsDeleteFuture
type DeepCreatedOrigin = original.DeepCreatedOrigin
type DeepCreatedOriginProperties = original.DeepCreatedOriginProperties
type EdgeNode = original.EdgeNode
type EdgeNodeProperties = original.EdgeNodeProperties
type EdgenodeResult = original.EdgenodeResult
type EdgenodeResultIterator = original.EdgenodeResultIterator
type EdgenodeResultPage = original.EdgenodeResultPage
type Endpoint = original.Endpoint
type EndpointListResult = original.EndpointListResult
type EndpointListResultIterator = original.EndpointListResultIterator
type EndpointListResultPage = original.EndpointListResultPage
type EndpointProperties = original.EndpointProperties
type EndpointPropertiesUpdateParameters = original.EndpointPropertiesUpdateParameters
type EndpointsCreateFuture = original.EndpointsCreateFuture
type EndpointsDeleteFuture = original.EndpointsDeleteFuture
type EndpointsLoadContentFuture = original.EndpointsLoadContentFuture
type EndpointsPurgeContentFuture = original.EndpointsPurgeContentFuture
type EndpointsStartFuture = original.EndpointsStartFuture
type EndpointsStopFuture = original.EndpointsStopFuture
type EndpointsUpdateFuture = original.EndpointsUpdateFuture
type EndpointUpdateParameters = original.EndpointUpdateParameters
type ErrorResponse = original.ErrorResponse
type GeoFilter = original.GeoFilter
type IPAddressGroup = original.IPAddressGroup
type LoadParameters = original.LoadParameters
type Operation = original.Operation
type OperationDisplay = original.OperationDisplay
type OperationsListResult = original.OperationsListResult
type OperationsListResultIterator = original.OperationsListResultIterator
type OperationsListResultPage = original.OperationsListResultPage
type Origin = original.Origin
type OriginListResult = original.OriginListResult
type OriginListResultIterator = original.OriginListResultIterator
type OriginListResultPage = original.OriginListResultPage
type OriginProperties = original.OriginProperties
type OriginPropertiesParameters = original.OriginPropertiesParameters
type OriginsUpdateFuture = original.OriginsUpdateFuture
type OriginUpdateParameters = original.OriginUpdateParameters
type Profile = original.Profile
type ProfileListResult = original.ProfileListResult
type ProfileListResultIterator = original.ProfileListResultIterator
type ProfileListResultPage = original.ProfileListResultPage
type ProfileProperties = original.ProfileProperties
type ProfilesCreateFuture = original.ProfilesCreateFuture
type ProfilesDeleteFuture = original.ProfilesDeleteFuture
type ProfilesUpdateFuture = original.ProfilesUpdateFuture
type ProfileUpdateParameters = original.ProfileUpdateParameters
type ProxyResource = original.ProxyResource
type PurgeParameters = original.PurgeParameters
type Resource = original.Resource
type ResourceUsage = original.ResourceUsage
type ResourceUsageListResult = original.ResourceUsageListResult
type ResourceUsageListResultIterator = original.ResourceUsageListResultIterator
type ResourceUsageListResultPage = original.ResourceUsageListResultPage
type Sku = original.Sku
type SsoURI = original.SsoURI
type SupportedOptimizationTypesListResult = original.SupportedOptimizationTypesListResult
type TrackedResource = original.TrackedResource
type ValidateCustomDomainInput = original.ValidateCustomDomainInput
type ValidateCustomDomainOutput = original.ValidateCustomDomainOutput
type ValidateProbeInput = original.ValidateProbeInput
type ValidateProbeOutput = original.ValidateProbeOutput
type OriginsClient = original.OriginsClient
type ResourceUsageClient = original.ResourceUsageClient
type OperationsClient = original.OperationsClient
type ProfilesClient = original.ProfilesClient

func NewEdgeNodesClient(subscriptionID string) EdgeNodesClient {
	return original.NewEdgeNodesClient(subscriptionID)
}
func NewEdgeNodesClientWithBaseURI(baseURI string, subscriptionID string) EdgeNodesClient {
	return original.NewEdgeNodesClientWithBaseURI(baseURI, subscriptionID)
}
func NewEndpointsClient(subscriptionID string) EndpointsClient {
	return original.NewEndpointsClient(subscriptionID)
}
func NewEndpointsClientWithBaseURI(baseURI string, subscriptionID string) EndpointsClient {
	return original.NewEndpointsClientWithBaseURI(baseURI, subscriptionID)
}
func NewOriginsClient(subscriptionID string) OriginsClient {
	return original.NewOriginsClient(subscriptionID)
}
func NewOriginsClientWithBaseURI(baseURI string, subscriptionID string) OriginsClient {
	return original.NewOriginsClientWithBaseURI(baseURI, subscriptionID)
}
func NewResourceUsageClient(subscriptionID string) ResourceUsageClient {
	return original.NewResourceUsageClient(subscriptionID)
}
func NewResourceUsageClientWithBaseURI(baseURI string, subscriptionID string) ResourceUsageClient {
	return original.NewResourceUsageClientWithBaseURI(baseURI, subscriptionID)
}
func New(subscriptionID string) BaseClient {
	return original.New(subscriptionID)
}
func NewWithBaseURI(baseURI string, subscriptionID string) BaseClient {
	return original.NewWithBaseURI(baseURI, subscriptionID)
}
func NewCustomDomainsClient(subscriptionID string) CustomDomainsClient {
	return original.NewCustomDomainsClient(subscriptionID)
}
func NewCustomDomainsClientWithBaseURI(baseURI string, subscriptionID string) CustomDomainsClient {
	return original.NewCustomDomainsClientWithBaseURI(baseURI, subscriptionID)
}
func UserAgent() string {
	return original.UserAgent() + " profiles/latest"
}
func Version() string {
	return original.Version()
}
func NewOperationsClient(subscriptionID string) OperationsClient {
	return original.NewOperationsClient(subscriptionID)
}
func NewOperationsClientWithBaseURI(baseURI string, subscriptionID string) OperationsClient {
	return original.NewOperationsClientWithBaseURI(baseURI, subscriptionID)
}
func NewProfilesClient(subscriptionID string) ProfilesClient {
	return original.NewProfilesClient(subscriptionID)
}
func NewProfilesClientWithBaseURI(baseURI string, subscriptionID string) ProfilesClient {
	return original.NewProfilesClientWithBaseURI(baseURI, subscriptionID)
}
