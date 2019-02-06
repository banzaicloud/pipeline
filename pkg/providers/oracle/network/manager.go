// Copyright © 2018 Banzai Cloud
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

package network

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
	"github.com/oracle/oci-go-sdk/identity"
)

// VCNManager for creating and deleting preconfigured VCN
type VCNManager struct {
	oci *oci.OCI
	vn  *oci.VirtualNetwork
	vcn core.Vcn
}

// NetworkValues holds network related values used in cluster create/update requests
type NetworkValues struct {
	LBSubnetIDs []string
	WNSubnetIDs []string
}

// NewVCNManager creates a new VCNManager
func NewVCNManager(oci *oci.OCI) *VCNManager {

	return &VCNManager{
		oci: oci,
	}
}

// GetNetworkValues gives back NetworkValues collected from OCI for a given VCN
func (m *VCNManager) GetNetworkValues(vcnID string) (values NetworkValues, err error) {

	vn, err := m.oci.NewVirtualNetworkClient()
	if err != nil {
		return values, err
	}

	vcn, err := vn.GetVCN(&vcnID)
	if err != nil {
		return values, err
	}

	var subnet core.Subnet
	for i := 1; i < 3; i++ {
		if subnet, err = vn.GetSubnetByName(fmt.Sprintf("lb-%d", i), vcn.Id); err != nil {
			return values, err
		}
		values.LBSubnetIDs = append(values.LBSubnetIDs, *subnet.Id)
	}

	for i := 1; i < 4; i++ {
		if subnet, err = vn.GetSubnetByName(fmt.Sprintf("wn-%d", i), vcn.Id); err != nil {
			return values, err
		}
		values.WNSubnetIDs = append(values.WNSubnetIDs, *subnet.Id)
	}

	return values, err
}

// Create creates a preconfigured VCN with the given name
//
// VCN CIDR: 10.0.0.0/16
// - 3 subnets for worker nodes each in different AD within the region
//   10.0.11.0/24, 10.0.12.0/24, 10.0.13.0/24
// - 2 subnets for loadbalancers each in different AD within the region
//   10.0.21.0/24, 10.0.22.0/24
// - 2 security lists
//   workernodes, loadbalancers
func (m *VCNManager) Create(name string) (vcn core.Vcn, err error) {

	vn, err := m.oci.NewVirtualNetworkClient()
	if err != nil {
		return vcn, err
	}
	m.vn = vn

	vcn, err = m.createVCN(name, "10.0.0.0/16")
	if err != nil {
		return vcn, err
	}
	m.vcn = vcn

	igw, err := m.createIGW("gateway-0")
	if err != nil {
		return vcn, err
	}

	err = m.addDefaultRoute(vcn.DefaultRouteTableId, igw)
	if err != nil {
		return vcn, err
	}

	wnSecurityList, err := m.createWorkerNodesSecurityList("workernodes", "10.0.0.0/16")
	if err != nil {
		return vcn, err
	}

	lbSecurityList, err := m.createLoadBalancersSecurityList("loadbalancers")
	if err != nil {
		return vcn, err
	}

	ads, err := m.getAvailabilityDomains()
	if err != nil {
		return vcn, err
	}

	for i := 1; i < 3; i++ {
		if _, err = m.createSubnet(fmt.Sprintf("lb-%d", i), fmt.Sprintf("10.0.2%d.0/24", i), ads[i-1].Name, vcn.DefaultDhcpOptionsId, vcn.DefaultRouteTableId, lbSecurityList.Id); err != nil {
			return vcn, err
		}
	}

	for i := 1; i < 4; i++ {
		if _, err = m.createSubnet(fmt.Sprintf("wn-%d", i), fmt.Sprintf("10.0.1%d.0/24", i), ads[i-1].Name, vcn.DefaultDhcpOptionsId, vcn.DefaultRouteTableId, wnSecurityList.Id); err != nil {
			return vcn, err
		}
	}

	return vcn, nil
}

// Delete deletes a VCN and all related resources by id
func (m *VCNManager) Delete(id *string) error {

	vn, err := m.oci.NewVirtualNetworkClient()
	if err != nil {
		return err
	}
	m.vn = vn

	vcn, err := vn.GetVCN(id)
	if err != nil {
		return err
	}
	m.vcn = vcn

	err = m.removeSubnets()
	if err != nil {
		return err
	}

	err = m.removeAllSecurityList()
	if err != nil {
		return err
	}

	err = m.removeRouteTables()
	if err != nil {
		return err
	}

	err = m.removeInternetGateways()
	if err != nil {
		return err
	}

	return m.vn.DeleteVCN(vcn.Id)
}

func (m *VCNManager) addDefaultRoute(id *string, igw core.InternetGateway) (err error) {

	r := core.UpdateRouteTableRequest{
		RtId: id,
		UpdateRouteTableDetails: core.UpdateRouteTableDetails{
			RouteRules: []core.RouteRule{
				{
					Destination:     common.String("0.0.0.0/0"),
					NetworkEntityId: igw.Id,
				},
			},
		},
	}

	m.oci.GetLogger().Debugf("Adding default route to IGW '%s'", *igw.DisplayName)

	_, err = m.vn.UpdateRouteTable(r)

	return err
}

func (m *VCNManager) createVCN(name string, CIDR string) (vcn core.Vcn, err error) {

	r := core.CreateVcnRequest{
		CreateVcnDetails: core.CreateVcnDetails{
			DisplayName:   common.String(name),
			CidrBlock:     common.String(CIDR),
			CompartmentId: common.String(m.oci.CompartmentOCID),
			DnsLabel:      common.String(CreateDNSLabel(name)),
			FreeformTags:  map[string]string{"created-by": "pipeline"},
		},
	}

	m.oci.GetLogger().Debugf("Creating VCN '%s'", name)

	return m.vn.CreateVCN(r)
}

func (m *VCNManager) createIGW(name string) (igw core.InternetGateway, err error) {

	r := core.CreateInternetGatewayRequest{
		CreateInternetGatewayDetails: core.CreateInternetGatewayDetails{
			CompartmentId: common.String(m.oci.CompartmentOCID),
			VcnId:         m.vcn.Id,
			DisplayName:   common.String(name),
			IsEnabled:     common.Bool(true),
		},
	}

	m.oci.GetLogger().Debugf("Creating IGW '%s'", name)

	return m.vn.CreateInternetGateway(r)
}

func (m *VCNManager) createSecurityList(name string, egressRules []core.EgressSecurityRule, ingressRules []core.IngressSecurityRule) (list core.SecurityList, err error) {

	r := core.CreateSecurityListRequest{
		CreateSecurityListDetails: core.CreateSecurityListDetails{
			CompartmentId:        m.vcn.CompartmentId,
			VcnId:                m.vcn.Id,
			DisplayName:          common.String(name),
			EgressSecurityRules:  egressRules,
			IngressSecurityRules: ingressRules,
		},
	}

	m.oci.GetLogger().Debugf("Creating Security List '%s'", name)

	return m.vn.CreateSecurityList(r)
}

func (m *VCNManager) createWorkerNodesSecurityList(name string, CIDR string) (list core.SecurityList, err error) {

	egress := []core.EgressSecurityRule{
		{
			Destination: common.String(CIDR),
			Protocol:    common.String("all"),
			IsStateless: common.Bool(true),
		},
		{
			Destination: common.String("0.0.0.0/0"),
			Protocol:    common.String("all"),
			IsStateless: common.Bool(false),
		},
	}

	ingress := []core.IngressSecurityRule{
		{
			Source:      common.String(CIDR),
			Protocol:    common.String("all"),
			IsStateless: common.Bool(true),
		},
		{
			Source:      common.String("0.0.0.0/0"),
			Protocol:    common.String("1"),
			IsStateless: common.Bool(false),
			IcmpOptions: &core.IcmpOptions{
				Type: common.Int(3),
				Code: common.Int(4),
			},
		},
		{
			Source:      common.String("0.0.0.0/0"),
			Protocol:    common.String("6"),
			IsStateless: common.Bool(false),
			TcpOptions: &core.TcpOptions{
				DestinationPortRange: &core.PortRange{
					Min: common.Int(22),
					Max: common.Int(22),
				},
			},
		},
		{
			Source:      common.String("130.35.0.0/16"),
			Protocol:    common.String("6"),
			IsStateless: common.Bool(false),
			TcpOptions: &core.TcpOptions{
				DestinationPortRange: &core.PortRange{
					Min: common.Int(22),
					Max: common.Int(22),
				},
			},
		},
		{
			Source:      common.String("138.1.0.0/17"),
			Protocol:    common.String("6"),
			IsStateless: common.Bool(false),
			TcpOptions: &core.TcpOptions{
				DestinationPortRange: &core.PortRange{
					Min: common.Int(22),
					Max: common.Int(22),
				},
			},
		},
		{
			Source:      common.String("134.70.0.0/17"),
			Protocol:    common.String("6"),
			IsStateless: common.Bool(false),
			TcpOptions: &core.TcpOptions{
				DestinationPortRange: &core.PortRange{
					Min: common.Int(22),
					Max: common.Int(22),
				},
			},
		},
		{
			Source:      common.String("140.91.0.0/17"),
			Protocol:    common.String("6"),
			IsStateless: common.Bool(false),
			TcpOptions: &core.TcpOptions{
				DestinationPortRange: &core.PortRange{
					Min: common.Int(22),
					Max: common.Int(22),
				},
			},
		},
		{
			Source:      common.String("147.154.0.0/16"),
			Protocol:    common.String("6"),
			IsStateless: common.Bool(false),
			TcpOptions: &core.TcpOptions{
				DestinationPortRange: &core.PortRange{
					Min: common.Int(22),
					Max: common.Int(22),
				},
			},
		},
		{
			Source:      common.String("192.29.0.0/16"),
			Protocol:    common.String("6"),
			IsStateless: common.Bool(false),
			TcpOptions: &core.TcpOptions{
				DestinationPortRange: &core.PortRange{
					Min: common.Int(22),
					Max: common.Int(22),
				},
			},
		},
	}

	return m.createSecurityList(name, egress, ingress)
}

func (m *VCNManager) createLoadBalancersSecurityList(name string) (list core.SecurityList, err error) {

	egress := []core.EgressSecurityRule{
		{
			Destination: common.String("0.0.0.0/0"),
			Protocol:    common.String("6"),
			IsStateless: common.Bool(true),
		},
	}

	ingress := []core.IngressSecurityRule{
		{
			Source:      common.String("0.0.0.0/0"),
			Protocol:    common.String("6"),
			IsStateless: common.Bool(true),
		},
	}

	return m.createSecurityList(name, egress, ingress)
}

func (m *VCNManager) createSubnet(name string, CIDR string, AD *string, DHCPOptionsID *string, RouteTableID *string, SecurityListID *string) (subnet core.Subnet, err error) {

	r := core.CreateSubnetRequest{
		CreateSubnetDetails: core.CreateSubnetDetails{
			AvailabilityDomain: AD,
			CidrBlock:          common.String(CIDR),
			CompartmentId:      m.vcn.CompartmentId,
			VcnId:              m.vcn.Id,
			DisplayName:        common.String(name),
			DhcpOptionsId:      DHCPOptionsID,
			RouteTableId:       RouteTableID,
			SecurityListIds:    []string{*SecurityListID},
			DnsLabel:           common.String(CreateDNSLabel(name)),
		},
	}

	m.oci.GetLogger().Debugf("Creating Subnet '%s'", name)

	return m.vn.CreateSubnet(r)
}

func (m *VCNManager) removeAllSecurityList() error {

	lists, err := m.vn.GetSecurityLists(m.vcn.Id)
	if err != nil {
		return err
	}

	for _, list := range lists {
		// skip the default security list of the vcn
		if *m.vcn.DefaultSecurityListId == *list.Id {
			continue
		}
		m.oci.GetLogger().Debugf("Removing Security List '%s'", *list.DisplayName)
		err = m.vn.DeleteSecurityList(list.Id)
		if err != nil {
			return err
		}
	}

	return err
}

func (m *VCNManager) removeRouteTables() error {

	tables, err := m.vn.GetRouteTables(m.vcn.Id)
	if err != nil {
		return err
	}

	for _, table := range tables {
		if *m.vcn.DefaultRouteTableId == *table.Id {
			if len(table.RouteRules) > 0 {
				m.removeRoutesFromRouteTable(table.Id)
			}
		} else {
			m.oci.GetLogger().Debugf("Removing Route Table '%s'", *table.DisplayName)
			m.vn.DeleteRouteTable(table.Id)
		}
	}

	return nil
}

func (m *VCNManager) removeInternetGateways() error {

	igws, err := m.vn.GetInternetGateways(m.vcn.Id)
	if err != nil {
		return err
	}

	for _, igw := range igws {
		m.oci.GetLogger().Debugf("Removing Internet Gateway '%s'", *igw.DisplayName)
		m.vn.DeleteInternetGateway(igw.Id)
	}

	return nil
}

func (m *VCNManager) removeSubnets() error {

	subnets, err := m.vn.GetSubnets(m.vcn.Id)
	if err != nil {
		return err
	}

	for _, subnet := range subnets {
		m.oci.GetLogger().Debugf("Removing Subnet '%s'", *subnet.DisplayName)
		err = m.vn.DeleteSubnet(subnet.Id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *VCNManager) removeRoutesFromRouteTable(id *string) (err error) {

	r := core.UpdateRouteTableRequest{
		RtId: id,
		UpdateRouteTableDetails: core.UpdateRouteTableDetails{
			RouteRules: []core.RouteRule{},
		},
	}

	m.oci.GetLogger().Debug("Removing routes from Route Table")

	_, err = m.vn.UpdateRouteTable(r)

	return err
}

func (m *VCNManager) getAvailabilityDomains() (domains []identity.AvailabilityDomain, err error) {

	i, err := m.oci.NewIdentityClient()
	if err != nil {
		return domains, err
	}

	return i.GetAvailabilityDomains()
}

// CreateDNSLabel creates max 15 char long lowercased dns label string from the given str
func CreateDNSLabel(str string) (cstr string) {

	cstr = ""
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return cstr
	}
	cstr = strings.ToLower(reg.ReplaceAllString(str, ""))

	if len(cstr) > 15 {
		cstr = cstr[0:15]
	}

	return cstr
}
