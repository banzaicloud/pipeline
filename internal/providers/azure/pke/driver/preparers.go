// Copyright © 2019 Banzai Cloud
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

package driver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
)

// NodePoolsPreparer implements []NodePool preparation
type NodePoolsPreparer struct {
	logger       logrus.FieldLogger
	namespace    string
	dataProvider interface {
		getExistingNodePoolByName(ctx context.Context, nodePoolName string) (pke.NodePool, error)
		getSubnetCIDR(ctx context.Context, nodePool pke.NodePool) (string, error)
		getVirtualNetworkAddressRange(ctx context.Context) (net.IPNet, error)
	}
}

func (p NodePoolsPreparer) getNodePoolPreparer(i int) NodePoolPreparer {
	return NodePoolPreparer{
		logger:       p.logger,
		namespace:    fmt.Sprintf("%s[%d]", p.namespace, i),
		dataProvider: p.dataProvider,
	}
}

// Prepare validates and provides defaults for a set of NodePools
func (p NodePoolsPreparer) Prepare(ctx context.Context, nodePools []NodePool) error {
	names := make(map[string]bool)
	subnets := make(map[string]string)

	for i := range nodePools {
		np := &nodePools[i]

		if names[np.Name] {
			return validationErrorf("multiple node pools named %q", np.Name)
		}

		if err := p.getNodePoolPreparer(i).Prepare(ctx, np); err != nil {
			return emperror.Wrap(err, "failed to prepare node pool")
		}

		if cidr := subnets[np.Subnet.Name]; cidr == "" {
			subnets[np.Subnet.Name] = np.Subnet.CIDR
		} else if np.Subnet.CIDR != "" {
			_, n1, err := net.ParseCIDR(cidr)
			if err != nil {
				return emperror.Wrap(err, "failed to parse CIDR")
			}
			_, n2, err := net.ParseCIDR(np.Subnet.CIDR)
			if err != nil {
				return emperror.Wrap(err, "failed to parse CIDR")
			}
			if sameNet(*n1, *n2) {
				return emperror.With(errors.New("found identically named subnets with different network ranges"), "subnetName", np.Subnet.Name, "cidr1", cidr, "cidr2", np.Subnet.CIDR)
			}
		}
	}

	reservedRanges := make(map[string]*net.IPNet)
	for _, cidr := range subnets {
		if cidr != "" {
			_, n, err := net.ParseCIDR(cidr)
			if err != nil {
				return emperror.Wrap(err, "failed to parse CIDR")
			}
			if r := reservedRanges[n.IP.String()]; r != nil {
				return emperror.With(errors.New("overlapping network ranges assigned to subnets"), "cidr1", r.String(), "cidr2", cidr)
			}
			reservedRanges[cidr] = n
		}
	}

	vnetAddrRange, err := p.dataProvider.getVirtualNetworkAddressRange(ctx)
	if err != nil {
		return emperror.Wrap(err, "failed to get virtual network CIDR")
	}
	if ones, bits := vnetAddrRange.Mask.Size(); ones > 16 || bits != 32 {
		p.logger.WithField("vnetCIDR", vnetAddrRange).Warning("only /16 or larger virtual networks are supported")
	}
	vnetIP := vnetAddrRange.IP.To4()
	for name, cidr := range subnets {
		if cidr == "" {
			var sn net.IPNet
			sn.IP = cloneIP(vnetIP)
			sn.Mask = net.CIDRMask(24, 32)
			for reservedRanges[sn.IP.String()] != nil {
				sn.IP[2]++
				if sn.IP[2] == 0 {
					return emperror.With(errors.New("no free address range for subnet"), "subnet", name)
				}
			}
			subnets[name] = sn.String()
			reservedRanges[sn.IP.String()] = &sn
		}
	}
	for i := range nodePools {
		nodePools[i].Subnet.CIDR = subnets[nodePools[i].Subnet.Name]
	}
	return nil
}

func sameNet(lhs net.IPNet, rhs net.IPNet) bool {
	return bytes.Equal(lhs.IP, rhs.IP) && bytes.Equal(lhs.Mask, rhs.Mask)
}

func cloneIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	clone := make(net.IP, len(ip))
	copy(clone, ip)
	return clone
}

// NodePoolPreparer implements NodePool preparation
type NodePoolPreparer struct {
	logger       logrus.FieldLogger
	namespace    string
	dataProvider interface {
		getExistingNodePoolByName(ctx context.Context, nodePoolName string) (pke.NodePool, error)
		getSubnetCIDR(ctx context.Context, nodePool pke.NodePool) (string, error)
		getVirtualNetworkAddressRange(ctx context.Context) (net.IPNet, error)
	}
}

// Prepare validates and provides defaults for NodePool fields
func (p NodePoolPreparer) Prepare(ctx context.Context, nodePool *NodePool) error {
	if nodePool == nil {
		return nil
	}

	if nodePool.Name == "" {
		return validationErrorf("%s.Name must be specified", p.namespace)
	}

	np, err := p.dataProvider.getExistingNodePoolByName(ctx, nodePool.Name)
	if pke.IsNotFound(err) {
		return p.prepareNewNodePool(ctx, nodePool)
	} else if err != nil {
		return emperror.Wrap(err, "failed to get node pool by name")
	}

	return p.prepareExistingNodePool(ctx, nodePool, np)
}

func (p NodePoolPreparer) getLogger() logrus.FieldLogger {
	return p.logger
}

func (p NodePoolPreparer) getNamespace() string {
	return p.namespace
}

func (p NodePoolPreparer) prepareNewNodePool(ctx context.Context, nodePool *NodePool) error {
	if nodePool.InstanceType == "" {
		return validationErrorf("%s.InstanceType must be specified", p.namespace)
	}

	if len(nodePool.Roles) < 1 {
		return validationErrorf("%s.Roles must not be empty", p.namespace)
	}

	if nodePool.Min > nodePool.Max {
		return validationErrorf("%[1]s.Min must not be greater than %[1]s.Max", p.namespace)
	}

	if nodePool.Subnet.Name == "" {
		nodePool.Subnet.Name = fmt.Sprintf("subnet-%s", nodePool.Name)
		p.logger.Debugf("%s.Subnet.Name not specified, defaulting to [%s]", p.namespace, nodePool.Subnet.Name)
	}

	if cidr := nodePool.Subnet.CIDR; cidr != "" {
		ip, n, err := net.ParseCIDR(cidr)
		if err != nil {
			return validationErrorf("%s.Subnet.CIDR is not valid: %s", p.namespace, err.Error())
		}
		vnetAddrRange, err := p.dataProvider.getVirtualNetworkAddressRange(ctx)
		if err != nil {
			return emperror.Wrap(err, "failed to get virtual network CIDR")
		}
		if !vnetAddrRange.Contains(ip) {
			return emperror.With(validationErrorf("%s.Subnet.CIDR is outside of virtual network address range"), "vnetCIDR", vnetAddrRange.String(), "subnetCIDR", cidr)
		}
		vnetOnes, _ := vnetAddrRange.Mask.Size()
		subnetOnes, _ := n.Mask.Size()
		if vnetOnes > subnetOnes {
			return emperror.With(validationErrorf("%s.Subnet.CIDR is bigger than virtual network address range"), "vnetCIDR", vnetAddrRange.String(), "subnetCIDR", cidr)
		}
	}

	return nil
}

func (p NodePoolPreparer) prepareExistingNodePool(ctx context.Context, nodePool *NodePool, existing pke.NodePool) error {
	if nodePool.CreatedBy != existing.CreatedBy {
		if nodePool.CreatedBy != 0 {
			logMismatchOn(p, "CreatedBy", existing.CreatedBy, nodePool.CreatedBy)
		}
		nodePool.CreatedBy = existing.CreatedBy
	}
	if nodePool.InstanceType != existing.InstanceType {
		if nodePool.InstanceType != "" {
			logMismatchOn(p, "InstanceType", existing.InstanceType, nodePool.InstanceType)
		}
		nodePool.InstanceType = existing.InstanceType
	}
	if stringSliceSetEqual(nodePool.Roles, existing.Roles) {
		if nodePool.Roles != nil {
			logMismatchOn(p, "Roles", existing.Roles, nodePool.Roles)
		}
		nodePool.Roles = existing.Roles
	}
	if nodePool.Subnet.Name != existing.Subnet.Name {
		if nodePool.Subnet.Name != "" {
			logMismatchOn(p, "Subnet.Name", existing.Subnet.Name, nodePool.Subnet.Name)
		}
		nodePool.Subnet.Name = existing.Subnet.Name
	}
	existingSubnetCIDR, err := p.dataProvider.getSubnetCIDR(ctx, existing)
	if err != nil {
		return emperror.Wrap(err, "failed to get subnet CIDR")
	}
	if nodePool.Subnet.CIDR != existingSubnetCIDR {
		if nodePool.Subnet.CIDR != "" {
			logMismatchOn(p, "Subnet.CIDR", existingSubnetCIDR, nodePool.Subnet.CIDR)
		}
		nodePool.Subnet.CIDR = existingSubnetCIDR
	}
	if stringSliceSetEqual(nodePool.Zones, existing.Zones) {
		if nodePool.Zones != nil {
			logMismatchOn(p, "Zones", existing.Zones, nodePool.Zones)
		}
		nodePool.Zones = existing.Zones
	}

	return nil
}

func logMismatchOn(nl interface {
	getLogger() logrus.FieldLogger
	getNamespace() string
}, fieldName string, currentValue, incomingValue interface{}) {
	logMismatch(nl.getLogger(), nl.getNamespace(), fieldName, currentValue, incomingValue)
}

func logMismatch(logger logrus.FieldLogger, namespace, fieldName string, currentValue, incomingValue interface{}) {
	logger.WithField("current", currentValue).WithField("incoming", incomingValue).Warningf("%s.%s does not match existing value", namespace, fieldName)
}

func stringSliceSetEqual(lhs, rhs []string) bool {
	lset := make(map[string]bool, len(lhs))
	for _, e := range lhs {
		lset[e] = true
	}
	if len(lhs) != len(lset) {
		return false // duplicates in lhs
	}

	rset := make(map[string]bool, len(rhs))
	for _, e := range rhs {
		rset[e] = true
	}
	if len(rhs) != len(rset) {
		return false // duplicates in rhs
	}

	if len(lset) != len(rset) {
		return false // different element counts
	}
	for e := range lset {
		if !rset[e] {
			return false // element in lhs missing from rhs
		}
	}
	return true
}

// VirtualNetworkPreparer implements VirtualNetwork preparation
type VirtualNetworkPreparer struct {
	clusterName       string
	connection        *azure.CloudConnection
	logger            logrus.FieldLogger
	namespace         string
	resourceGroupName string
}

const DefaultVirtualNetworkCIDR = "10.0.0.0/16"

// Prepare validates and provides defaults for VirtualNetwork fields
func (p VirtualNetworkPreparer) Prepare(ctx context.Context, vnet *VirtualNetwork) error {
	if vnet.Name == "" {
		vnet.Name = fmt.Sprintf("%s-vnet", p.clusterName)
		p.logger.Debugf("%s.Name not specified, defaulting to [%s]", p.namespace, vnet.Name)
	}
	if vnet.CIDR == "" {
		vnet.CIDR = DefaultVirtualNetworkCIDR
		p.logger.Debugf("%s.CIDR not specified, defaulting to [%s]", p.namespace, vnet.CIDR)
	}
	if vnet.Location == "" {
		rg, err := p.connection.GetGroupsClient().Get(ctx, p.resourceGroupName)
		if err != nil && rg.Response.StatusCode != http.StatusNotFound {
			return emperror.WrapWith(err, "failed to fetch Azure resource group", "resourceGroupName", p.resourceGroupName)
		}
		if rg.Response.StatusCode == http.StatusNotFound || rg.Location == nil || *rg.Location == "" {
			// resource group does not exist (or somehow has no Location), cannot provide default
			return validationErrorf("%s.Location must be specified", p.namespace)
		}
		vnet.Location = *rg.Location
		p.logger.Debugf("%s.Location not specified, defaulting to resource group location [%s]", p.namespace, vnet.Location)
	}
	return nil
}

type validationError struct {
	msg string
}

func validationErrorf(msg string, args ...interface{}) validationError {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return validationError{
		msg: msg,
	}
}

func (e validationError) Error() string {
	return e.msg
}

func (e validationError) InputValidationError() bool {
	return true
}
