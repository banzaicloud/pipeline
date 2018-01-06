package constants

// ### [ Constants to log ] ### //
const (
	TagInit                  = "Init"
	TagCreateCluster         = "CreateCluster"
	TagValidateCreateCluster = "ValidateCreateCluster"
	TagValidateUpdateCluster = "ValidateUpdateCluster"
	TagGetClusterStatus      = "GetClusterStatus"
	TagUpdateCluster         = "UpdateCluster"
	TagGetCluster            = "GetCluster"
	TagDeleteCluster         = "DeleteCluster"
	TagDeleteDeployment      = "DeleteDeployment"
	TagCreateDeployment      = "CreateDeployment"
	TagListDeployments       = "ListDeployments"
	TagPrometheus            = "Prometheus"
	TagListClusters          = "ListClusters"
	TagGetClusterInfo        = "GetClusterInfo"
	TagFetchClusterConfig    = "FetchClusterConfig"
	TagGetTillerStatus       = "GetTillerStatus"
	TagFetchDeploymentStatus = "FetchDeploymentStatus"
	TagStatus                = "Status"
	TagSlack                 = "Slack"
	TagAuth                  = "Auth"
	TagDatabase              = "Database"
	TagKubernetes            = "Kubernetes"
	TagFormat                = "Format"
	TagHelmInstall           = "HelmInstall"
)

// ### [ Constants to Azure cluster default values ] ### //
const (
	AzureDefaultAgentCount        = 1
	AzureDefaultAgentName         = "agentpool1"
	AzureDefaultKubernetesVersion = "1.7.7"
)

// ### [ Constants to Amazon cluster default values ] ### //
const (
	AmazonDefaultNodeImage          = "ami-bdba13c4"
	AmazonDefaultMasterImage        = "ami-bdba13c4"
	AmazonDefaultMasterInstanceType = "m4.xlarge"
	AmazonDefaultNodeMinCount       = 1
	AmazonDefaultNodeMaxCount       = 1
	AmazonDefaultNodeSpotPrice      = "0.2"
)

// ### [ Constants to cloud types ] ### //
const (
	Amazon = "amazon"
	Azure  = "azure"
)

// ### [ Constants to table names ] ### //
const (
	TableNameClusters         = "clusters"
	TableNameAmazonProperties = "amazon_cluster_properties"
	TableNameAzureProperties  = "azure_cluster_properties"
)

// ### [ Constants to Response codes ] ### //
const (
	OK                = 200
	Created           = 201
	Accepted          = 202
	NoContent         = 204
	InternalErrorCode = 500
	BadRequest        = 400
	NotFound          = 404
)
