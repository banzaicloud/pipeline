package cluster

import (
	"fmt"
	"regexp"

	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// Cluster describes Pipeline's Oracle fields of a Create/Update request
type Cluster struct {
	Version   string               `json:"version"`
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`

	vcnID       string
	lbSubnetID1 string
	lbSubnetID2 string
}

// NodePool describes Oracle's node fields of a Create/Update request
type NodePool struct {
	Version string            `json:"version,omitempty"`
	Count   uint              `json:"count,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
	Image   string            `json:"image,omitempty"`
	Shape   string            `json:"shape,omitempty"`

	subnetIds         []string
	quantityPerSubnet uint
}

// SetVCNID sets VCNID
func (c *Cluster) SetVCNID(id string) {

	c.vcnID = id
}

// GetVCNID gets VCNID
func (c *Cluster) GetVCNID() (id string) {

	return c.vcnID
}

// SetLBSubnetID1 sets LBSubnetID1
func (c *Cluster) SetLBSubnetID1(id string) {

	c.lbSubnetID1 = id
}

// GetLBSubnetID1 gets LBSubnetID1
func (c *Cluster) GetLBSubnetID1() (id string) {

	return c.lbSubnetID1
}

// SetLBSubnetID2 sets LBSubnetID2
func (c *Cluster) SetLBSubnetID2(id string) {

	c.lbSubnetID2 = id
}

// GetLBSubnetID2 gets LBSubnetID2
func (c *Cluster) GetLBSubnetID2() (id string) {

	return c.lbSubnetID2
}

// SetQuantityPerSubnet sets QuantityPerSubnet
func (np *NodePool) SetQuantityPerSubnet(q uint) {

	np.quantityPerSubnet = q
}

// GetQuantityPerSubnet gets QuantityPerSubnet
func (np *NodePool) GetQuantityPerSubnet() (q uint) {

	return np.quantityPerSubnet
}

// SetSubnetIDs sets SubnetIDs
func (np *NodePool) SetSubnetIDs(ids []string) {

	np.subnetIds = ids
}

// GetSubnetIDs gets SubnetIDs
func (np *NodePool) GetSubnetIDs() (ids []string) {

	return np.subnetIds
}

// AddDefaults adds default values to the request
func (c *Cluster) AddDefaults() error {

	if c == nil {
		return nil
	}

	for name, np := range c.NodePools {
		np.Labels[pkgCommon.LabelKey] = name
	}

	return nil
}

// Validate validates Oracle cluster create request
func (c *Cluster) Validate(update bool) error {

	if c == nil {
		return fmt.Errorf("Oracle is <nil>")
	}

	if !isValidVersion(c.Version) {
		return fmt.Errorf("Invalid k8s version: %s", c.Version)
	}

	if len(c.NodePools) < 1 {
		return fmt.Errorf("At least 1 node pool must be specified")
	}

	for name, nodePool := range c.NodePools {
		if nodePool.Version != c.Version {
			return fmt.Errorf("NodePool[%s]: Different k8s versions were specified for master and nodes", name)
		}
		if nodePool.Image == "" && !update {
			return fmt.Errorf("NodePool[%s]: Node image must be specified", name)
		}
		if nodePool.Shape == "" && !update {
			return fmt.Errorf("NodePool[%s]: Node shape must be specified", name)
		}
	}

	return nil
}

// isValidVersion validates the given K8S version
func isValidVersion(version string) bool {

	isOk, _ := regexp.MatchString("^v\\d+\\.\\d+\\.\\d+", version)
	return isOk
}
