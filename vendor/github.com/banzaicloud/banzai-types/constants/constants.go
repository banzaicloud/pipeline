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
	AzureDefaultKubernetesVersion = "1.9.2"
)

// ### [ Constants to Amazon cluster default values ] ### //
const (
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
	Dummy  = "dummy"
	BYOC   = "byoc"
)

// ### [ Constants to table names ] ### //
const (
	TableNameClusters         = "clusters"
	TableNameAmazonProperties = "amazon_cluster_properties"
	TableNameAzureProperties  = "azure_cluster_properties"
	TableNameGoogleProperties = "google_cluster_properties"
	TableNameDummyProperties  = "dummy_cluster_properties"
	TableNameBYOCProperties   = "byoc_cluster_properties"
)

// ### [ Errors ] ### //
var (
	ErrorNotSupportedCloudType      = errors.New("Not supported cloud type")
	ErrorAmazonClusterNameRegexp    = errors.New("Up to 255 letters (uppercase and lowercase), numbers, hyphens, and underscores are allowed.")
	ErrorGoogleClusterNameRegexp    = errors.New("Name must start with a lowercase letter followed by up to 40 lowercase letters, numbers, or hyphens, and cannot end with a hyphen.")
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
	ErrorLocationEmpty              = errors.New("Location field is empty")
	ErrorNodeInstanceTypeEmpty      = errors.New("NodeInstanceType field is empty")
)

// ### [ Regexps for cluster names ] ### //
const (
	RegexpAWSName = `^[A-z0-9-_]{1,255}$`
	RegexpAKSName = `^[a-z0-9_]{0,31}[a-z0-9]$`
	RegexpGKEName = `^[a-z]$|^[a-z][a-z0-9-]{0,38}[a-z0-9]$`
)
