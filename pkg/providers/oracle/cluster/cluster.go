package cluster

import (
	"fmt"
	"regexp"
)

// Cluster describes Pipeline's Oracle fields of a Create/Update request
type Cluster struct {
	Version     string               `json:"version,omitempty"`
	VCNID       string               `json:"vcnId,omitempty"`
	LBSubnetID1 string               `json:"LBSubnetID1,omitempty"`
	LBSubnetID2 string               `json:"LBSubnetID2,omitempty"`
	NodePools   map[string]*NodePool `json:"nodePools,omitempty"`
}

// NodePool describes Oracle's node fields of a Create/Update request
type NodePool struct {
	Version           string            `json:"version,omitempty"`
	SubnetIds         []string          `json:"subnetIds,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	QuantityPerSubnet int               `json:"quantityPerSubnet,omitempty"`
	Image             string            `json:"image,omitempty"`
	Shape             string            `json:"shape,omitempty"`
}

// Validate validates Oracle cluster create request
func (c *Cluster) Validate(update bool) error {

	if c == nil {
		return fmt.Errorf("Oracle is <nil>")
	}

	if !update {
		if c.VCNID == "" {
			return fmt.Errorf("VCN OCID must be specified")
		}

		if !isValidOCID(c.VCNID) {
			return fmt.Errorf("VCN OCID wrong format: %s", c.VCNID)
		}

		if c.LBSubnetID1 == "" || c.LBSubnetID2 == "" {
			return fmt.Errorf("2 LB subnet OCID must be specified")
		}

		if !isValidOCID(c.LBSubnetID1) {
			return fmt.Errorf("LB1 OCID %s: wrong format", c.LBSubnetID1)
		}

		if !isValidOCID(c.LBSubnetID2) {
			return fmt.Errorf("LB2 OCID %s: wrong format", c.LBSubnetID2)
		}
	}

	if !isValidVersion(c.Version) {
		return fmt.Errorf("Invalid k8s version: %s", c.Version)
	}

	if len(c.NodePools) < 1 {
		return fmt.Errorf("At least 1 node pool must be specified")
	}

	for name, nodePool := range c.NodePools {
		if len(nodePool.SubnetIds) < 1 {
			return fmt.Errorf("There must be at least 1 subnet specified")
		}
		for _, subnetOCID := range nodePool.SubnetIds {
			if !isValidOCID(subnetOCID) {
				return fmt.Errorf("NodePool[%s] Subnet OCID %s: wrong format", name, subnetOCID)
			}
		}
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

// isValidOCID validates the given OCID
func isValidOCID(ocid string) bool {

	isOK, _ := regexp.MatchString("^([0-9a-zA-Z-_]+[.:])([0-9a-zA-Z-_]*[.:]){3,}([0-9a-zA-Z-_]+)$", ocid)
	return isOK
}

// isValidVersion validates the given K8S version
func isValidVersion(version string) bool {

	isOk, _ := regexp.MatchString("^v\\d+\\.\\d+\\.\\d+", version)
	return isOk
}
