module github.com/banzaicloud/pipeline

go 1.15

require (
	cloud.google.com/go v0.51.0
	cloud.google.com/go/storage v1.0.0
	emperror.dev/emperror v0.32.0
	emperror.dev/errors v0.8.0
	emperror.dev/handler/logur v0.4.0
	github.com/Azure/azure-pipeline-go v0.2.3
	github.com/Azure/azure-sdk-for-go v44.2.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.10.0
	github.com/Azure/go-autorest/autorest v0.11.2
	github.com/Azure/go-autorest/autorest/adal v0.9.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.0
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.0
	github.com/MakeNowJust/heredoc v1.0.0
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/ThreeDotsLabs/watermill v1.1.0
	github.com/aliyun/aliyun-oss-go-sdk v2.0.5+incompatible
	github.com/antihax/optional v1.0.0
	github.com/aokoli/goutils v1.1.0
	github.com/asaskevich/EventBus v0.0.0-20180315140547-d46933a94f05
	github.com/aws/aws-sdk-go v1.35.20
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-00010101000000-000000000000 // indirect
	github.com/banzaicloud/anchore-image-validator v0.0.0-20190823121528-918b9fa6af62
	github.com/banzaicloud/bank-vaults/pkg/sdk v0.6.0
	github.com/banzaicloud/cadence-aws-sdk v0.0.0-20201104115745-5238cf77328b
	github.com/banzaicloud/gin-utilz v0.3.1
	github.com/banzaicloud/go-gin-prometheus v0.1.0
	github.com/banzaicloud/integrated-service-sdk v0.0.0
	github.com/banzaicloud/istio-operator v0.0.0-20200330114955-d15bdd228ae4
	github.com/banzaicloud/logging-operator/pkg/sdk v0.5.0
	github.com/banzaicloud/logrus-runtime-formatter v0.0.0-20180617171254-12df4a18567f
	github.com/banzaicloud/operator-tools v0.15.0
	github.com/banzaicloud/pipeline/pkg/sdk v0.0.1
	github.com/coreos/go-oidc v2.1.0+incompatible
	github.com/denisenkom/go-mssqldb v0.0.0-20200206145737-bbfc9a55622e // indirect
	github.com/dexidp/dex/api/v2 v2.0.0
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/gin-contrib/cors v1.3.0
	github.com/gin-gonic/gin v1.5.0
	github.com/go-kit/kit v0.10.0
	github.com/gofrs/flock v0.8.0
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/protobuf v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/sessions v1.2.1
	github.com/hashicorp/vault/api v1.0.4
	github.com/jinzhu/copier v0.0.0-20201025035756-632e723a6687
	github.com/jinzhu/gorm v1.9.16
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.1
	github.com/jmespath/go-jmespath v0.4.0
	github.com/jonboulle/clockwork v0.2.0
	github.com/json-iterator/go v1.1.10
	github.com/lestrrat-go/backoff v1.0.0
	github.com/mattn/go-sqlite3 v2.0.3+incompatible // indirect
	github.com/microcosm-cc/bluemonday v1.0.3
	github.com/mitchellh/copystructure v1.0.0
	github.com/mitchellh/mapstructure v1.3.3
	github.com/moogar0880/problems v0.1.1
	github.com/oklog/run v1.1.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pelletier/go-toml v1.6.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.10.0
	github.com/prometheus/prom2json v1.3.0
	github.com/sagikazarmark/appkit v0.8.0
	github.com/sagikazarmark/kitx v0.12.0
	github.com/sagikazarmark/ocmux v0.2.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/technosophos/moniker v0.0.0-20180509230615-a5dbd03a2245
	github.com/vmware-tanzu/velero v1.5.1
	github.com/vmware/govmomi v0.22.0
	go.opencensus.io v0.22.2
	go.uber.org/cadence v0.13.4
	go.uber.org/yarpc v1.45.0
	go.uber.org/zap v1.14.1
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	golang.org/x/net v0.0.0-20200822124328-c89045814202
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/api v0.15.1
	google.golang.org/genproto v0.0.0-20200731012542-8145dea6a485
	google.golang.org/grpc v1.31.0
	gopkg.in/resty.v1 v1.12.0
	gopkg.in/square/go-jose.v2 v2.5.1
	gopkg.in/yaml.v2 v2.3.0
	helm.sh/helm/v3 v3.4.0
	k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/cli-runtime v0.19.3
	k8s.io/client-go v0.19.3
	k8s.io/cluster-bootstrap v0.19.3
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.3.0
	k8s.io/kubernetes v1.19.3
	logur.dev/adapter/logrus v0.4.1
	logur.dev/adapter/zap v0.4.1
	logur.dev/integration/logr v0.4.0
	logur.dev/integration/watermill v0.4.2
	logur.dev/integration/zap v0.3.2
	logur.dev/logur v0.17.0
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/kubefed v0.4.0
	sigs.k8s.io/testing_frameworks v0.1.2
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/apache/thrift => github.com/apache/thrift v0.0.0-20151001171628-53dd39833a08
	github.com/baiyubin/aliyun-sts-go-sdk => github.com/banzaicloud/aliyun-sts-go-sdk v0.0.0-20191023142834-57827dd1486a
	github.com/banzaicloud/cadence-aws-sdk => ./lib/cadence-aws-sdk
	github.com/banzaicloud/integrated-service-sdk => github.com/banzaicloud/pipeline/lib/integrated-service-sdk v0.4.0
	github.com/banzaicloud/pipeline/pkg/sdk => ./pkg/sdk

	github.com/jinzhu/gorm => github.com/jinzhu/gorm v1.9.1

	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc10

	google.golang.org/grpc => google.golang.org/grpc v1.27.1

	// Kubernetes
	k8s.io/api => k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.3
	k8s.io/apiserver => k8s.io/apiserver v0.19.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.3
	k8s.io/client-go => k8s.io/client-go v0.19.3
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.3
	k8s.io/code-generator => k8s.io/code-generator v0.19.3
	k8s.io/component-base => k8s.io/component-base v0.19.3
	k8s.io/cri-api => k8s.io/cri-api v0.19.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.3
	k8s.io/kubectl => k8s.io/kubectl v0.19.3
	k8s.io/kubelet => k8s.io/kubelet v0.19.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.3
	k8s.io/metrics => k8s.io/metrics v0.19.3
	k8s.io/node-api => k8s.io/node-api v0.19.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.3
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.19.3
	k8s.io/sample-controller => k8s.io/sample-controller v0.19.3
)
