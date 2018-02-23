/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azure

import (
	"context"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/azure-sdk-for-go/profiles/2017-03-09/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/2017-03-09/network/mgmt/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/types"
)

// requestBackoff if backoff is disabled in cloud provider it
// returns a new Backoff object steps = 1
// This is to make sure that the requested command executes
// at least once
func (az *Cloud) requestBackoff() (resourceRequestBackoff wait.Backoff) {
	if az.CloudProviderBackoff {
		return az.resourceRequestBackoff
	}
	resourceRequestBackoff = wait.Backoff{
		Steps: 1,
	}

	return resourceRequestBackoff
}

// GetVirtualMachineWithRetry invokes az.getVirtualMachine with exponential backoff retry
func (az *Cloud) GetVirtualMachineWithRetry(name types.NodeName) (compute.VirtualMachine, error) {
	var machine compute.VirtualMachine
	var retryErr error
	err := wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		machine, retryErr = az.getVirtualMachine(name)
		if retryErr != nil {
			glog.Errorf("backoff: failure, will retry,err=%v", retryErr)
			return false, nil
		}
		glog.V(2).Infof("backoff: success")
		return true, nil
	})
	if err == wait.ErrWaitTimeout {
		err = retryErr
	}

	return machine, err
}

// GetScaleSetsVMWithRetry invokes az.getScaleSetsVM with exponential backoff retry
func (az *Cloud) GetScaleSetsVMWithRetry(name types.NodeName) (compute.VirtualMachineScaleSetVM, bool, error) {
	var machine compute.VirtualMachineScaleSetVM
	var exists bool
	err := wait.ExponentialBackoff(az.resourceRequestBackoff, func() (bool, error) {
		var retryErr error
		machine, exists, retryErr = az.getVmssVirtualMachine(name)
		if retryErr != nil {
			glog.Errorf("GetScaleSetsVMWithRetry backoff: failure, will retry,err=%v", retryErr)
			return false, nil
		}
		glog.V(10).Infof("GetScaleSetsVMWithRetry backoff: success")
		return true, nil
	})
	return machine, exists, err
}

// VirtualMachineClientListWithRetry invokes az.VirtualMachinesClient.List with exponential backoff retry
func (az *Cloud) VirtualMachineClientListWithRetry() ([]compute.VirtualMachine, error) {
	allNodes := []compute.VirtualMachine{}
	var result compute.VirtualMachineListResult
	var resultPage compute.VirtualMachineListResultPage
	err := wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		cntx := context.Background()
		var retryErr error
		az.operationPollRateLimiter.Accept()
		glog.V(10).Infof("VirtualMachinesClient.List(%v): start", az.ResourceGroup)
		resultPage, retryErr = az.VirtualMachinesClient.List(cntx, az.ResourceGroup)
		result = resultPage.Response()
		glog.V(10).Infof("VirtualMachinesClient.List(%v): end", az.ResourceGroup)
		if retryErr != nil {
			glog.Errorf("VirtualMachinesClient.List(%v) - backoff: failure, will retry,err=%v",
				az.ResourceGroup,
				retryErr)
			return false, retryErr
		}
		glog.V(2).Infof("VirtualMachinesClient.List(%v) - backoff: success", az.ResourceGroup)
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	appendResults := (result.Value != nil && len(*result.Value) > 0)
	for appendResults {
		allNodes = append(allNodes, *result.Value...)
		appendResults = false
		// follow the next link to get all the vms for resource group
		if result.NextLink != nil {
			err := wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
				var retryErr error
				az.operationPollRateLimiter.Accept()
				glog.V(10).Infof("VirtualMachinesClient.ListNextResults(%v): start", az.ResourceGroup)
				retryErr = resultPage.Next()
				result = resultPage.Response()
				glog.V(10).Infof("VirtualMachinesClient.ListNextResults(%v): end", az.ResourceGroup)
				if retryErr != nil {
					glog.Errorf("VirtualMachinesClient.ListNextResults(%v) - backoff: failure, will retry,err=%v",
						az.ResourceGroup, retryErr)
					return false, retryErr
				}
				glog.V(2).Infof("VirtualMachinesClient.ListNextResults(%v): success", az.ResourceGroup)
				return true, nil
			})
			if err != nil {
				return allNodes, err
			}
			appendResults = (result.Value != nil && len(*result.Value) > 0)
		}
	}

	return allNodes, err
}

// GetIPForMachineWithRetry invokes az.getIPForMachine with exponential backoff retry
func (az *Cloud) GetIPForMachineWithRetry(name types.NodeName) (string, error) {
	var ip string
	err := wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		var retryErr error
		ip, retryErr = az.getIPForMachine(name)
		if retryErr != nil {
			glog.Errorf("backoff: failure, will retry,err=%v", retryErr)
			return false, nil
		}
		glog.V(2).Infof("backoff: success")
		return true, nil
	})
	return ip, err
}

// CreateOrUpdateSGWithRetry invokes az.SecurityGroupsClient.CreateOrUpdate with exponential backoff retry
func (az *Cloud) CreateOrUpdateSGWithRetry(sg network.SecurityGroup) error {
	return wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		az.operationPollRateLimiter.Accept()
		glog.V(10).Infof("SecurityGroupsClient.CreateOrUpdate(%s): start", *sg.Name)
		cntx := context.Background()
		future, _ := az.SecurityGroupsClient.CreateOrUpdate(cntx, az.ResourceGroup, *sg.Name, sg)
		resp, err := future.Result(az.SecurityGroupsClient)
		glog.V(10).Infof("SecurityGroupsClient.CreateOrUpdate(%s): end", *sg.Name)
		return processRetryResponse(resp.Response, err)
	})
}

// CreateOrUpdateLBWithRetry invokes az.LoadBalancerClient.CreateOrUpdate with exponential backoff retry
func (az *Cloud) CreateOrUpdateLBWithRetry(lb network.LoadBalancer) error {
	return wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		az.operationPollRateLimiter.Accept()
		glog.V(10).Infof("LoadBalancerClient.CreateOrUpdate(%s): start", *lb.Name)
		cntx := context.Background()
		future, _ := az.LoadBalancerClient.CreateOrUpdate(cntx, az.ResourceGroup, *lb.Name, lb)
		resp, err := future.Result(az.LoadBalancerClient)
		glog.V(10).Infof("LoadBalancerClient.CreateOrUpdate(%s): end", *lb.Name)
		return processRetryResponse(resp.Response, err)
	})
}

// ListLBWithRetry invokes az.LoadBalancerClient.List with exponential backoff retry
func (az *Cloud) ListLBWithRetry() ([]network.LoadBalancer, error) {
	allLBs := []network.LoadBalancer{}
	var result network.LoadBalancerListResult
	var resultPage network.LoadBalancerListResultPage

	err := wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		var retryErr error
		az.operationPollRateLimiter.Accept()
		glog.V(10).Infof("LoadBalancerClient.List(%v): start", az.ResourceGroup)
		cntx := context.Background()
		resultPage, retryErr = az.LoadBalancerClient.List(cntx, az.ResourceGroup)
		result = resultPage.Response()
		glog.V(10).Infof("LoadBalancerClient.List(%v): end", az.ResourceGroup)
		if retryErr != nil {
			glog.Errorf("LoadBalancerClient.List(%v) - backoff: failure, will retry,err=%v",
				az.ResourceGroup,
				retryErr)
			return false, retryErr
		}
		glog.V(2).Infof("LoadBalancerClient.List(%v) - backoff: success", az.ResourceGroup)
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	appendResults := (result.Value != nil && len(*result.Value) > 0)
	for appendResults {
		allLBs = append(allLBs, *result.Value...)
		appendResults = false

		// follow the next link to get all the vms for resource group
		if result.NextLink != nil {
			err := wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
				var retryErr error
				az.operationPollRateLimiter.Accept()
				glog.V(10).Infof("LoadBalancerClient.ListNextResults(%v): start", az.ResourceGroup)
				retryErr = resultPage.Next()
				result = resultPage.Response()
				glog.V(10).Infof("LoadBalancerClient.ListNextResults(%v): end", az.ResourceGroup)
				if retryErr != nil {
					glog.Errorf("LoadBalancerClient.ListNextResults(%v) - backoff: failure, will retry,err=%v",
						az.ResourceGroup,
						retryErr)
					return false, retryErr
				}
				glog.V(2).Infof("LoadBalancerClient.ListNextResults(%v) - backoff: success", az.ResourceGroup)
				return true, nil
			})
			if err != nil {
				return allLBs, err
			}
			appendResults = (result.Value != nil && len(*result.Value) > 0)
		}
	}

	return allLBs, nil
}

// ListPIPWithRetry list the PIP resources in az.ResourceGroup
func (az *Cloud) ListPIPWithRetry() ([]network.PublicIPAddress, error) {
	allPIPs := []network.PublicIPAddress{}
	var result network.PublicIPAddressListResult
	var resultPage network.PublicIPAddressListResultPage
	err := wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		var retryErr error
		az.operationPollRateLimiter.Accept()
		glog.V(10).Infof("PublicIPAddressesClient.List(%v): start", az.ResourceGroup)
		cntx := context.Background()
		resultPage, retryErr = az.PublicIPAddressesClient.List(cntx, az.ResourceGroup)
		result = resultPage.Response()
		glog.V(10).Infof("PublicIPAddressesClient.List(%v): end", az.ResourceGroup)
		if retryErr != nil {
			glog.Errorf("PublicIPAddressesClient.List(%v) - backoff: failure, will retry,err=%v",
				az.ResourceGroup,
				retryErr)
			return false, retryErr
		}
		glog.V(2).Infof("PublicIPAddressesClient.List(%v) - backoff: success", az.ResourceGroup)
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	appendResults := (result.Value != nil && len(*result.Value) > 0)
	for appendResults {
		allPIPs = append(allPIPs, *result.Value...)
		appendResults = false

		// follow the next link to get all the vms for resource group
		if result.NextLink != nil {
			err := wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
				var retryErr error
				az.operationPollRateLimiter.Accept()
				glog.V(10).Infof("PublicIPAddressesClient.ListNextResults(%v): start", az.ResourceGroup)
				retryErr = resultPage.Next()
				result = resultPage.Response()
				glog.V(10).Infof("PublicIPAddressesClient.ListNextResults(%v): end", az.ResourceGroup)
				if retryErr != nil {
					glog.Errorf("PublicIPAddressesClient.ListNextResults(%v) - backoff: failure, will retry,err=%v",
						az.ResourceGroup,
						retryErr)
					return false, retryErr
				}
				glog.V(2).Infof("PublicIPAddressesClient.ListNextResults(%v) - backoff: success", az.ResourceGroup)
				return true, nil
			})
			if err != nil {
				return allPIPs, err
			}
			appendResults = (result.Value != nil && len(*result.Value) > 0)
		}
	}

	return allPIPs, nil
}

// CreateOrUpdatePIPWithRetry invokes az.PublicIPAddressesClient.CreateOrUpdate with exponential backoff retry
func (az *Cloud) CreateOrUpdatePIPWithRetry(pip network.PublicIPAddress) error {
	return wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		az.operationPollRateLimiter.Accept()
		glog.V(10).Infof("PublicIPAddressesClient.CreateOrUpdate(%s): start", *pip.Name)
		cntx := context.Background()
		future, _ := az.PublicIPAddressesClient.CreateOrUpdate(cntx, az.ResourceGroup, *pip.Name, pip)
		resp, err := future.Result(az.PublicIPAddressesClient)
		glog.V(10).Infof("PublicIPAddressesClient.CreateOrUpdate(%s): end", *pip.Name)
		return processRetryResponse(resp.Response, err)
	})
}

// CreateOrUpdateInterfaceWithRetry invokes az.PublicIPAddressesClient.CreateOrUpdate with exponential backoff retry
func (az *Cloud) CreateOrUpdateInterfaceWithRetry(nic network.Interface) error {
	return wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		az.operationPollRateLimiter.Accept()
		glog.V(10).Infof("InterfacesClient.CreateOrUpdate(%s): start", *nic.Name)
		cntx := context.Background()
		future, _ := az.InterfacesClient.CreateOrUpdate(cntx, az.ResourceGroup, *nic.Name, nic)
		resp, err := future.Result(az.InterfacesClient)
		glog.V(10).Infof("InterfacesClient.CreateOrUpdate(%s): end", *nic.Name)
		return processRetryResponse(resp.Response, err)
	})
}

// DeletePublicIPWithRetry invokes az.PublicIPAddressesClient.Delete with exponential backoff retry
func (az *Cloud) DeletePublicIPWithRetry(pipName string) error {
	return wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		az.operationPollRateLimiter.Accept()
		glog.V(10).Infof("PublicIPAddressesClient.Delete(%s): start", pipName)
		cntx := context.Background()
		future, _ := az.PublicIPAddressesClient.Delete(cntx, az.ResourceGroup, pipName)
		resp, err := future.Result(az.PublicIPAddressesClient)
		glog.V(10).Infof("PublicIPAddressesClient.Delete(%s): end", pipName)
		return processRetryResponse(resp, err)
	})
}

// DeleteLBWithRetry invokes az.LoadBalancerClient.Delete with exponential backoff retry
func (az *Cloud) DeleteLBWithRetry(lbName string) error {
	return wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		az.operationPollRateLimiter.Accept()
		glog.V(10).Infof("LoadBalancerClient.Delete(%s): start", lbName)
		cntx := context.Background()
		future, _ := az.LoadBalancerClient.Delete(cntx, az.ResourceGroup, lbName)
		resp, err := future.Result(az.LoadBalancerClient)
		glog.V(10).Infof("LoadBalancerClient.Delete(%s): end", lbName)
		return processRetryResponse(resp, err)
	})
}

// CreateOrUpdateRouteTableWithRetry invokes az.RouteTablesClient.CreateOrUpdate with exponential backoff retry
func (az *Cloud) CreateOrUpdateRouteTableWithRetry(routeTable network.RouteTable) error {
	return wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		az.operationPollRateLimiter.Accept()
		glog.V(10).Infof("RouteTablesClient.CreateOrUpdate(%s): start", *routeTable.Name)
		cntx := context.Background()
		future, _ := az.RouteTablesClient.CreateOrUpdate(cntx, az.ResourceGroup, az.RouteTableName, routeTable)
		resp, err := future.Result(az.RouteTablesClient)
		glog.V(10).Infof("RouteTablesClient.CreateOrUpdate(%s): end", *routeTable.Name)
		return processRetryResponse(resp.Response, err)
	})
}

// CreateOrUpdateRouteWithRetry invokes az.RoutesClient.CreateOrUpdate with exponential backoff retry
func (az *Cloud) CreateOrUpdateRouteWithRetry(route network.Route) error {
	return wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		az.operationPollRateLimiter.Accept()
		glog.V(10).Infof("RoutesClient.CreateOrUpdate(%s): start", *route.Name)
		cntx := context.Background()
		future, _ := az.RoutesClient.CreateOrUpdate(cntx, az.ResourceGroup, az.RouteTableName, *route.Name, route)
		resp, err := future.Result(az.RoutesClient)
		glog.V(10).Infof("RoutesClient.CreateOrUpdate(%s): end", *route.Name)
		return processRetryResponse(resp.Response, err)
	})
}

// DeleteRouteWithRetry invokes az.RoutesClient.Delete with exponential backoff retry
func (az *Cloud) DeleteRouteWithRetry(routeName string) error {
	return wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		az.operationPollRateLimiter.Accept()
		glog.V(10).Infof("RoutesClient.Delete(%s): start", az.RouteTableName)
		cntx := context.Background()
		future, _ := az.RoutesClient.Delete(cntx, az.ResourceGroup, az.RouteTableName, routeName)
		resp, err := future.Result(az.RoutesClient)
		glog.V(10).Infof("RoutesClient.Delete(%s): end", az.RouteTableName)
		return processRetryResponse(resp, err)
	})
}

// CreateOrUpdateVMWithRetry invokes az.VirtualMachinesClient.CreateOrUpdate with exponential backoff retry
func (az *Cloud) CreateOrUpdateVMWithRetry(vmName string, newVM compute.VirtualMachine) error {
	return wait.ExponentialBackoff(az.requestBackoff(), func() (bool, error) {
		az.operationPollRateLimiter.Accept()
		glog.V(10).Infof("VirtualMachinesClient.CreateOrUpdate(%s): start", vmName)
		cntx := context.Background()
		future, _ := az.VirtualMachinesClient.CreateOrUpdate(cntx, az.ResourceGroup, vmName, newVM)
		resp, err := future.Result(az.VirtualMachinesClient)
		glog.V(10).Infof("VirtualMachinesClient.CreateOrUpdate(%s): end", vmName)
		return processRetryResponse(resp.Response, err)
	})
}

// A wait.ConditionFunc function to deal with common HTTP backoff response conditions
func processRetryResponse(resp autorest.Response, err error) (bool, error) {
	if isSuccessHTTPResponse(resp) {
		glog.V(2).Infof("backoff: success, HTTP response=%d", resp.StatusCode)
		return true, nil
	}
	if shouldRetryAPIRequest(resp, err) {
		glog.Errorf("backoff: failure, will retry, HTTP response=%d, err=%v", resp.StatusCode, err)
		// suppress the error object so that backoff process continues
		return false, nil
	}
	// Fall-through: stop periodic backoff, return error object from most recent request
	return true, err
}

// shouldRetryAPIRequest determines if the response from an HTTP request suggests periodic retry behavior
func shouldRetryAPIRequest(resp autorest.Response, err error) bool {
	if err != nil {
		return true
	}
	// HTTP 4xx or 5xx suggests we should retry
	if 399 < resp.StatusCode && resp.StatusCode < 600 {
		return true
	}
	return false
}

// isSuccessHTTPResponse determines if the response from an HTTP request suggests success
func isSuccessHTTPResponse(resp autorest.Response) bool {
	// HTTP 2xx suggests a successful response
	if 199 < resp.StatusCode && resp.StatusCode < 300 {
		return true
	}
	return false
}
