// +build !providerless

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
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog"
	utilnet "k8s.io/utils/net"
)

var (
	// ErrorNotVmssInstance indicates an instance is not belongint to any vmss.
	ErrorNotVmssInstance = errors.New("not a vmss instance")

	scaleSetNameRE         = regexp.MustCompile(`.*/subscriptions/(?:.*)/Microsoft.Compute/virtualMachineScaleSets/(.+)/virtualMachines(?:.*)`)
	resourceGroupRE        = regexp.MustCompile(`.*/subscriptions/(?:.*)/resourceGroups/(.+)/providers/Microsoft.Compute/virtualMachineScaleSets/(?:.*)/virtualMachines(?:.*)`)
	vmssMachineIDTemplate  = "/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachineScaleSets/%s/virtualMachines/%s"
	vmssIPConfigurationRE  = regexp.MustCompile(`.*/subscriptions/(?:.*)/resourceGroups/(.+)/providers/Microsoft.Compute/virtualMachineScaleSets/(.+)/virtualMachines/(.+)/networkInterfaces(?:.*)`)
	vmssPIPConfigurationRE = regexp.MustCompile(`.*/subscriptions/(?:.*)/resourceGroups/(.+)/providers/Microsoft.Compute/virtualMachineScaleSets/(.+)/virtualMachines/(.+)/networkInterfaces/(.+)/ipConfigurations/(.+)/publicIPAddresses/(.+)`)
	vmssVMProviderIDRE     = regexp.MustCompile(`azure:///subscriptions/(?:.*)/resourceGroups/(.+)/providers/Microsoft.Compute/virtualMachineScaleSets/(.+)/virtualMachines/(?:\d+)`)
)

// scaleSet implements VMSet interface for Azure scale set.
type scaleSet struct {
	*Cloud

	// availabilitySet is also required for scaleSet because some instances
	// (e.g. master nodes) may not belong to any scale sets.
	availabilitySet VMSet

	vmssCache                 *timedCache
	vmssVMCache               *timedCache
	availabilitySetNodesCache *timedCache
}

// newScaleSet creates a new scaleSet.
func newScaleSet(az *Cloud) (VMSet, error) {
	var err error
	ss := &scaleSet{
		Cloud:           az,
		availabilitySet: newAvailabilitySet(az),
	}

	ss.availabilitySetNodesCache, err = ss.newAvailabilitySetNodesCache()
	if err != nil {
		return nil, err
	}

	ss.vmssCache, err = ss.newVMSSCache()
	if err != nil {
		return nil, err
	}

	ss.vmssVMCache, err = ss.newVMSSVirtualMachinesCache()
	if err != nil {
		return nil, err
	}

	return ss, nil
}

func (ss *scaleSet) getVMSS(vmssName string, crt cacheReadType) (*compute.VirtualMachineScaleSet, error) {
	getter := func(vmssName string) (*compute.VirtualMachineScaleSet, error) {
		cached, err := ss.vmssCache.Get(vmssKey, crt)
		if err != nil {
			return nil, err
		}

		vmsses := cached.(*sync.Map)
		if vmss, ok := vmsses.Load(vmssName); ok {
			result := vmss.(*vmssEntry)
			return result.vmss, nil
		}

		return nil, nil
	}

	vmss, err := getter(vmssName)
	if err != nil {
		return nil, err
	}
	if vmss != nil {
		return vmss, nil
	}

	klog.V(3).Infof("Couldn't find VMSS with name %s, refreshing the cache", vmssName)
	ss.vmssCache.Delete(vmssKey)
	vmss, err = getter(vmssName)
	if err != nil {
		return nil, err
	}

	if vmss == nil {
		return nil, cloudprovider.InstanceNotFound
	}
	return vmss, nil
}

// getVmssVM gets virtualMachineScaleSetVM by nodeName from cache.
// It returns cloudprovider.InstanceNotFound if node does not belong to any scale sets.
func (ss *scaleSet) getVmssVM(nodeName string, crt cacheReadType) (string, string, *compute.VirtualMachineScaleSetVM, error) {
	getter := func(nodeName string) (string, string, *compute.VirtualMachineScaleSetVM, error) {
		cached, err := ss.vmssVMCache.Get(vmssVirtualMachinesKey, crt)
		if err != nil {
			return "", "", nil, err
		}

		virtualMachines := cached.(*sync.Map)
		if vm, ok := virtualMachines.Load(nodeName); ok {
			result := vm.(*vmssVirtualMachinesEntry)
			return result.vmssName, result.instanceID, result.virtualMachine, nil
		}

		return "", "", nil, nil
	}

	_, err := getScaleSetVMInstanceID(nodeName)
	if err != nil {
		return "", "", nil, err
	}

	vmssName, instanceID, vm, err := getter(nodeName)
	if err != nil {
		return "", "", nil, err
	}
	if vm != nil {
		return vmssName, instanceID, vm, nil
	}

	klog.V(3).Infof("Couldn't find VMSS VM with nodeName %s, refreshing the cache", nodeName)
	ss.vmssVMCache.Delete(vmssVirtualMachinesKey)
	vmssName, instanceID, vm, err = getter(nodeName)
	if err != nil {
		return "", "", nil, err
	}

	if vm == nil {
		return "", "", nil, cloudprovider.InstanceNotFound
	}
	return vmssName, instanceID, vm, nil
}

// GetPowerStatusByNodeName returns the power state of the specified node.
func (ss *scaleSet) GetPowerStatusByNodeName(name string) (powerState string, err error) {
	_, _, vm, err := ss.getVmssVM(name, cacheReadTypeDefault)
	if err != nil {
		return powerState, err
	}

	if vm.InstanceView != nil && vm.InstanceView.Statuses != nil {
		statuses := *vm.InstanceView.Statuses
		for _, status := range statuses {
			state := to.String(status.Code)
			if strings.HasPrefix(state, vmPowerStatePrefix) {
				return strings.TrimPrefix(state, vmPowerStatePrefix), nil
			}
		}
	}

	// vm.InstanceView or vm.InstanceView.Statuses are nil when the VM is under deleting.
	klog.V(3).Infof("InstanceView for node %q is nil, assuming it's stopped", name)
	return vmPowerStateStopped, nil
}

// getCachedVirtualMachineByInstanceID gets scaleSetVMInfo from cache.
// The node must belong to one of scale sets.
func (ss *scaleSet) getVmssVMByInstanceID(resourceGroup, scaleSetName, instanceID string, crt cacheReadType) (*compute.VirtualMachineScaleSetVM, error) {
	getter := func() (vm *compute.VirtualMachineScaleSetVM, found bool, err error) {
		cached, err := ss.vmssVMCache.Get(vmssVirtualMachinesKey, crt)
		if err != nil {
			return nil, false, err
		}

		virtualMachines := cached.(*sync.Map)
		virtualMachines.Range(func(key, value interface{}) bool {
			vmEntry := value.(*vmssVirtualMachinesEntry)
			if strings.EqualFold(vmEntry.resourceGroup, resourceGroup) &&
				strings.EqualFold(vmEntry.vmssName, scaleSetName) &&
				strings.EqualFold(vmEntry.instanceID, instanceID) {
				vm = vmEntry.virtualMachine
				found = true
				return false
			}

			return true
		})

		return vm, found, nil
	}

	vm, found, err := getter()
	if err != nil {
		return nil, err
	}
	if found {
		return vm, nil
	}

	klog.V(3).Infof("Couldn't find VMSS VM with scaleSetName %q and instanceID %q, refreshing the cache", scaleSetName, instanceID)
	ss.vmssVMCache.Delete(vmssVirtualMachinesKey)
	vm, found, err = getter()
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, cloudprovider.InstanceNotFound
	}

	return vm, nil
}

// GetInstanceIDByNodeName gets the cloud provider ID by node name.
// It must return ("", cloudprovider.InstanceNotFound) if the instance does
// not exist or is no longer running.
func (ss *scaleSet) GetInstanceIDByNodeName(name string) (string, error) {
	managedByAS, err := ss.isNodeManagedByAvailabilitySet(name, cacheReadTypeUnsafe)
	if err != nil {
		klog.Errorf("Failed to check isNodeManagedByAvailabilitySet: %v", err)
		return "", err
	}
	if managedByAS {
		// vm is managed by availability set.
		return ss.availabilitySet.GetInstanceIDByNodeName(name)
	}

	_, _, vm, err := ss.getVmssVM(name, cacheReadTypeUnsafe)
	if err != nil {
		return "", err
	}

	resourceID := *vm.ID
	convertedResourceID, err := convertResourceGroupNameToLower(resourceID)
	if err != nil {
		klog.Errorf("convertResourceGroupNameToLower failed with error: %v", err)
		return "", err
	}
	return convertedResourceID, nil
}

// GetNodeNameByProviderID gets the node name by provider ID.
func (ss *scaleSet) GetNodeNameByProviderID(providerID string) (types.NodeName, error) {
	// NodeName is not part of providerID for vmss instances.
	scaleSetName, err := extractScaleSetNameByProviderID(providerID)
	if err != nil {
		klog.V(4).Infof("Can not extract scale set name from providerID (%s), assuming it is mananaged by availability set: %v", providerID, err)
		return ss.availabilitySet.GetNodeNameByProviderID(providerID)
	}

	resourceGroup, err := extractResourceGroupByProviderID(providerID)
	if err != nil {
		return "", fmt.Errorf("error of extracting resource group for node %q", providerID)
	}

	instanceID, err := getLastSegment(providerID)
	if err != nil {
		klog.V(4).Infof("Can not extract instanceID from providerID (%s), assuming it is mananaged by availability set: %v", providerID, err)
		return ss.availabilitySet.GetNodeNameByProviderID(providerID)
	}

	vm, err := ss.getVmssVMByInstanceID(resourceGroup, scaleSetName, instanceID, cacheReadTypeUnsafe)
	if err != nil {
		return "", err
	}

	if vm.OsProfile != nil && vm.OsProfile.ComputerName != nil {
		nodeName := strings.ToLower(*vm.OsProfile.ComputerName)
		return types.NodeName(nodeName), nil
	}

	return "", nil
}

// GetInstanceTypeByNodeName gets the instance type by node name.
func (ss *scaleSet) GetInstanceTypeByNodeName(name string) (string, error) {
	managedByAS, err := ss.isNodeManagedByAvailabilitySet(name, cacheReadTypeUnsafe)
	if err != nil {
		klog.Errorf("Failed to check isNodeManagedByAvailabilitySet: %v", err)
		return "", err
	}
	if managedByAS {
		// vm is managed by availability set.
		return ss.availabilitySet.GetInstanceTypeByNodeName(name)
	}

	_, _, vm, err := ss.getVmssVM(name, cacheReadTypeUnsafe)
	if err != nil {
		return "", err
	}

	if vm.Sku != nil && vm.Sku.Name != nil {
		return *vm.Sku.Name, nil
	}

	return "", nil
}

// GetZoneByNodeName gets availability zone for the specified node. If the node is not running
// with availability zone, then it returns fault domain.
func (ss *scaleSet) GetZoneByNodeName(name string) (cloudprovider.Zone, error) {
	managedByAS, err := ss.isNodeManagedByAvailabilitySet(name, cacheReadTypeUnsafe)
	if err != nil {
		klog.Errorf("Failed to check isNodeManagedByAvailabilitySet: %v", err)
		return cloudprovider.Zone{}, err
	}
	if managedByAS {
		// vm is managed by availability set.
		return ss.availabilitySet.GetZoneByNodeName(name)
	}

	_, _, vm, err := ss.getVmssVM(name, cacheReadTypeUnsafe)
	if err != nil {
		return cloudprovider.Zone{}, err
	}

	var failureDomain string
	if vm.Zones != nil && len(*vm.Zones) > 0 {
		// Get availability zone for the node.
		zones := *vm.Zones
		zoneID, err := strconv.Atoi(zones[0])
		if err != nil {
			return cloudprovider.Zone{}, fmt.Errorf("failed to parse zone %q: %v", zones, err)
		}

		failureDomain = ss.makeZone(to.String(vm.Location), zoneID)
	} else if vm.InstanceView != nil && vm.InstanceView.PlatformFaultDomain != nil {
		// Availability zone is not used for the node, falling back to fault domain.
		failureDomain = strconv.Itoa(int(*vm.InstanceView.PlatformFaultDomain))
	} else {
		err = fmt.Errorf("failed to get zone info")
		klog.Errorf("GetZoneByNodeName: got unexpected error %v", err)
		ss.deleteCacheForNode(name)
		return cloudprovider.Zone{}, err
	}

	return cloudprovider.Zone{
		FailureDomain: failureDomain,
		Region:        to.String(vm.Location),
	}, nil
}

// GetPrimaryVMSetName returns the VM set name depending on the configured vmType.
// It returns config.PrimaryScaleSetName for vmss and config.PrimaryAvailabilitySetName for standard vmType.
func (ss *scaleSet) GetPrimaryVMSetName() string {
	return ss.Config.PrimaryScaleSetName
}

// GetIPByNodeName gets machine private IP and public IP by node name.
func (ss *scaleSet) GetIPByNodeName(nodeName string) (string, string, error) {
	nic, err := ss.GetPrimaryInterface(nodeName)
	if err != nil {
		klog.Errorf("error: ss.GetIPByNodeName(%s), GetPrimaryInterface(%q), err=%v", nodeName, nodeName, err)
		return "", "", err
	}

	ipConfig, err := getPrimaryIPConfig(nic)
	if err != nil {
		klog.Errorf("error: ss.GetIPByNodeName(%s), getPrimaryIPConfig(%v), err=%v", nodeName, nic, err)
		return "", "", err
	}

	internalIP := *ipConfig.PrivateIPAddress
	publicIP := ""
	if ipConfig.PublicIPAddress != nil && ipConfig.PublicIPAddress.ID != nil {
		pipID := *ipConfig.PublicIPAddress.ID
		matches := vmssPIPConfigurationRE.FindStringSubmatch(pipID)
		if len(matches) == 7 {
			resourceGroupName := matches[1]
			virtualMachineScaleSetName := matches[2]
			virtualmachineIndex := matches[3]
			networkInterfaceName := matches[4]
			IPConfigurationName := matches[5]
			publicIPAddressName := matches[6]
			pip, existsPip, err := ss.getVMSSPublicIPAddress(resourceGroupName, virtualMachineScaleSetName, virtualmachineIndex, networkInterfaceName, IPConfigurationName, publicIPAddressName)
			if err != nil {
				klog.Errorf("ss.getVMSSPublicIPAddress() failed with error: %v", err)
				return "", "", err
			}
			if existsPip && pip.IPAddress != nil {
				publicIP = *pip.IPAddress
			}
		} else {
			klog.Warningf("Failed to get VMSS Public IP with ID %s", pipID)
		}
	}

	return internalIP, publicIP, nil
}

func (ss *scaleSet) getVMSSPublicIPAddress(resourceGroupName string, virtualMachineScaleSetName string, virtualmachineIndex string, networkInterfaceName string, IPConfigurationName string, publicIPAddressName string) (pip network.PublicIPAddress, exists bool, err error) {
	var realErr error
	var message string
	ctx, cancel := getContextWithCancel()
	defer cancel()
	pip, err = ss.PublicIPAddressesClient.GetVirtualMachineScaleSetPublicIPAddress(ctx, resourceGroupName, virtualMachineScaleSetName, virtualmachineIndex, networkInterfaceName, IPConfigurationName, publicIPAddressName, "")
	exists, message, realErr = checkResourceExistsFromError(err)
	if realErr != nil {
		return pip, false, realErr
	}

	if !exists {
		klog.V(2).Infof("Public IP %q not found with message: %q", publicIPAddressName, message)
		return pip, false, nil
	}

	return pip, exists, err
}

// returns a list of private ips assigned to node
// TODO (khenidak): This should read all nics, not just the primary
// allowing users to split ipv4/v6 on multiple nics
func (ss *scaleSet) GetPrivateIPsByNodeName(nodeName string) ([]string, error) {
	ips := make([]string, 0)
	nic, err := ss.GetPrimaryInterface(nodeName)
	if err != nil {
		klog.Errorf("error: ss.GetIPByNodeName(%s), GetPrimaryInterface(%q), err=%v", nodeName, nodeName, err)
		return ips, err
	}

	if nic.IPConfigurations == nil {
		return ips, fmt.Errorf("nic.IPConfigurations for nic (nicname=%q) is nil", *nic.Name)
	}

	for _, ipConfig := range *(nic.IPConfigurations) {
		if ipConfig.PrivateIPAddress != nil {
			ips = append(ips, *(ipConfig.PrivateIPAddress))
		}
	}

	return ips, nil
}

// This returns the full identifier of the primary NIC for the given VM.
func (ss *scaleSet) getPrimaryInterfaceID(machine compute.VirtualMachineScaleSetVM) (string, error) {
	if len(*machine.NetworkProfile.NetworkInterfaces) == 1 {
		return *(*machine.NetworkProfile.NetworkInterfaces)[0].ID, nil
	}

	for _, ref := range *machine.NetworkProfile.NetworkInterfaces {
		if *ref.Primary {
			return *ref.ID, nil
		}
	}

	return "", fmt.Errorf("failed to find a primary nic for the vm. vmname=%q", *machine.Name)
}

func (ss *scaleSet) getPrimaryNetworkInterfaceConfiguration(networkConfigurations []compute.VirtualMachineScaleSetNetworkConfiguration, nodeName string) (*compute.VirtualMachineScaleSetNetworkConfiguration, error) {
	if len(networkConfigurations) == 1 {
		return &networkConfigurations[0], nil
	}

	for idx := range networkConfigurations {
		networkConfig := &networkConfigurations[idx]
		if networkConfig.Primary != nil && *networkConfig.Primary == true {
			return networkConfig, nil
		}
	}

	return nil, fmt.Errorf("failed to find a primary network configuration for the scale set VM %q", nodeName)
}

// getVmssMachineID returns the full identifier of a vmss virtual machine.
func (az *Cloud) getVmssMachineID(subscriptionID, resourceGroup, scaleSetName, instanceID string) string {
	return fmt.Sprintf(
		vmssMachineIDTemplate,
		subscriptionID,
		strings.ToLower(resourceGroup),
		scaleSetName,
		instanceID)
}

// machineName is composed of computerNamePrefix and 36-based instanceID.
// And instanceID part if in fixed length of 6 characters.
// Refer https://msftstack.wordpress.com/2017/05/10/figuring-out-azure-vm-scale-set-machine-names/.
func getScaleSetVMInstanceID(machineName string) (string, error) {
	nameLength := len(machineName)
	if nameLength < 6 {
		return "", ErrorNotVmssInstance
	}

	instanceID, err := strconv.ParseUint(machineName[nameLength-6:], 36, 64)
	if err != nil {
		return "", ErrorNotVmssInstance
	}

	return fmt.Sprintf("%d", instanceID), nil
}

// extractScaleSetNameByProviderID extracts the scaleset name by vmss node's ProviderID.
func extractScaleSetNameByProviderID(providerID string) (string, error) {
	matches := scaleSetNameRE.FindStringSubmatch(providerID)
	if len(matches) != 2 {
		return "", ErrorNotVmssInstance
	}

	return matches[1], nil
}

// extractResourceGroupByProviderID extracts the resource group name by vmss node's ProviderID.
func extractResourceGroupByProviderID(providerID string) (string, error) {
	matches := resourceGroupRE.FindStringSubmatch(providerID)
	if len(matches) != 2 {
		return "", ErrorNotVmssInstance
	}

	return matches[1], nil
}

// listScaleSets lists all scale sets.
func (ss *scaleSet) listScaleSets(resourceGroup string) ([]string, error) {
	var err error
	ctx, cancel := getContextWithCancel()
	defer cancel()

	allScaleSets, err := ss.VirtualMachineScaleSetsClient.List(ctx, resourceGroup)
	if err != nil {
		klog.Errorf("VirtualMachineScaleSetsClient.List failed: %v", err)
		return nil, err
	}

	ssNames := make([]string, 0)
	for _, vmss := range allScaleSets {
		name := *vmss.Name
		if vmss.Sku != nil && to.Int64(vmss.Sku.Capacity) == 0 {
			klog.V(3).Infof("Capacity of VMSS %q is 0, skipping", name)
			continue
		}

		ssNames = append(ssNames, name)
	}

	return ssNames, nil
}

// listScaleSetVMs lists VMs belonging to the specified scale set.
func (ss *scaleSet) listScaleSetVMs(scaleSetName, resourceGroup string) ([]compute.VirtualMachineScaleSetVM, error) {
	var err error
	ctx, cancel := getContextWithCancel()
	defer cancel()

	allVMs, err := ss.VirtualMachineScaleSetVMsClient.List(ctx, resourceGroup, scaleSetName, "", "", string(compute.InstanceView))
	if err != nil {
		klog.Errorf("VirtualMachineScaleSetVMsClient.List failed: %v", err)
		return nil, err
	}

	return allVMs, nil
}

// getAgentPoolScaleSets lists the virtual machines for the resource group and then builds
// a list of scale sets that match the nodes available to k8s.
func (ss *scaleSet) getAgentPoolScaleSets(nodes []*v1.Node) (*[]string, error) {
	agentPoolScaleSets := &[]string{}
	for nx := range nodes {
		if isMasterNode(nodes[nx]) {
			continue
		}

		if ss.ShouldNodeExcludedFromLoadBalancer(nodes[nx]) {
			continue
		}

		nodeName := nodes[nx].Name
		ssName, _, _, err := ss.getVmssVM(nodeName, cacheReadTypeDefault)
		if err != nil {
			return nil, err
		}

		if ssName == "" {
			klog.V(3).Infof("Node %q is not belonging to any known scale sets", nodeName)
			continue
		}

		*agentPoolScaleSets = append(*agentPoolScaleSets, ssName)
	}

	return agentPoolScaleSets, nil
}

// GetVMSetNames selects all possible availability sets or scale sets
// (depending vmType configured) for service load balancer. If the service has
// no loadbalancer mode annotation returns the primary VMSet. If service annotation
// for loadbalancer exists then return the eligible VMSet.
func (ss *scaleSet) GetVMSetNames(service *v1.Service, nodes []*v1.Node) (vmSetNames *[]string, err error) {
	hasMode, isAuto, serviceVMSetNames := getServiceLoadBalancerMode(service)
	if !hasMode {
		// no mode specified in service annotation default to PrimaryScaleSetName.
		scaleSetNames := &[]string{ss.Config.PrimaryScaleSetName}
		return scaleSetNames, nil
	}

	scaleSetNames, err := ss.getAgentPoolScaleSets(nodes)
	if err != nil {
		klog.Errorf("ss.GetVMSetNames - getAgentPoolScaleSets failed err=(%v)", err)
		return nil, err
	}
	if len(*scaleSetNames) == 0 {
		klog.Errorf("ss.GetVMSetNames - No scale sets found for nodes in the cluster, node count(%d)", len(nodes))
		return nil, fmt.Errorf("No scale sets found for nodes, node count(%d)", len(nodes))
	}

	// sort the list to have deterministic selection
	sort.Strings(*scaleSetNames)

	if !isAuto {
		if serviceVMSetNames == nil || len(serviceVMSetNames) == 0 {
			return nil, fmt.Errorf("service annotation for LoadBalancerMode is empty, it should have __auto__ or availability sets value")
		}
		// validate scale set exists
		var found bool
		for sasx := range serviceVMSetNames {
			for asx := range *scaleSetNames {
				if strings.EqualFold((*scaleSetNames)[asx], serviceVMSetNames[sasx]) {
					found = true
					serviceVMSetNames[sasx] = (*scaleSetNames)[asx]
					break
				}
			}
			if !found {
				klog.Errorf("ss.GetVMSetNames - scale set (%s) in service annotation not found", serviceVMSetNames[sasx])
				return nil, fmt.Errorf("scale set (%s) - not found", serviceVMSetNames[sasx])
			}
		}
		vmSetNames = &serviceVMSetNames
	}

	return vmSetNames, nil
}

// extractResourceGroupByVMSSNicID extracts the resource group name by vmss nicID.
func extractResourceGroupByVMSSNicID(nicID string) (string, error) {
	matches := vmssIPConfigurationRE.FindStringSubmatch(nicID)
	if len(matches) != 4 {
		return "", fmt.Errorf("error of extracting resourceGroup from nicID %q", nicID)
	}

	return matches[1], nil
}

// GetPrimaryInterface gets machine primary network interface by node name and vmSet.
func (ss *scaleSet) GetPrimaryInterface(nodeName string) (network.Interface, error) {
	managedByAS, err := ss.isNodeManagedByAvailabilitySet(nodeName, cacheReadTypeDefault)
	if err != nil {
		klog.Errorf("Failed to check isNodeManagedByAvailabilitySet: %v", err)
		return network.Interface{}, err
	}
	if managedByAS {
		// vm is managed by availability set.
		return ss.availabilitySet.GetPrimaryInterface(nodeName)
	}

	ssName, instanceID, vm, err := ss.getVmssVM(nodeName, cacheReadTypeDefault)
	if err != nil {
		// VM is availability set, but not cached yet in availabilitySetNodesCache.
		if err == ErrorNotVmssInstance {
			return ss.availabilitySet.GetPrimaryInterface(nodeName)
		}

		klog.Errorf("error: ss.GetPrimaryInterface(%s), ss.getVmssVM(%s), err=%v", nodeName, nodeName, err)
		return network.Interface{}, err
	}

	primaryInterfaceID, err := ss.getPrimaryInterfaceID(*vm)
	if err != nil {
		klog.Errorf("error: ss.GetPrimaryInterface(%s), ss.getPrimaryInterfaceID(), err=%v", nodeName, err)
		return network.Interface{}, err
	}

	nicName, err := getLastSegment(primaryInterfaceID)
	if err != nil {
		klog.Errorf("error: ss.GetPrimaryInterface(%s), getLastSegment(%s), err=%v", nodeName, primaryInterfaceID, err)
		return network.Interface{}, err
	}
	resourceGroup, err := extractResourceGroupByVMSSNicID(primaryInterfaceID)
	if err != nil {
		return network.Interface{}, err
	}

	ctx, cancel := getContextWithCancel()
	defer cancel()
	nic, err := ss.InterfacesClient.GetVirtualMachineScaleSetNetworkInterface(ctx, resourceGroup, ssName, instanceID, nicName, "")
	if err != nil {
		exists, _, realErr := checkResourceExistsFromError(err)
		if realErr != nil {
			klog.Errorf("error: ss.GetPrimaryInterface(%s), ss.GetVirtualMachineScaleSetNetworkInterface.Get(%s, %s, %s), err=%v", nodeName, resourceGroup, ssName, nicName, realErr)
			return network.Interface{}, err
		}

		if !exists {
			return network.Interface{}, cloudprovider.InstanceNotFound
		}
	}

	// Fix interface's location, which is required when updating the interface.
	// TODO: is this a bug of azure SDK?
	if nic.Location == nil || *nic.Location == "" {
		nic.Location = vm.Location
	}

	return nic, nil
}

// getPrimaryNetworkConfiguration gets primary network interface configuration for scale sets.
func (ss *scaleSet) getPrimaryNetworkConfiguration(networkConfigurationList *[]compute.VirtualMachineScaleSetNetworkConfiguration, scaleSetName string) (*compute.VirtualMachineScaleSetNetworkConfiguration, error) {
	networkConfigurations := *networkConfigurationList
	if len(networkConfigurations) == 1 {
		return &networkConfigurations[0], nil
	}

	for idx := range networkConfigurations {
		networkConfig := &networkConfigurations[idx]
		if networkConfig.Primary != nil && *networkConfig.Primary == true {
			return networkConfig, nil
		}
	}

	return nil, fmt.Errorf("failed to find a primary network configuration for the scale set %q", scaleSetName)
}

func (ss *scaleSet) getPrimaryIPConfigForScaleSet(config *compute.VirtualMachineScaleSetNetworkConfiguration, scaleSetName string) (*compute.VirtualMachineScaleSetIPConfiguration, error) {
	ipConfigurations := *config.IPConfigurations
	if len(ipConfigurations) == 1 {
		return &ipConfigurations[0], nil
	}

	for idx := range ipConfigurations {
		ipConfig := &ipConfigurations[idx]
		if ipConfig.Primary != nil && *ipConfig.Primary == true {
			return ipConfig, nil
		}
	}

	return nil, fmt.Errorf("failed to find a primary IP configuration for the scale set %q", scaleSetName)
}

func getPrimaryIPConfigFromVMSSNetworkConfig(config *compute.VirtualMachineScaleSetNetworkConfiguration) (*compute.VirtualMachineScaleSetIPConfiguration, error) {
	ipConfigurations := *config.IPConfigurations
	if len(ipConfigurations) == 1 {
		return &ipConfigurations[0], nil
	}

	for idx := range ipConfigurations {
		ipConfig := &ipConfigurations[idx]
		if ipConfig.Primary != nil && *ipConfig.Primary == true {
			return ipConfig, nil
		}
	}

	return nil, fmt.Errorf("failed to find a primary IP configuration")
}

// EnsureHostInPool ensures the given VM's Primary NIC's Primary IP Configuration is
// participating in the specified LoadBalancer Backend Pool.
func (ss *scaleSet) EnsureHostInPool(service *v1.Service, nodeName types.NodeName, backendPoolID string, vmSetName string, isInternal bool) error {
	klog.V(3).Infof("ensuring node %q of scaleset %q in LB backendpool %q", nodeName, vmSetName, backendPoolID)
	vmName := mapNodeNameToVMName(nodeName)
	ssName, instanceID, vm, err := ss.getVmssVM(vmName, cacheReadTypeDefault)
	if err != nil {
		return err
	}

	// Check scale set name:
	// - For basic SKU load balancer, return nil if the node's scale set is mismatched with vmSetName.
	// - For standard SKU load balancer, backend could belong to multiple VMSS, so we
	//   don't check vmSet for it.
	if vmSetName != "" && !ss.useStandardLoadBalancer() && !strings.EqualFold(vmSetName, ssName) {
		klog.V(3).Infof("EnsureHostInPool skips node %s because it is not in the scaleSet %s", vmName, vmSetName)
		return nil
	}

	// Find primary network interface configuration.
	if vm.NetworkProfileConfiguration.NetworkInterfaceConfigurations == nil {
		klog.V(4).Infof("EnsureHostInPool: cannot obtain the primary network interface configuration, of vm %s, probably because the vm's being deleted", vmName)
		return nil
	}

	networkInterfaceConfigurations := *vm.NetworkProfileConfiguration.NetworkInterfaceConfigurations
	primaryNetworkInterfaceConfiguration, err := ss.getPrimaryNetworkInterfaceConfiguration(networkInterfaceConfigurations, vmName)
	if err != nil {
		return err
	}

	var primaryIPConfiguration *compute.VirtualMachineScaleSetIPConfiguration
	// Find primary network interface configuration.
	if !utilfeature.DefaultFeatureGate.Enabled(IPv6DualStack) {
		// Find primary IP configuration.
		primaryIPConfiguration, err = getPrimaryIPConfigFromVMSSNetworkConfig(primaryNetworkInterfaceConfiguration)
		if err != nil {
			return err
		}
	} else {
		ipv6 := utilnet.IsIPv6String(service.Spec.ClusterIP)
		primaryIPConfiguration, err = ss.getConfigForScaleSetByIPFamily(primaryNetworkInterfaceConfiguration, vmName, ipv6)
		if err != nil {
			return err
		}
	}

	// Update primary IP configuration's LoadBalancerBackendAddressPools.
	foundPool := false
	newBackendPools := []compute.SubResource{}
	if primaryIPConfiguration.LoadBalancerBackendAddressPools != nil {
		newBackendPools = *primaryIPConfiguration.LoadBalancerBackendAddressPools
	}
	for _, existingPool := range newBackendPools {
		if strings.EqualFold(backendPoolID, *existingPool.ID) {
			foundPool = true
			break
		}
	}

	// The backendPoolID has already been found from existing LoadBalancerBackendAddressPools.
	if foundPool {
		return nil
	}

	if ss.useStandardLoadBalancer() && len(newBackendPools) > 0 {
		// Although standard load balancer supports backends from multiple scale
		// sets, the same network interface couldn't be added to more than one load balancer of
		// the same type. Omit those nodes (e.g. masters) so Azure ARM won't complain
		// about this.
		newBackendPoolsIDs := make([]string, 0, len(newBackendPools))
		for _, pool := range newBackendPools {
			if pool.ID != nil {
				newBackendPoolsIDs = append(newBackendPoolsIDs, *pool.ID)
			}
		}
		isSameLB, oldLBName, err := isBackendPoolOnSameLB(backendPoolID, newBackendPoolsIDs)
		if err != nil {
			return err
		}
		if !isSameLB {
			klog.V(4).Infof("Node %q has already been added to LB %q, omit adding it to a new one", nodeName, oldLBName)
			return nil
		}
	}

	// Compose a new vmssVM with added backendPoolID.
	newBackendPools = append(newBackendPools,
		compute.SubResource{
			ID: to.StringPtr(backendPoolID),
		})
	primaryIPConfiguration.LoadBalancerBackendAddressPools = &newBackendPools
	newVM := compute.VirtualMachineScaleSetVM{
		Sku:      vm.Sku,
		Location: vm.Location,
		VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
			HardwareProfile: vm.HardwareProfile,
			NetworkProfileConfiguration: &compute.VirtualMachineScaleSetVMNetworkProfileConfiguration{
				NetworkInterfaceConfigurations: &networkInterfaceConfigurations,
			},
		},
	}

	// Get the node resource group.
	nodeResourceGroup, err := ss.GetNodeResourceGroup(vmName)
	if err != nil {
		return err
	}

	// Invalidate the cache since right after update
	defer ss.deleteCacheForNode(vmName)

	// Update vmssVM with backoff.
	ctx, cancel := getContextWithCancel()
	defer cancel()
	klog.V(2).Infof("EnsureHostInPool begins to update vmssVM(%s) with new backendPoolID %s", vmName, backendPoolID)
	resp, err := ss.VirtualMachineScaleSetVMsClient.Update(ctx, nodeResourceGroup, ssName, instanceID, newVM, "network_update")
	if ss.CloudProviderBackoff && shouldRetryHTTPRequest(resp, err) {
		klog.V(2).Infof("EnsureHostInPool update backing off vmssVM(%s) with new backendPoolID %s, err: %v", vmName, backendPoolID, err)
		retryErr := ss.UpdateVmssVMWithRetry(nodeResourceGroup, ssName, instanceID, newVM, "network_update")
		if retryErr != nil {
			err = retryErr
			klog.Errorf("EnsureHostInPool update abort backoff vmssVM(%s) with new backendPoolID %s, err: %v", vmName, backendPoolID, err)
		}
	}

	return err
}

func (ss *scaleSet) getConfigForScaleSetByIPFamily(config *compute.VirtualMachineScaleSetNetworkConfiguration, nodeName string, IPv6 bool) (*compute.VirtualMachineScaleSetIPConfiguration, error) {
	ipConfigurations := *config.IPConfigurations

	var ipVersion compute.IPVersion
	if IPv6 {
		ipVersion = compute.IPv6
	} else {
		ipVersion = compute.IPv4
	}
	for idx := range ipConfigurations {
		ipConfig := &ipConfigurations[idx]
		if ipConfig.PrivateIPAddressVersion == ipVersion {
			return ipConfig, nil
		}
	}

	return nil, fmt.Errorf("failed to find a  IPconfiguration(IPv6=%v) for the scale set VM %q", IPv6, nodeName)
}

// updateVMSSInstances invokes ss.VirtualMachineScaleSetsClient.UpdateInstances with exponential backoff retry.
func (ss *scaleSet) updateVMSSInstances(service *v1.Service, scaleSetName string, vmInstanceIDs compute.VirtualMachineScaleSetVMInstanceRequiredIDs) error {
	if ss.Config.shouldOmitCloudProviderBackoff() {
		ctx, cancel := getContextWithCancel()
		defer cancel()
		resp, err := ss.VirtualMachineScaleSetsClient.UpdateInstances(ctx, ss.ResourceGroup, scaleSetName, vmInstanceIDs)
		klog.V(10).Infof("VirtualMachineScaleSetsClient.UpdateInstances(%s): end", scaleSetName)
		return ss.processHTTPResponse(service, "CreateOrUpdateVMSSInstance", resp, err)
	}

	return ss.updateVMSSInstancesWithRetry(service, scaleSetName, vmInstanceIDs)
}

// updateVMSSInstancesWithRetry invokes ss.VirtualMachineScaleSetsClient.UpdateInstances with exponential backoff retry.
func (ss *scaleSet) updateVMSSInstancesWithRetry(service *v1.Service, scaleSetName string, vmInstanceIDs compute.VirtualMachineScaleSetVMInstanceRequiredIDs) error {
	return wait.ExponentialBackoff(ss.RequestBackoff(), func() (bool, error) {
		ctx, cancel := getContextWithCancel()
		defer cancel()
		resp, err := ss.VirtualMachineScaleSetsClient.UpdateInstances(ctx, ss.ResourceGroup, scaleSetName, vmInstanceIDs)
		klog.V(10).Infof("VirtualMachineScaleSetsClient.UpdateInstances(%s): end", scaleSetName)
		return ss.processHTTPRetryResponse(service, "CreateOrUpdateVMSSInstance", resp, err)
	})
}

// getNodesScaleSets returns scalesets with instanceIDs and standard node names for given nodes.
func (ss *scaleSet) getNodesScaleSets(nodes []*v1.Node) (map[string]sets.String, []*v1.Node, error) {
	scalesets := make(map[string]sets.String)
	standardNodes := []*v1.Node{}

	for _, curNode := range nodes {
		if ss.useStandardLoadBalancer() && ss.excludeMasterNodesFromStandardLB() && isMasterNode(curNode) {
			klog.V(4).Infof("Excluding master node %q from load balancer backendpool", curNode.Name)
			continue
		}

		if ss.ShouldNodeExcludedFromLoadBalancer(curNode) {
			klog.V(4).Infof("Excluding unmanaged/external-resource-group node %q", curNode.Name)
			continue
		}

		curScaleSetName, err := extractScaleSetNameByProviderID(curNode.Spec.ProviderID)
		if err != nil {
			klog.V(4).Infof("Node %q is not belonging to any scale sets, assuming it is belong to availability sets", curNode.Name)
			standardNodes = append(standardNodes, curNode)
			continue
		}

		if _, ok := scalesets[curScaleSetName]; !ok {
			scalesets[curScaleSetName] = sets.NewString()
		}

		instanceID, err := getLastSegment(curNode.Spec.ProviderID)
		if err != nil {
			klog.Errorf("Failed to get instance ID for node %q: %v", curNode.Spec.ProviderID, err)
			return nil, nil, err
		}

		scalesets[curScaleSetName].Insert(instanceID)
	}

	return scalesets, standardNodes, nil
}

// ensureHostsInVMSetPool ensures the given Node's primary IP configurations are
// participating in the vmSet's LoadBalancer Backend Pool.
func (ss *scaleSet) ensureHostsInVMSetPool(service *v1.Service, backendPoolID string, vmSetName string, instanceIDs []string, isInternal bool) error {
	klog.V(3).Infof("ensuring hosts %q of scaleset %q in LB backendpool %q", instanceIDs, vmSetName, backendPoolID)
	serviceName := getServiceName(service)
	virtualMachineScaleSet, exists, err := ss.getScaleSet(service, vmSetName)
	if err != nil {
		klog.Errorf("ss.getScaleSet(%s) for service %q failed: %v", vmSetName, serviceName, err)
		return err
	}
	if !exists {
		errorMessage := fmt.Errorf("Scale set %q not found", vmSetName)
		klog.Errorf("%v", errorMessage)
		return errorMessage
	}

	// Find primary network interface configuration.
	networkConfigureList := virtualMachineScaleSet.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations
	primaryNetworkConfiguration, err := ss.getPrimaryNetworkConfiguration(networkConfigureList, vmSetName)
	if err != nil {
		return err
	}

	// Find primary IP configuration.
	primaryIPConfiguration, err := ss.getPrimaryIPConfigForScaleSet(primaryNetworkConfiguration, vmSetName)
	if err != nil {
		return err
	}

	// Update primary IP configuration's LoadBalancerBackendAddressPools.
	foundPool := false
	newBackendPools := []compute.SubResource{}
	if primaryIPConfiguration.LoadBalancerBackendAddressPools != nil {
		newBackendPools = *primaryIPConfiguration.LoadBalancerBackendAddressPools
	}
	for _, existingPool := range newBackendPools {
		if strings.EqualFold(backendPoolID, *existingPool.ID) {
			foundPool = true
			break
		}
	}
	if !foundPool {
		if ss.useStandardLoadBalancer() && len(newBackendPools) > 0 {
			// Although standard load balancer supports backends from multiple vmss,
			// the same network interface couldn't be added to more than one load balancer of
			// the same type. Omit those nodes (e.g. masters) so Azure ARM won't complain
			// about this.
			newBackendPoolsIDs := make([]string, 0, len(newBackendPools))
			for _, pool := range newBackendPools {
				if pool.ID != nil {
					newBackendPoolsIDs = append(newBackendPoolsIDs, *pool.ID)
				}
			}
			isSameLB, oldLBName, err := isBackendPoolOnSameLB(backendPoolID, newBackendPoolsIDs)
			if err != nil {
				return err
			}
			if !isSameLB {
				klog.V(4).Infof("VMSS %q has already been added to LB %q, omit adding it to a new one", vmSetName, oldLBName)
				return nil
			}
		}

		newBackendPools = append(newBackendPools,
			compute.SubResource{
				ID: to.StringPtr(backendPoolID),
			})
		primaryIPConfiguration.LoadBalancerBackendAddressPools = &newBackendPools

		err := ss.createOrUpdateVMSS(service, virtualMachineScaleSet)
		if err != nil {
			return err
		}
	}

	// Update instances to latest VMSS model.
	vmInstanceIDs := compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
		InstanceIds: &instanceIDs,
	}
	err = ss.updateVMSSInstances(service, vmSetName, vmInstanceIDs)
	if err != nil {
		return err
	}

	return nil
}

// getPrimaryNetworkInterfaceConfigurationForScaleSet gets primary network interface configuration for scale set.
func (ss *scaleSet) getPrimaryNetworkInterfaceConfigurationForScaleSet(networkConfigurations []compute.VirtualMachineScaleSetNetworkConfiguration, vmssName string) (*compute.VirtualMachineScaleSetNetworkConfiguration, error) {
	if len(networkConfigurations) == 1 {
		return &networkConfigurations[0], nil
	}

	for idx := range networkConfigurations {
		networkConfig := &networkConfigurations[idx]
		if networkConfig.Primary != nil && *networkConfig.Primary == true {
			return networkConfig, nil
		}
	}

	return nil, fmt.Errorf("failed to find a primary network configuration for the scale set %q", vmssName)
}

// EnsureHostsInPool ensures the given Node's primary IP configurations are
// participating in the specified LoadBalancer Backend Pool.
func (ss *scaleSet) EnsureHostsInPool(service *v1.Service, nodes []*v1.Node, backendPoolID string, vmSetName string, isInternal bool) error {
	serviceName := getServiceName(service)
	scalesets, standardNodes, err := ss.getNodesScaleSets(nodes)
	if err != nil {
		klog.Errorf("getNodesScaleSets() for service %q failed: %v", serviceName, err)
		return err
	}

	for ssName, instanceIDs := range scalesets {
		// Only add nodes belonging to specified vmSet for basic SKU LB.
		if !ss.useStandardLoadBalancer() && !strings.EqualFold(ssName, vmSetName) {
			continue
		}

		if instanceIDs.Len() == 0 {
			// This may happen when scaling a vmss capacity to 0.
			klog.V(3).Infof("scale set %q has 0 nodes, adding it to load balancer anyway", ssName)
			// InstanceIDs is required to update vmss, use * instead here since there are no nodes actually.
			instanceIDs.Insert("*")
		}

		err := ss.ensureHostsInVMSetPool(service, backendPoolID, ssName, instanceIDs.List(), isInternal)
		if err != nil {
			klog.Errorf("ensureHostsInVMSetPool() with scaleSet %q for service %q failed: %v", ssName, serviceName, err)
			return err
		}
	}

	if ss.useStandardLoadBalancer() && len(standardNodes) > 0 {
		err := ss.availabilitySet.EnsureHostsInPool(service, standardNodes, backendPoolID, "", isInternal)
		if err != nil {
			klog.Errorf("availabilitySet.EnsureHostsInPool() for service %q failed: %v", serviceName, err)
			return err
		}
	}

	return nil
}

// ensureScaleSetBackendPoolDeleted ensures the loadBalancer backendAddressPools deleted from the specified scaleset.
func (ss *scaleSet) ensureScaleSetBackendPoolDeleted(service *v1.Service, poolID, ssName string) error {
	klog.V(3).Infof("ensuring backend pool %q deleted from scaleset %q", poolID, ssName)
	virtualMachineScaleSet, exists, err := ss.getScaleSet(service, ssName)
	if err != nil {
		klog.Errorf("ss.ensureScaleSetBackendPoolDeleted(%s, %s) getScaleSet(%s) failed: %v", poolID, ssName, ssName, err)
		return err
	}
	if !exists {
		klog.V(2).Infof("ss.ensureScaleSetBackendPoolDeleted(%s, %s), scale set %s has already been non-exist", poolID, ssName, ssName)
		return nil
	}

	// Find primary network interface configuration.
	networkConfigureList := virtualMachineScaleSet.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations
	primaryNetworkConfiguration, err := ss.getPrimaryNetworkConfiguration(networkConfigureList, ssName)
	if err != nil {
		return err
	}

	// Find primary IP configuration.
	primaryIPConfiguration, err := ss.getPrimaryIPConfigForScaleSet(primaryNetworkConfiguration, ssName)
	if err != nil {
		return err
	}

	// Construct new loadBalancerBackendAddressPools and remove backendAddressPools from primary IP configuration.
	if primaryIPConfiguration.LoadBalancerBackendAddressPools == nil || len(*primaryIPConfiguration.LoadBalancerBackendAddressPools) == 0 {
		return nil
	}
	existingBackendPools := *primaryIPConfiguration.LoadBalancerBackendAddressPools
	newBackendPools := []compute.SubResource{}
	foundPool := false
	for i := len(existingBackendPools) - 1; i >= 0; i-- {
		curPool := existingBackendPools[i]
		if strings.EqualFold(poolID, *curPool.ID) {
			klog.V(10).Infof("ensureScaleSetBackendPoolDeleted gets unwanted backend pool %q for scale set %q", poolID, ssName)
			foundPool = true
			newBackendPools = append(existingBackendPools[:i], existingBackendPools[i+1:]...)
		}
	}
	if !foundPool {
		// Pool not found, assume it has been already removed.
		return nil
	}

	// Update scale set with backoff.
	primaryIPConfiguration.LoadBalancerBackendAddressPools = &newBackendPools
	klog.V(3).Infof("VirtualMachineScaleSetsClient.CreateOrUpdate: scale set (%s) - updating", ssName)
	err = ss.createOrUpdateVMSS(service, virtualMachineScaleSet)
	if err != nil {
		return err
	}

	// Update instances to latest VMSS model.
	instanceIDs := []string{"*"}
	vmInstanceIDs := compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
		InstanceIds: &instanceIDs,
	}
	err = ss.updateVMSSInstances(service, ssName, vmInstanceIDs)
	if err != nil {
		return err
	}

	// Update virtualMachineScaleSet again. This is a workaround for removing VMSS reference from LB.
	// TODO: remove this workaround when figuring out the root cause.
	if len(newBackendPools) == 0 {
		err = ss.createOrUpdateVMSS(service, virtualMachineScaleSet)
		if err != nil {
			klog.V(2).Infof("VirtualMachineScaleSetsClient.CreateOrUpdate abort backoff: scale set (%s) - updating", ssName)
		}
	}

	return nil
}

// ensureBackendPoolDeletedFromNode ensures the loadBalancer backendAddressPools deleted from the specified node.
func (ss *scaleSet) ensureBackendPoolDeletedFromNode(service *v1.Service, nodeName, backendPoolID string) error {
	ssName, instanceID, vm, err := ss.getVmssVM(nodeName, cacheReadTypeDefault)
	if err != nil {
		return err
	}

	// Find primary network interface configuration.
	if vm.NetworkProfileConfiguration.NetworkInterfaceConfigurations == nil {
		klog.V(4).Infof("EnsureHostInPool: cannot obtain the primary network interface configuration, of vm %s, probably because the vm's being deleted", nodeName)
		return nil
	}
	networkInterfaceConfigurations := *vm.NetworkProfileConfiguration.NetworkInterfaceConfigurations
	primaryNetworkInterfaceConfiguration, err := ss.getPrimaryNetworkInterfaceConfiguration(networkInterfaceConfigurations, nodeName)
	if err != nil {
		return err
	}

	// Find primary IP configuration.
	primaryIPConfiguration, err := getPrimaryIPConfigFromVMSSNetworkConfig(primaryNetworkInterfaceConfiguration)
	if err != nil {
		return err
	}
	if primaryIPConfiguration.LoadBalancerBackendAddressPools == nil || len(*primaryIPConfiguration.LoadBalancerBackendAddressPools) == 0 {
		return nil
	}

	// Construct new loadBalancerBackendAddressPools and remove backendAddressPools from primary IP configuration.
	existingBackendPools := *primaryIPConfiguration.LoadBalancerBackendAddressPools
	newBackendPools := []compute.SubResource{}
	foundPool := false
	for i := len(existingBackendPools) - 1; i >= 0; i-- {
		curPool := existingBackendPools[i]
		if strings.EqualFold(backendPoolID, *curPool.ID) {
			klog.V(10).Infof("ensureBackendPoolDeletedFromNode gets unwanted backend pool %q for node %s", backendPoolID, nodeName)
			foundPool = true
			newBackendPools = append(existingBackendPools[:i], existingBackendPools[i+1:]...)
		}
	}

	// Pool not found, assume it has been already removed.
	if !foundPool {
		return nil
	}

	// Compose a new vmssVM with added backendPoolID.
	primaryIPConfiguration.LoadBalancerBackendAddressPools = &newBackendPools
	newVM := compute.VirtualMachineScaleSetVM{
		Sku:      vm.Sku,
		Location: vm.Location,
		VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
			HardwareProfile: vm.HardwareProfile,
			NetworkProfileConfiguration: &compute.VirtualMachineScaleSetVMNetworkProfileConfiguration{
				NetworkInterfaceConfigurations: &networkInterfaceConfigurations,
			},
		},
	}

	// Get the node resource group.
	nodeResourceGroup, err := ss.GetNodeResourceGroup(nodeName)
	if err != nil {
		return err
	}

	// Invalidate the cache since right after update
	defer ss.deleteCacheForNode(nodeName)

	// Update vmssVM with backoff.
	ctx, cancel := getContextWithCancel()
	defer cancel()
	klog.V(2).Infof("ensureBackendPoolDeletedFromNode begins to update vmssVM(%s) with backendPoolID %s", nodeName, backendPoolID)
	resp, err := ss.VirtualMachineScaleSetVMsClient.Update(ctx, nodeResourceGroup, ssName, instanceID, newVM, "network_update")
	if ss.CloudProviderBackoff && shouldRetryHTTPRequest(resp, err) {
		klog.V(2).Infof("ensureBackendPoolDeletedFromNode update backing off vmssVM(%s) with backendPoolID %s, err: %v", nodeName, backendPoolID, err)
		retryErr := ss.UpdateVmssVMWithRetry(nodeResourceGroup, ssName, instanceID, newVM, "network_update")
		if retryErr != nil {
			err = retryErr
			klog.Errorf("ensureBackendPoolDeletedFromNode update abort backoff vmssVM(%s) with backendPoolID %s, err: %v", nodeName, backendPoolID, err)
		}
	}
	if err != nil {
		klog.Errorf("ensureBackendPoolDeletedFromNode failed to update vmssVM(%s) with backendPoolID %s: %v", nodeName, backendPoolID, err)
	} else {
		klog.V(2).Infof("ensureBackendPoolDeletedFromNode update vmssVM(%s) with backendPoolID %s succeeded", nodeName, backendPoolID)
	}
	return err
}

func (ss *scaleSet) ensureBackendPoolDeletedFromVMSS(service *v1.Service, backendPoolID, vmSetName string, ipConfigurationIDs []string) error {
	vmssNamesMap := make(map[string]bool)

	// the standard load balancer supports multiple vmss in its backend while the basic sku doesn't
	if ss.useStandardLoadBalancer() {
		for _, ipConfigurationID := range ipConfigurationIDs {
			// in this scenario the vmSetName is an empty string and the name of vmss should be obtained from the provider IDs of nodes
			vmssName, resourceGroupName, err := getScaleSetAndResourceGroupNameByIPConfigurationID(ipConfigurationID)
			if err != nil {
				klog.V(4).Infof("ensureBackendPoolDeletedFromVMSS: found VMAS ipcConfigurationID %s, will skip checking and continue", ipConfigurationID)
				continue
			}
			// only vmsses in the resource group same as it's in azure config are included
			if strings.EqualFold(resourceGroupName, ss.ResourceGroup) {
				vmssNamesMap[vmssName] = true
			}
		}
	} else {
		vmssNamesMap[vmSetName] = true
	}

	for vmssName := range vmssNamesMap {
		vmss, err := ss.getVMSS(vmssName, cacheReadTypeDefault)

		// When vmss is being deleted, CreateOrUpdate API would report "the vmss is being deleted" error.
		// Since it is being deleted, we shouldn't send more CreateOrUpdate requests for it.
		if vmss.ProvisioningState != nil && strings.EqualFold(*vmss.ProvisioningState, virtualMachineScaleSetsDeallocating) {
			klog.V(3).Infof("ensureVMSSInPool: found vmss %s being deleted, skipping", vmssName)
			continue
		}

		if err != nil {
			return err
		}
		if vmss.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations == nil {
			klog.V(4).Infof("EnsureHostInPool: cannot obtain the primary network interface configuration, of vmss %s", vmssName)
			continue
		}
		vmssNIC := *vmss.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations
		primaryNIC, err := ss.getPrimaryNetworkInterfaceConfigurationForScaleSet(vmssNIC, vmssName)
		if err != nil {
			return err
		}
		primaryIPConfig, err := getPrimaryIPConfigFromVMSSNetworkConfig(primaryNIC)
		if err != nil {
			return err
		}
		loadBalancerBackendAddressPools := []compute.SubResource{}
		if primaryIPConfig.LoadBalancerBackendAddressPools != nil {
			loadBalancerBackendAddressPools = *primaryIPConfig.LoadBalancerBackendAddressPools
		}

		var found bool
		var newBackendPools []compute.SubResource
		for i := len(loadBalancerBackendAddressPools) - 1; i >= 0; i-- {
			curPool := loadBalancerBackendAddressPools[i]
			if strings.EqualFold(backendPoolID, *curPool.ID) {
				klog.V(10).Infof("ensureBackendPoolDeletedFromVMSS gets unwanted backend pool %q for VMSS %s", backendPoolID, vmssName)
				found = true
				newBackendPools = append(loadBalancerBackendAddressPools[:i], loadBalancerBackendAddressPools[i+1:]...)
			}
		}
		if !found {
			continue
		}

		// Compose a new vmss with added backendPoolID.
		primaryIPConfig.LoadBalancerBackendAddressPools = &newBackendPools
		newVMSS := compute.VirtualMachineScaleSet{
			Sku:      vmss.Sku,
			Location: vmss.Location,
			VirtualMachineScaleSetProperties: &compute.VirtualMachineScaleSetProperties{
				VirtualMachineProfile: &compute.VirtualMachineScaleSetVMProfile{
					NetworkProfile: &compute.VirtualMachineScaleSetNetworkProfile{
						NetworkInterfaceConfigurations: &vmssNIC,
					},
				},
			},
		}

		// Update vmssVM with backoff.
		ctx, cancel := getContextWithCancel()
		defer cancel()

		klog.V(2).Infof("ensureBackendPoolDeletedFromVMSS begins to update vmss(%s) with backendPoolID %s", vmssName, backendPoolID)
		resp, err := ss.VirtualMachineScaleSetsClient.CreateOrUpdate(ctx, ss.ResourceGroup, vmssName, newVMSS)
		if ss.CloudProviderBackoff && shouldRetryHTTPRequest(resp, err) {
			klog.V(2).Infof("ensureBackendPoolDeletedFromVMSS update backing off vmss(%s) with backendPoolID %s, err: %v", vmssName, backendPoolID, err)
			retryErr := ss.CreateOrUpdateVmssWithRetry(ss.ResourceGroup, vmssName, newVMSS)
			if retryErr != nil {
				err = retryErr
				klog.Errorf("ensureBackendPoolDeletedFromVMSS update abort backoff vmssVM(%s) with backendPoolID %s, err: %v", vmssName, backendPoolID, err)
			}
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// EnsureBackendPoolDeleted ensures the loadBalancer backendAddressPools deleted from the specified vmSet.
func (ss *scaleSet) EnsureBackendPoolDeleted(service *v1.Service, poolID, vmSetName string, backendAddressPools *[]network.BackendAddressPool) error {
	if backendAddressPools == nil {
		return nil
	}

	scalesets := sets.NewString()
	for _, backendPool := range *backendAddressPools {
		if strings.EqualFold(*backendPool.ID, poolID) && backendPool.BackendIPConfigurations != nil {
			for _, ipConfigurations := range *backendPool.BackendIPConfigurations {
				if ipConfigurations.ID == nil {
					continue
				}

				ssName, err := extractScaleSetNameByProviderID(*ipConfigurations.ID)
				if err != nil {
					klog.V(4).Infof("backend IP configuration %q is not belonging to any vmss, omit it", *ipConfigurations.ID)
					continue
				}

				scalesets.Insert(ssName)
			}
			break
		}
	}

	for ssName := range scalesets {
		// Only remove nodes belonging to specified vmSet to basic LB backends.
		if !ss.useStandardLoadBalancer() && !strings.EqualFold(ssName, vmSetName) {
			continue
		}

		err := ss.ensureScaleSetBackendPoolDeleted(service, poolID, ssName)
		if err != nil {
			klog.Errorf("ensureScaleSetBackendPoolDeleted() with scaleSet %q failed: %v", ssName, err)
			return err
		}
	}

	return nil
}

// getScaleSet gets scale set with exponential backoff retry
func (ss *scaleSet) getScaleSet(service *v1.Service, name string) (compute.VirtualMachineScaleSet, bool, error) {
	if ss.Config.shouldOmitCloudProviderBackoff() {
		var result compute.VirtualMachineScaleSet
		var exists bool

		cached, err := ss.vmssVMCache.Get(name, cacheReadTypeDefault)
		if err != nil {
			ss.Event(service, v1.EventTypeWarning, "GetVirtualMachineScaleSet", err.Error())
			klog.Errorf("backoff: failure for scale set %q, will retry,err=%v", name, err)
			return result, false, nil
		}

		if cached != nil {
			exists = true
			result = *(cached.(*compute.VirtualMachineScaleSet))
		}

		return result, exists, err
	}

	return ss.getScaleSetWithRetry(service, name)
}

// getScaleSetWithRetry gets scale set with exponential backoff retry
func (ss *scaleSet) getScaleSetWithRetry(service *v1.Service, name string) (compute.VirtualMachineScaleSet, bool, error) {
	var result compute.VirtualMachineScaleSet
	var exists bool

	err := wait.ExponentialBackoff(ss.RequestBackoff(), func() (bool, error) {
		cached, retryErr := ss.vmssVMCache.Get(name, cacheReadTypeDefault)
		if retryErr != nil {
			ss.Event(service, v1.EventTypeWarning, "GetVirtualMachineScaleSet", retryErr.Error())
			klog.Errorf("backoff: failure for scale set %q, will retry,err=%v", name, retryErr)
			return false, nil
		}
		klog.V(4).Infof("backoff: success for scale set %q", name)

		if cached != nil {
			exists = true
			result = *(cached.(*compute.VirtualMachineScaleSet))
		}

		return true, nil
	})

	return result, exists, err
}

// createOrUpdateVMSS invokes ss.VirtualMachineScaleSetsClient.CreateOrUpdate with exponential backoff retry.
func (ss *scaleSet) createOrUpdateVMSS(service *v1.Service, virtualMachineScaleSet compute.VirtualMachineScaleSet) error {
	if ss.Config.shouldOmitCloudProviderBackoff() {
		ctx, cancel := getContextWithCancel()
		defer cancel()
		resp, err := ss.VirtualMachineScaleSetsClient.CreateOrUpdate(ctx, ss.ResourceGroup, *virtualMachineScaleSet.Name, virtualMachineScaleSet)
		klog.V(10).Infof("VirtualMachineScaleSetsClient.CreateOrUpdate(%s): end", *virtualMachineScaleSet.Name)
		return ss.processHTTPResponse(service, "CreateOrUpdateVMSS", resp, err)
	}

	return ss.createOrUpdateVMSSWithRetry(service, virtualMachineScaleSet)
}

// createOrUpdateVMSSWithRetry invokes ss.VirtualMachineScaleSetsClient.CreateOrUpdate with exponential backoff retry.
func (ss *scaleSet) createOrUpdateVMSSWithRetry(service *v1.Service, virtualMachineScaleSet compute.VirtualMachineScaleSet) error {
	return wait.ExponentialBackoff(ss.RequestBackoff(), func() (bool, error) {
		ctx, cancel := getContextWithCancel()
		defer cancel()
		resp, err := ss.VirtualMachineScaleSetsClient.CreateOrUpdate(ctx, ss.ResourceGroup, *virtualMachineScaleSet.Name, virtualMachineScaleSet)
		klog.V(10).Infof("VirtualMachineScaleSetsClient.CreateOrUpdate(%s): end", *virtualMachineScaleSet.Name)
		return ss.processHTTPRetryResponse(service, "CreateOrUpdateVMSS", resp, err)
	})
}

// getNodeNameByIPConfigurationID gets the node name by IP configuration ID.
func (ss *scaleSet) getNodeNameByIPConfigurationID(ipConfigurationID string) (string, error) {
	matches := vmssIPConfigurationRE.FindStringSubmatch(ipConfigurationID)
	if len(matches) != 4 {
		klog.V(4).Infof("Can not extract scale set name from ipConfigurationID (%s), assuming it is mananaged by availability set", ipConfigurationID)
		return "", ErrorNotVmssInstance
	}

	resourceGroup := matches[1]
	scaleSetName := matches[2]
	instanceID := matches[3]
	vm, err := ss.getVmssVMByInstanceID(resourceGroup, scaleSetName, instanceID, cacheReadTypeUnsafe)
	if err != nil {
		return "", err
	}

	if vm.OsProfile != nil && vm.OsProfile.ComputerName != nil {
		return strings.ToLower(*vm.OsProfile.ComputerName), nil
	}

	return "", nil
}

func getScaleSetAndResourceGroupNameByIPConfigurationID(ipConfigurationID string) (string, string, error) {
	matches := vmssIPConfigurationRE.FindStringSubmatch(ipConfigurationID)
	if len(matches) != 4 {
		klog.V(4).Infof("Can not extract scale set name from ipConfigurationID (%s), assuming it is mananaged by availability set", ipConfigurationID)
		return "", "", ErrorNotVmssInstance
	}

	resourceGroup := matches[1]
	scaleSetName := matches[2]
	return scaleSetName, resourceGroup, nil
}
