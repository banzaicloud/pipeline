package constants

import "errors"

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
	TagGetClusterProfile     = "GetClusterProfile"
	TagSetClusterProfile     = "SetClusterProfile"
	TagUpdateClusterProfile  = "UpdateClusterProfile"
	TagDeleteClusterProfile  = "DeleteClusterProfile"
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

// ### [ Constants to Google cluster default values ] ### //
const (
	GoogleDefaultNodeCount = 1
)

// ### [ Constants to helm]
const (
	HELM_RETRY_ATTEMPT_CONFIG = "helm.retryAttempt"
	HELM_RETRY_SLEEP_SECONDS  = "helm.retrySleepSeconds"
)

// ### [ Constants to cloud types ] ### //
const (
	Amazon = "amazon"
	Azure  = "azure"
	Google = "google"
)

// ### [ Constants to table names ] ### //
const (
	TableNameClusters         = "clusters"
	TableNameAmazonProperties = "amazon_cluster_properties"
	TableNameAzureProperties  = "azure_cluster_properties"
	TableNameGoogleProperties = "google_cluster_properties"
)

// ### [ Errors ] ### //
var (
	ErrorNotSupportedCloudType      = errors.New("Not supported cloud type")
	ErrorAzureClusterNameRegexp     = errors.New("Only numbers, lowercase letters and underscores are allowed under name property. In addition, the value cannot end with an underscore, and must also be less than 32 characters long.")
	ErrorAzureClusterNameEmpty      = errors.New("The name should not be empty.")
	ErrorAzureClusterNameTooLong    = errors.New("Cluster name is greater than or equal 32")
	ErrorAzureCLusterStageFailed    = errors.New("cluster stage is 'Failed'")
	ErrorNotDifferentInterfaces     = errors.New("There is no change in data")
	ErrorReconcile                  = errors.New("Error during reconcile")
	ErrorEmptyUpdateRequest         = errors.New("Empty update cluster request")
	ErrorClusterNotReady            = errors.New("Cluster not ready yet")
	ErrorNilCluster                 = errors.New("<nil> cluster")
	ErrorWrongKubernetesVersion     = errors.New("Wrong kubernetes version for master/nodes. The required minimum kubernetes version is 1.8.x ")
	ErrorDifferentKubernetesVersion = errors.New("Different kubernetes version for master and nodes")
)
