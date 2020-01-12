module github.com/banzaicloud/pipeline

go 1.13

require (
	cloud.google.com/go v0.51.0
	cloud.google.com/go/storage v1.0.0
	emperror.dev/emperror v0.21.3
	emperror.dev/errors v0.4.3
	emperror.dev/handler/logur v0.2.0
	emperror.dev/handler/stackdriver v0.1.1
	github.com/Azure/azure-pipeline-go v0.2.2
	github.com/Azure/azure-sdk-for-go v32.5.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.0.0-20181022225951-5152f14ace1c
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Azure/go-autorest/autorest v0.9.0
	github.com/Azure/go-autorest/autorest/adal v0.6.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.3.0
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/Azure/go-autorest/autorest/validation v0.2.0
	github.com/Masterminds/semver v1.4.2
	github.com/Masterminds/sprig v2.15.0+incompatible
	github.com/ThreeDotsLabs/watermill v1.0.2
	github.com/aliyun/alibaba-cloud-sdk-go v0.0.0-20191023062623-524ecb019aa6
	github.com/aliyun/aliyun-oss-go-sdk v2.0.3+incompatible
	github.com/anmitsu/go-shlex v0.0.0-20161002113705-648efa622239 // indirect
	github.com/antihax/optional v0.0.0-20180407024304-ca021399b1a6
	github.com/aokoli/goutils v1.0.1
	github.com/asaskevich/EventBus v0.0.0-20180315140547-d46933a94f05
	github.com/aws/aws-sdk-go v1.26.8
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/banzaicloud/anchore-image-validator v0.0.0-20190823121528-918b9fa6af62
	github.com/banzaicloud/bank-vaults/pkg/sdk v0.2.1
	github.com/banzaicloud/cicd-go v0.0.0-20190214150755-832df3e92677
	github.com/banzaicloud/gin-utilz v0.1.0
	github.com/banzaicloud/go-gin-prometheus v0.0.0-20181204122313-8145dbf52419
	github.com/banzaicloud/istio-operator v0.0.0-20191104140059-90d1290d7342
	github.com/banzaicloud/logging-operator/pkg/sdk v0.0.0-20191125142640-aa8071e64c9d
	github.com/banzaicloud/logrus-runtime-formatter v0.0.0-20180617171254-12df4a18567f
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/cenkalti/backoff v2.1.1+incompatible // indirect
	github.com/chai2010/gettext-go v0.0.0-20170215093142-bf70f2a70fb1 // indirect
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/coreos/go-oidc v2.0.0+incompatible
	github.com/crossdock/crossdock-go v0.0.0-20160816171116-049aabb0122b // indirect
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20190820223206-44cdfe8d8ba9 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/didip/tollbooth v4.0.0+incompatible
	github.com/docker/distribution v0.0.0-20180327202408-83389a148052 // indirect
	github.com/docker/docker v0.0.0-20170731201938-4f3616fb1c11 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/docker/libcompose v0.4.0
	github.com/docker/spdystream v0.0.0-20170912183627-bc6354cbbc29 // indirect
	github.com/elazarl/goproxy v0.0.0-20181111060418-2ce16c963a8a // indirect
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/fatih/structtag v1.0.0 // indirect
	github.com/flynn/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/gin-contrib/cors v0.0.0-20170318125340-cf4846e6a636
	github.com/gin-gonic/gin v1.4.0
	github.com/go-kit/kit v0.9.0
	github.com/go-openapi/spec v0.19.0 // indirect
	github.com/go-sql-driver/mysql v1.4.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/protobuf v1.3.2
	github.com/google/go-github v17.0.0+incompatible
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gorilla/mux v1.7.0
	github.com/gorilla/sessions v0.0.0-20181208214519-12bd4761fc66
	github.com/gosimple/slug v1.7.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/vault/api v1.0.4
	github.com/heptio/ark v0.9.3
	github.com/huandu/xstrings v1.0.0 // indirect
	github.com/jessevdk/go-flags v1.4.0 // indirect
	github.com/jinzhu/copier v0.0.0-20180308034124-7e38e58719c3
	github.com/jinzhu/gorm v1.9.10
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.0.1
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af
	github.com/kisielk/errcheck v1.2.0 // indirect
	github.com/kubernetes-sigs/kubefed v0.1.0-rc6
	github.com/lestrrat-go/backoff v0.0.0-20190107202757-0bc2a4274cd0
	github.com/lib/pq v0.0.0-20181016162627-9eb73efc1fcc // indirect
	github.com/mattn/go-sqlite3 v0.0.0-20160907162043-3fb7a0e792ed // indirect
	github.com/microcosm-cc/bluemonday v1.0.2
	github.com/mitchellh/copystructure v1.0.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/moogar0880/problems v0.1.1
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/oklog/run v1.0.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/oracle/oci-go-sdk v2.0.0+incompatible
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pborman/uuid v0.0.0-20180906182336-adf5a7427709 // indirect
	github.com/pelletier/go-toml v1.2.0
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
	github.com/prashantv/protectmem v0.0.0-20171002184600-e20412882b3a // indirect
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/common v0.4.1
	github.com/qor/assetfs v0.0.0-20170713023933-ff57fdc13a14 // indirect
	github.com/qor/auth v0.0.0-20190103025640-46aae9fa92fa
	github.com/qor/mailer v0.0.0-20170814094430-1e6ac7106955 // indirect
	github.com/qor/middlewares v0.0.0-20170822143614-781378b69454 // indirect
	github.com/qor/qor v0.0.0-20190319081902-186b0237364b
	github.com/qor/redirect_back v0.0.0-20170907030740-b4161ed6f848 // indirect
	github.com/qor/render v0.0.0-20171201033449-63566e46f01b // indirect
	github.com/qor/responder v0.0.0-20160314063933-ecae0be66c1a // indirect
	github.com/qor/session v0.0.0-20170907035918-8206b0adab70
	github.com/rainycape/unidecode v0.0.0-20150907023854-cb7f23ec59be // indirect
	github.com/robfig/cron v0.0.0-20180505203441-b41be1df6967 // indirect
	github.com/russross/blackfriday v1.5.1 // indirect
	github.com/sagikazarmark/kitx v0.3.0
	github.com/sagikazarmark/ocmux v0.2.0
	github.com/samuel/go-thrift v0.0.0-20160419172024-e9042807f4f5 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.6.1
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/technosophos/moniker v0.0.0-20180509230615-a5dbd03a2245
	github.com/uber-go/atomic v1.3.2 // indirect
	github.com/uber-go/mapdecode v1.0.0 // indirect
	github.com/uber-go/tally v3.3.7+incompatible // indirect
	github.com/uber/jaeger-client-go v2.17.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.0.0+incompatible // indirect
	github.com/uber/tchannel-go v1.12.0 // indirect
	github.com/ugorji/go v1.1.7 // indirect
	github.com/xanzy/go-gitlab v0.16.2-0.20190325100843-bbb1af7187c8
	github.com/xlab/handysort v0.0.0-20150421192137-fb3537ed64a1 // indirect
	go.opencensus.io v0.22.2
	go.uber.org/cadence v0.9.0
	go.uber.org/dig v1.7.0 // indirect
	go.uber.org/fx v1.9.0 // indirect
	go.uber.org/goleak v0.10.0 // indirect
	go.uber.org/net/metrics v1.0.1 // indirect
	go.uber.org/thriftrw v1.16.1 // indirect
	go.uber.org/yarpc v1.36.1
	go.uber.org/zap v1.12.0
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553
	golang.org/x/oauth2 v0.0.0-20191202225959-858c2ad4c8b6
	google.golang.org/api v0.15.0
	google.golang.org/genproto v0.0.0-20191230161307-f3c370f40bfb
	google.golang.org/grpc v1.26.0
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/gomail.v2 v2.0.0-20150902115704-41f357289737 // indirect
	gopkg.in/resty.v1 v1.12.0
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apiextensions-apiserver v0.0.0-20190918201827-3de75813f604
	k8s.io/apimachinery v0.0.0-20190823012420-8ca64af22337
	k8s.io/cli-runtime v0.0.0-20190404071300-cbd7455f4bce // indirect
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/cluster-bootstrap v0.0.0-20190404071559-03c28a85c7b7
	k8s.io/helm v2.12.2+incompatible
	k8s.io/kube-openapi v0.0.0-20190228160746-b3a7cee44a30 // indirect
	k8s.io/kubectl v0.0.0-20190523211420-5b63b0fd89bb // indirect
	k8s.io/kubernetes v1.13.5
	logur.dev/adapter/logrus v0.2.0
	logur.dev/adapter/zap v0.2.0
	logur.dev/integration/watermill v0.2.1
	logur.dev/integration/zap v0.2.0
	logur.dev/logur v0.15.0
	sigs.k8s.io/controller-runtime v0.2.0
	sigs.k8s.io/kubefed v0.1.0-rc6
	sigs.k8s.io/testing_frameworks v0.1.1
	vbom.ml/util v0.0.0-20170409195630-256737ac55c4 // indirect
)

replace (
	github.com/apache/thrift => github.com/apache/thrift v0.0.0-20151001171628-53dd39833a08
	github.com/baiyubin/aliyun-sts-go-sdk => github.com/banzaicloud/aliyun-sts-go-sdk v0.0.0-20191023142834-57827dd1486a
	github.com/jinzhu/gorm => github.com/jinzhu/gorm v1.9.1
	github.com/kubernetes-sigs/kubefed => github.com/kubernetes-sigs/kubefed v0.1.0-rc6
	github.com/qor/auth => github.com/banzaicloud/auth v0.1.3

	gopkg.in/yaml.v2 => github.com/banzaicloud/go-yaml v0.0.0-20190116151056-02e17e901182

	// Kubernetes 1.13.5
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190325193600-475668423e9f
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190319190228-a4358799e4fe
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190325194458-f2b4781c3ae1
	k8s.io/client-go => k8s.io/client-go v10.0.0+incompatible
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20180509051136-39cb288412c4
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.1.10
	sigs.k8s.io/kubefed => sigs.k8s.io/kubefed v0.1.0-rc6
)
