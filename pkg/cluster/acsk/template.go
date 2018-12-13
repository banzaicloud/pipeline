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

package acsk

import "time"

type AlibabaClusterCreateParams struct {
	ClusterType              string `json:"cluster_type"`                  // Network type. Always set to: kubernetes.
	DisableRollback          bool   `json:"disable_rollback,omitempty"`    // Whether the failure is rolled back, true means that the failure does not roll back, and false fails to roll back. If you choose to fail back, it will release the resources produced during the creation process. It is not recommended to use false.
	Name                     string `json:"name"`                          // Cluster name, cluster name can use uppercase and lowercase English letters, Chinese, numbers, and dash.
	TimeoutMins              int    `json:"timeout_mins,omitempty"`        // Cluster resource stack creation timeout in minutes, default value 60.
	RegionID                 string `json:"region_id"`                     // Domain ID of the cluster.
	ZoneID                   string `json:"zoneid"`                        // Regional Availability Zone.
	VPCID                    string `json:"vpcid,omitempty"`               // VPCID, can be empty. If it is not set, the system will automatically create a VPC. The network segment created by the system is 192.168.0.0/16. VpcId and vswitchid can only be empty at the same time or set the corresponding value at the same time.
	VSwitchID                string `json:"vswitchid,omitempty"`           // Switch ID, can be empty. If it is not set, the system automatically creates the switch. The network segment of the switch created by the system is 192.168.0.0/16..
	ContainerCIDR            string `json:"container_cidr,omitempty"`      // The container network segment cannot conflict with the VPC network segment. When the system is selected to automatically create a VPC, the network segment 172.16.0.0/16 is used by default.
	ServiceCIDR              string `json:"service_cidr,omitempty"`        // The service network segment cannot conflict with the VPC segment and container segment. When the system is selected to create a VPC automatically, the network segment 172.19.0.0/20 is used by default.
	MasterInstanceType       string `json:"master_instance_type"`          // Master node ECS specification type code.
	MasterSystemDiskCategory string `json:"master_system_disk_category"`   // Master node system disk type.
	MasterSystemDiskSize     int    `json:"master_system_disk_size"`       // Master node system disk size.
	WorkerInstanceType       string `json:"worker_instance_type"`          // Worker node ECS specification type code.
	WorkerSystemDiskCategory string `json:"worker_system_disk_category"`   // Worker node system disk type.
	KeyPair                  string `json:"key_pair"`                      // Keypair name. Choose one with login_password
	NumOfNodes               int    `json:"num_of_nodes"`                  // Worker node number. The range is [0,300].
	SNATEntry                bool   `json:"snat_entry"`                    // Whether to configure SNAT for the network. If it is automatically created VPC must be set to true. If you are using an existing VPC, set it according to whether you have network access capability
	SSHFlags                 bool   `json:"ssh_flags,omitempty"`           // Whether to open public network SSH login.
	CloudMonitorFlags        bool   `json:"cloud_monitor_flags,omitempty"` // Whether to install cloud monitoring plug-in.
}

type AlibabaClusterCreateResponse struct {
	ClusterID string `json:"cluster_id"`
	RequestID string `json:"request_id"`
	TaskID    string `json:"task_id"`
}

type AlibabaDescribeClusterResponse struct {
	AgentVersion           string       `json:"agent_version"`            // The Agent version.
	ClusterID              string       `json:"cluster_id"`               // The cluster ID, which is the unique identifier of the cluster.
	Created                time.Time    `json:"created"`                  // The created time of the cluster.
	ExternalLoadbalancerID string       `json:"external_loadbalancer_id"` // The Server Load Balancer instance ID of the cluster.
	MasterURL              string       `json:"master_url"`               // The master address of the cluster, which is used to connect to the cluster to perform operations.
	Name                   string       `json:"name"`                     // The cluster name, which is specified when you create the cluster and is unique for each account.
	NetworkMode            string       `json:"network_mode"`             // The network mode of the cluster (Classic or Virtual Private Cloud (VPC)).
	RegionID               string       `json:"region_id"`                // The ID of the region in which the cluster resides.
	SecurityGroupID        string       `json:"security_group_id"`        // The security group ID.
	Size                   int          `json:"size"`                     // The number of nodes.
	State                  string       `json:"state"`                    // The cluster status.
	Updated                time.Time    `json:"updated"`                  // Last updated time.
	VPCID                  string       `json:"vpc_id"`                   // VPC ID.
	VSwitchID              string       `json:"vswitch_id"`               // VSwitch ID.
	ZoneID                 string       `json:"zone_id"`                  // Zone ID.
	Outputs                []outputItem `json:"outputs,omitempty"`
	KubernetesVersion      string       `json:"current_version"`
}

type outputItem struct {
	Description string
	OutputKey   string
	OutputValue interface{}
}

type AlibabaScaleClusterParams struct {
	DisableRollback    bool   `json:"disable_rollback,omitempty"` // Whether the failure is rolled back, true means that the failure does not roll back, and false fails to roll back. If you choose to fail back, it will release the resources produced during the creation process. It is not recommended to use false.
	TimeoutMins        int    `json:"timeout_mins,omitempty"`     // Cluster resource stack creation timeout in minutes, default value 60.
	WorkerInstanceType string `json:"worker_instance_type"`       // Worker node ECS specification type code.
	NumOfNodes         int    `json:"num_of_nodes"`               // Worker node number. The range is [0,300].
}

// AlibabaDescribeClusterLogResponseEntry is the response struct of a cluster log entry for api DescribeClusterLogs
type AlibabaDescribeClusterLogResponseEntry struct {
	ID        uint   `json:"ID"`
	ClusterID string `json:"cluster_id"`  // The cluster ID supplied by provider, which is the unique identifier of the cluster.
	Log       string `json:"cluster_log"` // Cluster log entry
	//LogLevel  interface{}    `json:"log_level"`
	Created time.Time `json:"created"` // The create time of the log entry
	Updated time.Time `json:"updated"` // The update time of the log entry
}
