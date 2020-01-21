module github.com/banzaicloud/pipeline

go 1.13

require (
	cloud.google.com/go v0.51.0
	cloud.google.com/go/storage v1.0.0
	emperror.dev/emperror v0.32.0
	emperror.dev/errors v0.7.0
	emperror.dev/handler/logur v0.4.0
	emperror.dev/handler/stackdriver v0.3.0
	github.com/Azure/azure-pipeline-go v0.2.2
	github.com/Azure/azure-sdk-for-go v38.0.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/Azure/go-autorest/autorest v0.9.3
	github.com/Azure/go-autorest/autorest/adal v0.8.1
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/Azure/go-autorest/autorest/validation v0.2.0
	github.com/DATA-DOG/go-sqlmock v1.4.0 // indirect
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/Masterminds/sprig/v3 v3.0.2
	github.com/ThreeDotsLabs/watermill v1.1.0
	github.com/aliyun/alibaba-cloud-sdk-go v1.60.327
	github.com/aliyun/aliyun-oss-go-sdk v2.0.5+incompatible
	github.com/antihax/optional v1.0.0
	github.com/aokoli/goutils v1.1.0
	github.com/apache/thrift v0.0.0-00010101000000-000000000000 // indirect
	github.com/asaskevich/EventBus v0.0.0-20180315140547-d46933a94f05
	github.com/aws/aws-sdk-go v1.28.0
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-00010101000000-000000000000 // indirect
	github.com/banzaicloud/anchore-image-validator v0.0.0-20190823121528-918b9fa6af62
	github.com/banzaicloud/bank-vaults/pkg/sdk v0.2.1
	github.com/banzaicloud/cicd-go v0.0.0-20190214150755-832df3e92677
	github.com/banzaicloud/gin-utilz v0.1.0
	github.com/banzaicloud/go-gin-prometheus v0.0.0-20181204122313-8145dbf52419
	github.com/banzaicloud/istio-operator v0.0.0-20191104140059-90d1290d7342
	github.com/banzaicloud/logging-operator/pkg/sdk v0.1.1
	github.com/banzaicloud/logrus-runtime-formatter v0.0.0-20180617171254-12df4a18567f
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/chai2010/gettext-go v0.0.0-20191225085308-6b9f4b1008e1 // indirect
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/coreos/go-oidc v2.1.0+incompatible
	github.com/crossdock/crossdock-go v0.0.0-20160816171116-049aabb0122b // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20191128021309-1d7a30a10f73 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/didip/tollbooth v4.0.2+incompatible
	github.com/docker/libcompose v0.4.0
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20191011121108-aa519ddbe484 // indirect
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/gin-contrib/cors v1.3.0
	github.com/gin-gonic/gin v1.5.0
	github.com/go-kit/kit v0.9.0
	github.com/go-openapi/spec v0.19.5 // indirect
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.2
	github.com/google/go-github v17.0.0+incompatible
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/sessions v1.2.0
	github.com/gosimple/slug v1.9.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/vault/api v1.0.4
	github.com/heptio/ark v0.9.3
	github.com/huandu/xstrings v1.2.1 // indirect
	github.com/iancoleman/orderedmap v0.0.0-20190318233801-ac98e3ecb4b0 // indirect
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/jinzhu/gorm v1.9.10
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.1
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af
	github.com/jmoiron/sqlx v1.2.0 // indirect
	github.com/lestrrat-go/backoff v1.0.0
	github.com/lib/pq v1.3.0 // indirect
	github.com/mattn/go-sqlite3 v2.0.2+incompatible // indirect
	github.com/microcosm-cc/bluemonday v1.0.2
	github.com/mitchellh/copystructure v1.0.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/moogar0880/problems v0.1.1
	github.com/oklog/run v1.1.0
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/oracle/oci-go-sdk v2.0.0+incompatible
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pelletier/go-toml v1.6.0
	github.com/pkg/errors v0.9.0
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
	github.com/prashantv/protectmem v0.0.0-20171002184600-e20412882b3a // indirect
	github.com/prometheus/client_golang v1.3.0
	github.com/prometheus/common v0.7.0
	github.com/qor/assetfs v0.0.0-20170713023933-ff57fdc13a14 // indirect
	github.com/qor/auth v0.0.0-20190103025640-46aae9fa92fa
	github.com/qor/mailer v0.0.0-20170814094430-1e6ac7106955 // indirect
	github.com/qor/middlewares v0.0.0-20170822143614-781378b69454 // indirect
	github.com/qor/qor v0.0.0-20190319081902-186b0237364b
	github.com/qor/redirect_back v0.0.0-20170907030740-b4161ed6f848 // indirect
	github.com/qor/render v0.0.0-20171201033449-63566e46f01b // indirect
	github.com/qor/responder v0.0.0-20160314063933-ecae0be66c1a // indirect
	github.com/qor/session v0.0.0-20170907035918-8206b0adab70
	github.com/robfig/cron v1.2.0 // indirect
	github.com/rubenv/sql-migrate v0.0.0-20200119084958-8794cecc920c // indirect
	github.com/sagikazarmark/appkit v0.4.0
	github.com/sagikazarmark/kitx v0.10.0
	github.com/sagikazarmark/ocmux v0.2.0
	github.com/samuel/go-thrift v0.0.0-20191111193933-5165175b40af // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.2
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/technosophos/moniker v0.0.0-20180509230615-a5dbd03a2245
	github.com/uber-go/mapdecode v1.0.0 // indirect
	github.com/uber-go/tally v3.3.13+incompatible // indirect
	github.com/uber/jaeger-client-go v2.21.1+incompatible // indirect
	github.com/uber/jaeger-lib v2.2.0+incompatible // indirect
	github.com/uber/tchannel-go v1.16.0 // indirect
	github.com/xanzy/go-gitlab v0.16.2-0.20190325100843-bbb1af7187c8
	go.opencensus.io v0.22.2
	go.uber.org/cadence v0.9.0
	go.uber.org/fx v1.10.0 // indirect
	go.uber.org/net/metrics v1.2.0 // indirect
	go.uber.org/thriftrw v1.21.0 // indirect
	go.uber.org/yarpc v1.36.1
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20191206172530-e9b2fee46413
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/api v0.15.0
	google.golang.org/genproto v0.0.0-20191230161307-f3c370f40bfb
	google.golang.org/grpc v1.26.0
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df // indirect
	gopkg.in/resty.v1 v1.12.0
	gopkg.in/yaml.v2 v2.2.5
	k8s.io/api v0.17.1
	k8s.io/apiextensions-apiserver v0.17.1
	k8s.io/apimachinery v0.17.1
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/cluster-bootstrap v0.17.1
	k8s.io/helm v2.16.1+incompatible
	k8s.io/kubectl v0.17.1 // indirect
	k8s.io/kubernetes v1.17.1
	logur.dev/adapter/logrus v0.4.1
	logur.dev/adapter/zap v0.4.1
	logur.dev/integration/watermill v0.4.2
	logur.dev/integration/zap v0.3.2
	logur.dev/logur v0.16.2
	sigs.k8s.io/controller-runtime v0.4.0
	sigs.k8s.io/kubefed v0.1.0-rc6
	sigs.k8s.io/testing_frameworks v0.1.2
	vbom.ml/util v0.0.0-20180919145318-efcd4e0f9787 // indirect
)

replace (
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.2
	github.com/apache/thrift => github.com/apache/thrift v0.0.0-20151001171628-53dd39833a08
	github.com/baiyubin/aliyun-sts-go-sdk => github.com/banzaicloud/aliyun-sts-go-sdk v0.0.0-20191023142834-57827dd1486a

	github.com/jinzhu/gorm => github.com/jinzhu/gorm v1.9.1
	github.com/qor/auth => github.com/banzaicloud/auth v0.1.3

	gopkg.in/yaml.v2 => github.com/banzaicloud/go-yaml v0.0.0-20190116151056-02e17e901182

	// Kubernetes 1.17.1
	k8s.io/api => k8s.io/api v0.17.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.1
	k8s.io/apiserver => k8s.io/apiserver v0.17.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.1
	k8s.io/client-go => k8s.io/client-go v0.17.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.1
	k8s.io/code-generator => k8s.io/code-generator v0.17.1
	k8s.io/component-base => k8s.io/component-base v0.17.1
	k8s.io/cri-api => k8s.io/cri-api v0.17.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.1
	k8s.io/helm => github.com/banzaicloud/helm v2.7.1-0.20200120004947-14c2e60cf8e2+incompatible
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.1
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.1
	k8s.io/kubectl => k8s.io/kubectl v0.17.1
	k8s.io/kubelet => k8s.io/kubelet v0.17.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.1
	k8s.io/metrics => k8s.io/metrics v0.17.1
	k8s.io/node-api => k8s.io/node-api v0.17.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.1
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.17.1
	k8s.io/sample-controller => k8s.io/sample-controller v0.17.1
	sigs.k8s.io/kubefed => github.com/banzaicloud/kubefed v0.1.0-rc2.0.20200121120031-f51ecb598a2d
)
