module github.com/banzaicloud/pipeline

require (
	cloud.google.com/go v0.33.1
	github.com/Azure/azure-pipeline-go v0.1.8
	github.com/Azure/azure-sdk-for-go v23.2.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.0.0-20181022225951-5152f14ace1c
	github.com/Azure/go-autorest v11.2.8+incompatible
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/Masterminds/semver v1.4.2
	github.com/Masterminds/sprig v2.15.0+incompatible
	github.com/aliyun/alibaba-cloud-sdk-go v0.0.0-20190408123700-6ae3b7a159fd
	github.com/aliyun/aliyun-oss-go-sdk v0.0.0-20180615125516-36bf7aa2f916
	github.com/anmitsu/go-shlex v0.0.0-20161002113705-648efa622239 // indirect
	github.com/antihax/optional v0.0.0-20180407024304-ca021399b1a6
	github.com/aokoli/goutils v1.0.1
	github.com/apache/thrift v0.0.0-20151001171628-53dd39833a08 // indirect
	github.com/asaskevich/EventBus v0.0.0-20180315140547-d46933a94f05
	github.com/aws/aws-sdk-go v1.16.11
	github.com/banzaicloud/anchore-image-validator v0.0.0-20181204185657-bf9806201a4e
	github.com/banzaicloud/bank-vaults v0.0.0-20190408133613-dc8e83242b1b
	github.com/banzaicloud/cicd-go v0.0.0-20190214150755-832df3e92677
	github.com/banzaicloud/go-gin-prometheus v0.0.0-20181204122313-8145dbf52419
	github.com/banzaicloud/istio-operator v0.0.0-20190312122926-a4debb5bafe9
	github.com/banzaicloud/logrus-runtime-formatter v0.0.0-20180617171254-12df4a18567f
	github.com/banzaicloud/nodepool-labels-operator v0.0.0-20190219103855-a13c1b05f240
	github.com/banzaicloud/prometheus-config v0.0.0-20181214142820-fc6ae4756a29
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/cactus/go-statsd-client v3.1.1+incompatible // indirect
	github.com/chai2010/gettext-go v0.0.0-20170215093142-bf70f2a70fb1 // indirect
	github.com/coreos/go-oidc v2.0.0+incompatible
	github.com/crossdock/crossdock-go v0.0.0-20160816171116-049aabb0122b // indirect
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/dexidp/dex v0.0.0-20190205125449-7bd4071b4c8c
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/didip/tollbooth v4.0.0+incompatible
	github.com/docker/distribution v0.0.0-20180327202408-83389a148052 // indirect
	github.com/docker/docker v0.0.0-20170731201938-4f3616fb1c11 // indirect
	github.com/docker/libcompose v0.4.0
	github.com/docker/spdystream v0.0.0-20170912183627-bc6354cbbc29 // indirect
	github.com/elazarl/goproxy v0.0.0-20181111060418-2ce16c963a8a // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/fatih/structtag v1.0.0 // indirect
	github.com/flynn/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/gin-contrib/cors v0.0.0-20170318125340-cf4846e6a636
	github.com/gin-gonic/gin v1.3.1-0.20190402010134-2e915f4e5083
	github.com/go-logr/zapr v0.1.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/mock v1.2.0 // indirect
	github.com/golang/protobuf v1.3.0
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/uuid v1.1.0 // indirect
	github.com/goph/emperror v0.17.1
	github.com/goph/logur v0.11.0
	github.com/gorilla/sessions v0.0.0-20181208214519-12bd4761fc66
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0 // indirect
	github.com/hashicorp/vault v1.0.1
	github.com/heptio/ark v0.9.3
	github.com/jessevdk/go-flags v1.4.0 // indirect
	github.com/jinzhu/copier v0.0.0-20180308034124-7e38e58719c3
	github.com/jinzhu/gorm v1.9.1
	github.com/jinzhu/now v0.0.0-20180511015916-ed742868f2ae
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af
	github.com/kisielk/errcheck v1.2.0 // indirect
	github.com/lestrrat-go/backoff v0.0.0-20190107202757-0bc2a4274cd0
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/microcosm-cc/bluemonday v0.0.0-20180327211928-995366fdf961
	github.com/mitchellh/go-wordwrap v0.0.0-20150314170334-ad45545899c7 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/mwitkow/go-conntrack v0.0.0-20161129095857-cc309e4a2223 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/oklog/run v1.0.0
	github.com/oracle/oci-go-sdk v2.0.0+incompatible
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pelletier/go-toml v1.2.0
	github.com/pkg/errors v0.8.1
	github.com/prashantv/protectmem v0.0.0-20171002184600-e20412882b3a // indirect
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/common v0.0.0-20181126121408-4724e9255275
	github.com/qor/assetfs v0.0.0-20170713023933-ff57fdc13a14 // indirect
	github.com/qor/auth v0.0.0-20190103025640-46aae9fa92fa
	github.com/qor/mailer v0.0.0-20170814094430-1e6ac7106955 // indirect
	github.com/qor/middlewares v0.0.0-20170822143614-781378b69454 // indirect
	github.com/qor/qor v0.0.0-20180518090926-f171bc73933e
	github.com/qor/redirect_back v0.0.0-20170907030740-b4161ed6f848 // indirect
	github.com/qor/render v0.0.0-20171201033449-63566e46f01b // indirect
	github.com/qor/responder v0.0.0-20160314063933-ecae0be66c1a // indirect
	github.com/qor/session v0.0.0-20170907035918-8206b0adab70
	github.com/robfig/cron v0.0.0-20180505203441-b41be1df6967 // indirect
	github.com/russross/blackfriday v1.5.1 // indirect
	github.com/samuel/go-thrift v0.0.0-20160419172024-e9042807f4f5 // indirect
	github.com/sirupsen/logrus v1.3.0
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.3.1
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/technosophos/moniker v0.0.0-20180509230615-a5dbd03a2245
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/uber-common/bark v1.2.1
	github.com/uber-go/mapdecode v1.0.0 // indirect
	github.com/uber-go/tally v3.3.7+incompatible // indirect
	github.com/uber/jaeger-client-go v2.15.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.0.0+incompatible // indirect
	github.com/uber/tchannel-go v1.12.0 // indirect
	github.com/xanzy/go-gitlab v0.16.2-0.20190325100843-bbb1af7187c8
	github.com/xlab/handysort v0.0.0-20150421192137-fb3537ed64a1 // indirect
	go.uber.org/cadence v0.8.0
	go.uber.org/dig v1.7.0 // indirect
	go.uber.org/fx v1.9.0 // indirect
	go.uber.org/goleak v0.10.0 // indirect
	go.uber.org/net/metrics v1.0.1 // indirect
	go.uber.org/thriftrw v1.16.1 // indirect
	go.uber.org/tools v0.0.0-20170523140223-ce2550dad714 // indirect
	go.uber.org/yarpc v1.36.1
	go.uber.org/zap v1.9.1
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	golang.org/x/lint v0.0.0-20181217174547-8f45f776aaf1 // indirect
	golang.org/x/net v0.0.0-20190311183353-d8887717615a
	golang.org/x/oauth2 v0.0.0-20181203162652-d668ce993890
	golang.org/x/tools v0.0.0-20190318200714-bb1270c20edf // indirect
	google.golang.org/api v0.0.0-20190111181425-455dee39f703
	google.golang.org/appengine v1.5.0 // indirect
	google.golang.org/genproto v0.0.0-20181202183823-bd91e49a0898
	google.golang.org/grpc v1.17.0
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/gomail.v2 v2.0.0-20150902115704-41f357289737 // indirect
	gopkg.in/yaml.v2 v2.2.2
	gotest.tools v2.2.0+incompatible
	honnef.co/go/tools v0.0.0-20190102054323-c2f93a96b099 // indirect
	k8s.io/api v0.0.0-20190404065945-709cf190c7b7
	k8s.io/apimachinery v0.0.0-20190404065847-4a4abcd45006
	k8s.io/apiserver v0.0.0-20180327065226-f4a9d3132586 // indirect
	k8s.io/cli-runtime v0.0.0-20190404071300-cbd7455f4bce // indirect
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/cluster-bootstrap v0.0.0-20190404071559-03c28a85c7b7
	k8s.io/code-generator v0.0.0-20190311155051-e4c2b1329cf7 // indirect
	k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a // indirect
	k8s.io/helm v2.12.2+incompatible
	k8s.io/kubernetes v1.13.5
	k8s.io/utils v0.0.0-20190221042446-c2654d5206da // indirect
	vbom.ml/util v0.0.0-20170409195630-256737ac55c4 // indirect
)

replace (
	github.com/qor/auth v0.0.0-20190103025640-46aae9fa92fa => github.com/banzaicloud/auth v0.1.1
	gopkg.in/yaml.v2 => github.com/banzaicloud/go-yaml v0.0.0-20190116151056-02e17e901182

	// Kubernetes 1.13.5
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190325193600-475668423e9f
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190319190228-a4358799e4fe
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190325194458-f2b4781c3ae1
	k8s.io/client-go => k8s.io/client-go v10.0.0+incompatible
)
