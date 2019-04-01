module github.com/banzaicloud/pipeline

require (
	cloud.google.com/go v0.33.1
	contrib.go.opencensus.io/exporter/ocagent v0.2.0 // indirect
	github.com/Azure/azure-pipeline-go v0.1.8
	github.com/Azure/azure-sdk-for-go v22.2.2+incompatible
	github.com/Azure/azure-storage-blob-go v0.0.0-20181022225951-5152f14ace1c
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Azure/go-autorest v11.2.8+incompatible
	github.com/BurntSushi/toml v0.3.0 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/Masterminds/semver v1.4.0
	github.com/Masterminds/sprig v2.14.1+incompatible
	github.com/Microsoft/go-winio v0.4.12 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/PuerkitoBio/purell v1.1.0 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/SAP/go-hdb v0.13.2 // indirect
	github.com/SermoDigital/jose v0.9.1 // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v0.0.0-20180822052843-1c5a1c93a9c1
	github.com/aliyun/aliyun-oss-go-sdk v0.0.0-20180615125516-36bf7aa2f916
	github.com/anmitsu/go-shlex v0.0.0-20161002113705-648efa622239 // indirect
	github.com/antihax/optional v0.0.0-20180407024304-ca021399b1a6
	github.com/aokoli/goutils v1.0.1
	github.com/apache/thrift v0.0.0-20151001171628-53dd39833a08 // indirect
	github.com/araddon/gou v0.0.0-20190110011759-c797efecbb61 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/asaskevich/EventBus v0.0.0-20180315140547-d46933a94f05
	github.com/asaskevich/govalidator v0.0.0-20180720115003-f9ffefc3facf // indirect
	github.com/aws/aws-sdk-go v1.16.11
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/banzaicloud/anchore-image-validator v0.0.0-20181204185657-bf9806201a4e
	github.com/banzaicloud/bank-vaults v0.0.0-20181228154154-8f1348d7821a
	github.com/banzaicloud/cicd-go v0.0.0-20190214150755-832df3e92677
	github.com/banzaicloud/go-gin-prometheus v0.0.0-20181204122313-8145dbf52419
	github.com/banzaicloud/istio-operator v0.0.0-20190312122926-a4debb5bafe9
	github.com/banzaicloud/logrus-runtime-formatter v0.0.0-20180617171254-12df4a18567f
	github.com/banzaicloud/nodepool-labels-operator v0.0.0-20190219103855-a13c1b05f240
	github.com/banzaicloud/prometheus-config v0.0.0-20181214142820-fc6ae4756a29
	github.com/bitly/go-hostpool v0.0.0-20171023180738-a3a6125de932 // indirect
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/boombuler/barcode v1.0.0 // indirect
	github.com/briankassouf/jose v0.9.1 // indirect
	github.com/cactus/go-statsd-client v3.1.1+incompatible // indirect
	github.com/cenkalti/backoff v2.1.1+incompatible // indirect
	github.com/census-instrumentation/opencensus-proto v0.1.0 // indirect
	github.com/centrify/cloud-golang-sdk v0.0.0-20190214225812-119110094d0f // indirect
	github.com/chai2010/gettext-go v0.0.0-20170215093142-bf70f2a70fb1 // indirect
	github.com/chrismalek/oktasdk-go v0.0.0-20181212195951-3430665dfaa0 // indirect
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/containerd/continuity v0.0.0-20181203112020-004b46473808 // indirect
	github.com/coreos/bbolt v1.3.2 // indirect
	github.com/coreos/go-oidc v2.0.0+incompatible
	github.com/coreos/go-systemd v0.0.0-20190212144455-93d5ec2c7f76 // indirect
	github.com/crossdock/crossdock-go v0.0.0-20160816171116-049aabb0122b // indirect
	github.com/dancannon/gorethink v4.0.0+incompatible // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20190204142019-df6d76eb9289 // indirect
	github.com/dexidp/dex v0.0.0-20190205125449-7bd4071b4c8c
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/didip/tollbooth v4.0.0+incompatible
	github.com/dimchansky/utfbom v1.0.0 // indirect
	github.com/docker/distribution v0.0.0-20180327202408-83389a148052 // indirect
	github.com/docker/docker v0.0.0-20170731201938-4f3616fb1c11 // indirect
	github.com/docker/go-connections v0.3.0 // indirect
	github.com/docker/go-units v0.3.2 // indirect
	github.com/docker/libcompose v0.4.0
	github.com/docker/spdystream v0.0.0-20170912183627-bc6354cbbc29 // indirect
	github.com/duosecurity/duo_api_golang v0.0.0-20190107154727-539434bf0d45 // indirect
	github.com/elazarl/go-bindata-assetfs v1.0.0 // indirect
	github.com/elazarl/goproxy v0.0.0-20181111060418-2ce16c963a8a // indirect
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/evanphx/json-patch v3.0.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/fatih/structtag v1.0.0 // indirect
	github.com/flynn/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/fullsailor/pkcs7 v0.0.0-20180613152042-8306686428a5 // indirect
	github.com/gammazero/deque v0.0.0-20190130191400-2afb3858e9c7 // indirect
	github.com/gammazero/workerpool v0.0.0-20181230203049-86a96b5d5d92 // indirect
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/gin-contrib/cors v0.0.0-20170318125340-cf4846e6a636
	github.com/gin-gonic/gin v1.3.1-0.20190204012700-5acf6601170b
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-ldap/ldap v3.0.1+incompatible // indirect
	github.com/go-logr/zapr v0.1.1 // indirect
	github.com/go-openapi/jsonpointer v0.0.0-20180322222829-3a0015ad55fa // indirect
	github.com/go-openapi/jsonreference v0.0.0-20180322222742-3fb327e6747d // indirect
	github.com/go-openapi/spec v0.0.0-20180326232708-9acd88844bc1 // indirect
	github.com/go-openapi/swag v0.0.0-20180302192843-ceb469cb0fdf // indirect
	github.com/go-stomp/stomp v2.0.2+incompatible // indirect
	github.com/go-test/deep v1.0.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gocql/gocql v0.0.0-20190219221429-ec4793573d14 // indirect
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/mock v1.2.0 // indirect
	github.com/golang/protobuf v1.2.0
	github.com/google/go-github v15.0.0+incompatible
	github.com/google/martian v2.1.0+incompatible // indirect
	github.com/google/uuid v1.1.0 // indirect
	github.com/googleapis/gax-go v2.0.0+incompatible // indirect
	github.com/goph/emperror v0.17.1
	github.com/goph/logur v0.11.0
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75 // indirect
	github.com/gorilla/sessions v0.0.0-20181208214519-12bd4761fc66
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/gosimple/slug v1.1.1 // indirect
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/hashicorp/consul v1.4.2 // indirect
	github.com/hashicorp/go-cleanhttp v0.0.0-20171218145408-d5fe4b57a186 // indirect
	github.com/hashicorp/go-gcp-common v0.0.0-20180425173946-763e39302965 // indirect
	github.com/hashicorp/go-memdb v0.0.0-20181108192425-032f93b25bec // indirect
	github.com/hashicorp/go-plugin v0.0.0-20190220160451-3f118e8ee104 // indirect
	github.com/hashicorp/go-retryablehttp v0.0.0-20180531211321-3b087ef2d313 // indirect
	github.com/hashicorp/go-rootcerts v0.0.0-20160503143440-6bb64b370b90 // indirect
	github.com/hashicorp/go-version v1.1.0 // indirect
	github.com/hashicorp/nomad v0.8.7 // indirect
	github.com/hashicorp/raft v1.0.0 // indirect
	github.com/hashicorp/serf v0.8.2 // indirect
	github.com/hashicorp/vault v1.0.1
	github.com/hashicorp/vault-plugin-auth-alicloud v0.0.0-20181109180636-f278a59ca3e8 // indirect
	github.com/hashicorp/vault-plugin-auth-azure v0.0.0-20190201222632-0af1d040b5b3 // indirect
	github.com/hashicorp/vault-plugin-auth-centrify v0.0.0-20180816201131-66b0a34a58bf // indirect
	github.com/hashicorp/vault-plugin-auth-gcp v0.0.0-20190201215414-7d4c2101e7d0 // indirect
	github.com/hashicorp/vault-plugin-auth-jwt v0.0.0-20190214185457-a61556be6730 // indirect
	github.com/hashicorp/vault-plugin-auth-kubernetes v0.0.0-20190201222209-db96aa4ab438 // indirect
	github.com/hashicorp/vault-plugin-secrets-ad v0.0.0-20190131222416-4796d9980125 // indirect
	github.com/hashicorp/vault-plugin-secrets-alicloud v0.0.0-20190131211812-b0abe36195cb // indirect
	github.com/hashicorp/vault-plugin-secrets-azure v0.0.0-20181207232500-0087bdef705a // indirect
	github.com/hashicorp/vault-plugin-secrets-gcp v0.0.0-20180921173200-d6445459e80c // indirect
	github.com/hashicorp/vault-plugin-secrets-gcpkms v0.0.0-20190116164938-d6b25b0b4a39 // indirect
	github.com/hashicorp/vault-plugin-secrets-kv v0.0.0-20190227052836-76a82948fe5b // indirect
	github.com/heptio/ark v0.9.3
	github.com/huandu/xstrings v1.0.0 // indirect
	github.com/jeffchao/backoff v0.0.0-20140404060208-9d7fd7aa17f2 // indirect
	github.com/jefferai/jsonx v1.0.0 // indirect
	github.com/jessevdk/go-flags v1.4.0 // indirect
	github.com/jinzhu/copier v0.0.0-20180308034124-7e38e58719c3
	github.com/jinzhu/gorm v1.9.1
	github.com/jinzhu/inflection v0.0.0-20180308033659-04140366298a // indirect
	github.com/jinzhu/now v0.0.0-20180314132004-b7dfa9a24504
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/keybase/go-crypto v0.0.0-20181127160227-255a5089e85a // indirect
	github.com/kisielk/errcheck v1.2.0 // indirect
	github.com/lestrrat-go/backoff v0.0.0-20190107202757-0bc2a4274cd0
	github.com/lib/pq v1.0.0 // indirect
	github.com/mailru/easyjson v0.0.0-20180323154445-8b799c424f57 // indirect
	github.com/mattbaird/elastigo v0.0.0-20170123220020-2fe47fd29e4b // indirect
	github.com/mattn/go-sqlite3 v1.10.0 // indirect
	github.com/michaelklishin/rabbit-hole v1.5.0 // indirect
	github.com/microcosm-cc/bluemonday v0.0.0-20180327211928-995366fdf961
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.0.0 // indirect
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/mitchellh/go-wordwrap v0.0.0-20150314170334-ad45545899c7 // indirect
	github.com/mitchellh/hashstructure v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/mitchellh/pointerstructure v0.0.0-20170205204203-f2329fcfa9e2 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20161129095857-cc309e4a2223 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/oklog/run v1.0.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/oracle/oci-go-sdk v2.0.0+incompatible
	github.com/ory-am/common v0.4.0 // indirect
	github.com/ory/dockertest v3.3.4+incompatible // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pborman/uuid v0.0.0-20180906182336-adf5a7427709 // indirect
	github.com/pelletier/go-toml v1.2.0
	github.com/pierrec/lz4 v0.0.0-20181005164709-635575b42742 // indirect
	github.com/pkg/errors v0.8.1
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
	github.com/pquerna/otp v1.1.0 // indirect
	github.com/prashantv/protectmem v0.0.0-20171002184600-e20412882b3a // indirect
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/common v0.0.0-20181126121408-4724e9255275
	github.com/qor/assetfs v0.0.0-20170713023933-ff57fdc13a14 // indirect
	github.com/qor/auth v0.0.0-20190103025640-46aae9fa92fa
	github.com/qor/mailer v0.0.0-20170814094430-1e6ac7106955 // indirect
	github.com/qor/middlewares v0.0.0-20170822143614-781378b69454 // indirect
	github.com/qor/qor v0.0.0-20180313040854-1fb0672178f1
	github.com/qor/redirect_back v0.0.0-20170907030740-b4161ed6f848 // indirect
	github.com/qor/render v0.0.0-20171201033449-63566e46f01b // indirect
	github.com/qor/responder v0.0.0-20160314063933-ecae0be66c1a // indirect
	github.com/qor/session v0.0.0-20170907035918-8206b0adab70
	github.com/rainycape/unidecode v0.0.0-20150907023854-cb7f23ec59be // indirect
	github.com/robfig/cron v0.0.0-20180505203441-b41be1df6967 // indirect
	github.com/russross/blackfriday v1.5.1 // indirect
	github.com/ryanuber/go-glob v0.0.0-20160226084822-572520ed46db // indirect
	github.com/samuel/go-thrift v0.0.0-20160419172024-e9042807f4f5 // indirect
	github.com/samuel/go-zookeeper v0.0.0-20180130194729-c4fab1ac1bec // indirect
	github.com/sirupsen/logrus v1.3.0
	github.com/smartystreets/goconvey v0.0.0-20190222223459-a17d461953aa // indirect
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/spf13/cast v1.3.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.3.1
	github.com/streadway/amqp v0.0.0-20190225234609-30f8ed68076e // indirect
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/technosophos/moniker v0.0.0-20180509230615-a5dbd03a2245
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/uber-common/bark v1.2.1
	github.com/uber-go/atomic v1.3.2 // indirect
	github.com/uber-go/mapdecode v1.0.0 // indirect
	github.com/uber-go/tally v3.3.7+incompatible // indirect
	github.com/uber/jaeger-client-go v2.15.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.0.0+incompatible // indirect
	github.com/uber/tchannel-go v1.12.0 // indirect
	github.com/xanzy/go-gitlab v0.16.2-0.20190325100843-bbb1af7187c8
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	github.com/xlab/handysort v0.0.0-20150421192137-fb3537ed64a1 // indirect
	go.etcd.io/bbolt v1.3.2 // indirect
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
	google.golang.org/genproto v0.0.0-20181202183823-bd91e49a0898
	google.golang.org/grpc v1.17.0
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/gomail.v2 v2.0.0-20150902115704-41f357289737 // indirect
	gopkg.in/gorethink/gorethink.v4 v4.1.0 // indirect
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
	gopkg.in/ory-am/dockertest.v2 v2.2.3 // indirect
	gopkg.in/yaml.v2 v2.2.2
	gotest.tools v2.2.0+incompatible // indirect
	honnef.co/go/tools v0.0.0-20190102054323-c2f93a96b099 // indirect
	k8s.io/api v0.0.0-20180712090710-2d6f90ab1293
	k8s.io/apiextensions-apiserver v0.0.0-20180328075702-9beab23b2663 // indirect
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/apiserver v0.0.0-20180327065226-f4a9d3132586 // indirect
	k8s.io/client-go v8.0.0+incompatible
	k8s.io/helm v2.11.0+incompatible
	k8s.io/kubernetes v1.11.5
	k8s.io/utils v0.0.0-20180208044234-258e2a2fa645 // indirect
	layeh.com/radius v0.0.0-20190118135028-0f678f039617 // indirect
	sigs.k8s.io/controller-runtime v0.1.10 // indirect
	sigs.k8s.io/testing_frameworks v0.1.1 // indirect
	vbom.ml/util v0.0.0-20170409195630-256737ac55c4 // indirect
)

replace github.com/qor/auth v0.0.0-20190103025640-46aae9fa92fa => github.com/banzaicloud/auth v0.1.1

replace gopkg.in/yaml.v2 => github.com/banzaicloud/go-yaml v0.0.0-20190116151056-02e17e901182
