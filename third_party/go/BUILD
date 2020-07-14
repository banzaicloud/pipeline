package(default_visibility = ["PUBLIC"])

go_module_download(
    name = "cloud.google.com_go",
    module = "cloud.google.com/go",
    sum = "h1:PvKAVQWCtlGUSlZkGW3QLelKaWq7KYv/MW1EboG8bfM=",
    tag = "download",
    version = "v0.51.0",
    deps = [],
)

go_get(
    name = "cloud.google.com_go_compute_metadata",
    src = ":_cloud.google.com_go#download",
    install = ["compute/metadata"],
    module = "cloud.google.com/go",
    deps = [],
)

go_get(
    name = "cloud.google.com_go_iam_credentials_apiv1",
    src = ":_cloud.google.com_go#download",
    install = ["iam/credentials/apiv1"],
    module = "cloud.google.com/go",
    deps = [
        ":github.com_googleapis_gax-go_v2",
        ":google.golang.org_api",
        ":google.golang.org_genproto_googleapis_iam_credentials_v1",
        ":google.golang.org_grpc",
    ],
)

go_get(
    name = "cloud.google.com_go_iam",
    src = ":_cloud.google.com_go#download",
    install = ["iam"],
    module = "cloud.google.com/go",
    deps = [
        ":github.com_googleapis_gax-go_v2",
        ":google.golang.org_genproto_googleapis_iam_v1",
        ":google.golang.org_grpc",
    ],
)

go_get(
    name = "cloud.google.com_go_internal",
    src = ":_cloud.google.com_go#download",
    install = ["internal"],
    module = "cloud.google.com/go",
    deps = [
        ":github.com_googleapis_gax-go_v2",
        ":google.golang.org_api",
        ":google.golang.org_grpc",
    ],
)

go_get(
    name = "cloud.google.com_go_internal_optional",
    src = ":_cloud.google.com_go#download",
    install = ["internal/optional"],
    module = "cloud.google.com/go",
    deps = [],
)

go_get(
    name = "cloud.google.com_go_internal_trace",
    src = ":_cloud.google.com_go#download",
    install = ["internal/trace"],
    module = "cloud.google.com/go",
    deps = [
        ":go.opencensus.io",
        ":google.golang.org_api",
        ":google.golang.org_genproto_googleapis_rpc_code",
        ":google.golang.org_grpc",
    ],
)

go_get(
    name = "cloud.google.com_go_internal_version",
    src = ":_cloud.google.com_go#download",
    install = ["internal/version"],
    module = "cloud.google.com/go",
    deps = [],
)

go_get(
    name = "cloud.google.com_go_errorreporting_apiv1beta1",
    src = ":_cloud.google.com_go#download",
    install = ["errorreporting/apiv1beta1"],
    module = "cloud.google.com/go",
    deps = [
        ":github.com_golang_protobuf",
        ":github.com_googleapis_gax-go_v2",
        ":google.golang.org_api",
        ":google.golang.org_genproto_googleapis_devtools_clouderrorreporting_v1beta1",
        ":google.golang.org_grpc",
    ],
)

go_get(
    name = "cloud.google.com_go_errorreporting",
    src = ":_cloud.google.com_go#download",
    install = ["errorreporting"],
    module = "cloud.google.com/go",
    deps = [
        ":cloud.google.com_go_errorreporting_apiv1beta1",
        ":cloud.google.com_go_internal_version",
        ":github.com_golang_protobuf",
        ":github.com_googleapis_gax-go_v2",
        ":google.golang.org_api",
        ":google.golang.org_genproto_googleapis_devtools_clouderrorreporting_v1beta1",
    ],
)

go_get(
    name = "cloud.google.com_go_storage",
    download_deps = [":_cloud.google.com_go#download"],
    install = [""],
    module = "cloud.google.com/go/storage",
    sum = "h1:VV2nUM3wwLLGh9lSABFgZMjInyUbJeaRSE64WuAIQ+4=",
    version = "v1.0.0",
    deps = [
        ":cloud.google.com_go_iam",
        ":cloud.google.com_go_internal",
        ":cloud.google.com_go_internal_optional",
        ":cloud.google.com_go_internal_trace",
        ":cloud.google.com_go_internal_version",
        ":github.com_googleapis_gax-go_v2",
        ":google.golang.org_api",
        ":google.golang.org_genproto_googleapis_iam_v1",
    ],
)

go_get(
    name = "emperror.dev_emperror",
    download_deps = [],
    install = [""],
    module = "emperror.dev/emperror",
    sum = "h1:DWdD6gwPaY1FYVmtfBhX/cP+u8g4/W0FdE6w32IerNc=",
    version = "v0.32.0",
    deps = [":emperror.dev_errors"],
)

go_get(
    name = "emperror.dev_errors",
    download_deps = [],
    install = [
        "",
        "match",
        "utils/keyval",
    ],
    module = "emperror.dev/errors",
    sum = "h1:vf0gZ0j9BJCXQYm0M21DPKeXy9PUjvX1YHXhNzIsqOY=",
    version = "v0.7.0",
    deps = [
        ":github.com_pkg_errors",
        ":go.uber.org_multierr",
    ],
)

go_get(
    name = "emperror.dev_handler_logur",
    download_deps = [],
    install = [""],
    module = "emperror.dev/handler/logur",
    sum = "h1:mo12Nr3Ke3llow2lNi9PuS0h4vy24yDoAQ8bw98v1pY=",
    version = "v0.4.0",
    deps = [
        ":emperror.dev_errors",
        ":logur.dev_logur",
    ],
)

go_get(
    name = "emperror.dev_handler_stackdriver",
    download_deps = [],
    install = [""],
    module = "emperror.dev/handler/stackdriver",
    sum = "h1:wUoZUU7ee/UsDWLnwLpkOPrNp7lQpZ1pQfN507QNGfs=",
    version = "v0.3.0",
    deps = [
        ":cloud.google.com_go_errorreporting",
        ":emperror.dev_errors",
    ],
)

go_get(
    name = "github.com_Azure_azure-pipeline-go",
    download_deps = [],
    install = ["pipeline"],
    module = "github.com/Azure/azure-pipeline-go",
    sum = "h1:6oiIS9yaG6XCCzhgAgKFfIWyo4LLCiDhZot6ltoThhY=",
    version = "v0.2.2",
    deps = [":github.com_mattn_go-ieproxy"],
)

go_get(
    name = "github.com_Azure_azure-sdk-for-go",
    download_deps = [],
    install = [
        "version",
        "services/authorization/mgmt/2015-07-01/authorization",
        "services/compute/mgmt/2018-10-01/compute",
        "services/containerservice/mgmt/2020-02-01/containerservice",
        "services/network/mgmt/2018-10-01/network",
        "services/preview/monitor/mgmt/2018-09-01/insights",
        "services/resources/mgmt/2016-06-01/subscriptions",
        "services/resources/mgmt/2018-02-01/resources",
        "services/storage/mgmt/2017-10-01/storage",
        "profiles/latest/authorization/mgmt/authorization",
    ],
    module = "github.com/Azure/azure-sdk-for-go",
    sum = "h1:zeY2FNH9mauj53XcRJQ2JVSNH0M+eYxeWhep8nnhNoc=",
    version = "v39.2.0+incompatible",
    deps = [
        ":github.com_Azure_go-autorest_autorest",
        ":github.com_Azure_go-autorest_autorest_date",
        ":github.com_Azure_go-autorest_autorest_to",
        ":github.com_Azure_go-autorest_autorest_validation",
        ":github.com_Azure_go-autorest_tracing",
    ],
)

go_get(
    name = "github.com_Azure_azure-storage-blob-go",
    download_deps = [],
    install = ["azblob"],
    module = "github.com/Azure/azure-storage-blob-go",
    sum = "h1:53qhf0Oxa0nOjgbDeeYPUeyiNmafAFEY95rZLK0Tj6o=",
    version = "v0.8.0",
    deps = [":github.com_Azure_azure-pipeline-go"],
)

go_get(
    name = "github.com_Azure_go-autorest_autorest",
    download_deps = [],
    install = [
        "",
        "azure",
    ],
    module = "github.com/Azure/go-autorest/autorest",
    sum = "h1:OZEIaBbMdUE/Js+BQKlpO81XlISgipr6yDJ+PSwsgi4=",
    version = "v0.9.3",
    deps = [
        ":github.com_Azure_go-autorest_autorest_adal",
        ":github.com_Azure_go-autorest_logger",
        ":github.com_Azure_go-autorest_tracing",
    ],
)

go_get(
    name = "github.com_Azure_go-autorest_autorest_adal",
    download_deps = [":_github.com_Azure_go-autorest_autorest#download"],
    install = [""],
    module = "github.com/Azure/go-autorest/autorest/adal",
    sum = "h1:pZdL8o72rK+avFWl+p9nE8RWi1JInZrWJYlnpfXJwHk=",
    version = "v0.8.1",
    deps = [
        ":github.com_Azure_go-autorest_autorest_date",
        ":github.com_Azure_go-autorest_tracing",
        ":github.com_dgrijalva_jwt-go",
    ],
)

go_get(
    name = "github.com_Azure_go-autorest_autorest_azure_auth",
    download_deps = [":_github.com_Azure_go-autorest_autorest#download"],
    install = [""],
    module = "github.com/Azure/go-autorest/autorest/azure/auth",
    sum = "h1:iM6UAvjR97ZIeR93qTcwpKNMpV+/FTWjwEbuPD495Tk=",
    version = "v0.4.2",
    deps = [
        ":github.com_Azure_go-autorest_autorest",
        ":github.com_Azure_go-autorest_autorest_adal",
        ":github.com_Azure_go-autorest_autorest_azure_cli",
        ":github.com_dimchansky_utfbom",
        ":golang.org_x_crypto",
    ],
)

go_get(
    name = "github.com_Azure_go-autorest_autorest_azure_cli",
    download_deps = [":_github.com_Azure_go-autorest_autorest#download"],
    install = [""],
    module = "github.com/Azure/go-autorest/autorest/azure/cli",
    sum = "h1:LXl088ZQlP0SBppGFsRZonW6hSvwgL5gRByMbvUbx8U=",
    version = "v0.3.1",
    deps = [
        ":github.com_Azure_go-autorest_autorest_adal",
        ":github.com_Azure_go-autorest_autorest_date",
        ":github.com_dimchansky_utfbom",
        ":github.com_mitchellh_go-homedir",
    ],
)

go_get(
    name = "github.com_Azure_go-autorest_autorest_date",
    download_deps = [":_github.com_Azure_go-autorest_autorest#download"],
    install = [""],
    module = "github.com/Azure/go-autorest/autorest/date",
    sum = "h1:yW+Zlqf26583pE43KhfnhFcdmSWlm5Ew6bxipnr/tbM=",
    version = "v0.2.0",
    deps = [],
)

go_get(
    name = "github.com_Azure_go-autorest_autorest_to",
    download_deps = [":_github.com_Azure_go-autorest_autorest#download"],
    install = [""],
    module = "github.com/Azure/go-autorest/autorest/to",
    sum = "h1:zebkZaadz7+wIQYgC7GXaz3Wb28yKYfVkkBKwc38VF8=",
    version = "v0.3.0",
    deps = [],
)

go_get(
    name = "github.com_Azure_go-autorest_autorest_validation",
    download_deps = [":_github.com_Azure_go-autorest_autorest#download"],
    install = [""],
    module = "github.com/Azure/go-autorest/autorest/validation",
    sum = "h1:15vMO4y76dehZSq7pAaOLQxC6dZYsSrj2GQpflyM/L4=",
    version = "v0.2.0",
    deps = [],
)

go_get(
    name = "github.com_Azure_go-autorest_logger",
    download_deps = [],
    install = [""],
    module = "github.com/Azure/go-autorest/logger",
    sum = "h1:ruG4BSDXONFRrZZJ2GUXDiUyVpayPmb1GnWeHDdaNKY=",
    version = "v0.1.0",
    deps = [],
)

go_get(
    name = "github.com_Azure_go-autorest_tracing",
    download_deps = [],
    install = [""],
    module = "github.com/Azure/go-autorest/tracing",
    sum = "h1:TRn4WjSnkcSy5AEG3pnbtFSwNtwzjr4VYyQflFE619k=",
    version = "v0.5.0",
    deps = [],
)

go_get(
    name = "github.com_BurntSushi_toml",
    download_deps = [],
    install = [""],
    module = "github.com/BurntSushi/toml",
    sum = "h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=",
    version = "v0.3.1",
    deps = [],
)

go_get(
    name = "github.com_MakeNowJust_heredoc",
    download_deps = [],
    install = [""],
    module = "github.com/MakeNowJust/heredoc",
    sum = "h1:cXCdzVdstXyiTqTvfqk9SDHpKNjxuom+DOlyEeQ4pzQ=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "github.com_Masterminds_goutils",
    download_deps = [],
    install = [""],
    module = "github.com/Masterminds/goutils",
    sum = "h1:zukEsf/1JZwCMgHiK3GZftabmxiCw4apj3a28RPBiVg=",
    version = "v1.1.0",
    deps = [],
)

go_get(
    name = "github.com_Masterminds_semver",
    download_deps = [],
    install = [""],
    module = "github.com/Masterminds/semver",
    sum = "h1:H65muMkzWKEuNDnfl9d70GUjFniHKHRbFPGBuZ3QEww=",
    version = "v1.5.0",
    deps = [],
)

go_get(
    name = "github.com_Masterminds_semver_v3",
    download_deps = [":_github.com_Masterminds_semver#download"],
    install = [""],
    module = "github.com/Masterminds/semver/v3",
    sum = "h1:znjIyLfpXEDQjOIEWh+ehwpTU14UzUPub3c3sm36u14=",
    version = "v3.0.3",
    deps = [],
)

go_get(
    name = "github.com_Masterminds_sprig",
    download_deps = [],
    install = [""],
    module = "github.com/Masterminds/sprig",
    sum = "h1:z4yfnGrZ7netVz+0EDJ0Wi+5VZCSYp4Z0m2dk6cEM60=",
    version = "v2.22.0+incompatible",
    deps = [
        ":github.com_Masterminds_goutils",
        ":github.com_Masterminds_semver",
        ":github.com_google_uuid",
        ":github.com_huandu_xstrings",
        ":github.com_imdario_mergo",
        ":github.com_mitchellh_copystructure",
        ":golang.org_x_crypto",
    ],
)

go_get(
    name = "github.com_Masterminds_sprig_v3",
    download_deps = [":_github.com_Masterminds_sprig#download"],
    install = [""],
    module = "github.com/Masterminds/sprig/v3",
    sum = "h1:wz22D0CiSctrliXiI9ZO3HoNApweeRGftyDN+BQa3B8=",
    version = "v3.0.2",
    deps = [
        ":github.com_Masterminds_goutils",
        ":github.com_Masterminds_semver_v3",
        ":github.com_google_uuid",
        ":github.com_huandu_xstrings",
        ":github.com_imdario_mergo",
        ":github.com_mitchellh_copystructure",
        ":github.com_spf13_cast",
        ":golang.org_x_crypto",
    ],
)

go_get(
    name = "github.com_PuerkitoBio_purell",
    download_deps = [],
    install = [""],
    module = "github.com/PuerkitoBio/purell",
    sum = "h1:WEQqlqaGbrPkxLJWfBwQmfEAE1Z7ONdDLqrN38tNFfI=",
    version = "v1.1.1",
    deps = [
        ":github.com_PuerkitoBio_urlesc",
        ":golang.org_x_net",
        ":golang.org_x_text",
    ],
)

go_get(
    name = "github.com_PuerkitoBio_urlesc",
    download_deps = [],
    install = [""],
    module = "github.com/PuerkitoBio/urlesc",
    sum = "h1:d+Bc7a5rLufV/sSk/8dngufqelfh6jnri85riMAaF/M=",
    version = "v0.0.0-20170810143723-de5bf2ad4578",
    deps = [],
)

go_get(
    name = "github.com_ThreeDotsLabs_watermill",
    download_deps = [],
    install = [
        "",
        "internal",
        "pubsub/sync",
        "message",
        "components/cqrs",
        "message/router/middleware",
        "pubsub/gochannel",
        "message/subscriber",
    ],
    module = "github.com/ThreeDotsLabs/watermill",
    sum = "h1:RWVfySGHEaK4TZhr8L/rKkYkSbyzWRzE8ut7QP7esLY=",
    version = "v1.1.0",
    deps = [
        ":github.com_cenkalti_backoff_v3",
        ":github.com_gogo_protobuf",
        ":github.com_google_uuid",
        ":github.com_hashicorp_go-multierror",
        ":github.com_lithammer_shortuuid_v3",
        ":github.com_oklog_ulid",
        ":github.com_pkg_errors",
    ],
)

go_get(
    name = "github.com_aliyun_alibaba-cloud-sdk-go",
    download_deps = [],
    install = [
        "sdk/auth/credentials",
        "sdk/utils",
        "sdk/errors",
        "sdk/requests",
        "sdk/responses",
        "sdk/auth/signers",
        "sdk/auth",
        "sdk/auth/credentials/provider",
        "sdk/endpoints",
        "sdk",
        "services/vpc",
        "services/cs",
        "services/ecs",
        "services/ess",
    ],
    module = "github.com/aliyun/alibaba-cloud-sdk-go",
    sum = "h1:MZrGWhrO7yXMOv+Ul0V/yXzxQ3Pso6WGy+SNP2102+c=",
    version = "v1.60.327",
    deps = [
        ":github.com_jmespath_go-jmespath",
        ":github.com_json-iterator_go",
        ":gopkg.in_ini.v1",
    ],
)

go_get(
    name = "github.com_aliyun_aliyun-oss-go-sdk",
    download_deps = [],
    install = ["oss"],
    module = "github.com/aliyun/aliyun-oss-go-sdk",
    sum = "h1:A3oZlWPD/Poa19FvNbw+Zu4yKAurDBTjlRDilYGBiS4=",
    version = "v2.0.5+incompatible",
    deps = [":golang.org_x_time"],
)

go_get(
    name = "github.com_antihax_optional",
    download_deps = [],
    install = [""],
    module = "github.com/antihax/optional",
    sum = "h1:xK2lYat7ZLaVVcIuj82J8kIro4V6kDe0AUDFboUCwcg=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "github.com_aokoli_goutils",
    download_deps = [],
    install = [""],
    module = "github.com/aokoli/goutils",
    sum = "h1:jy4ghdcYvs5EIoGssZNslIASX5m+KNMfyyKvRQ0TEVE=",
    version = "v1.1.0",
    deps = [],
)

go_get(
    name = "github.com_apache_thrift",
    download_deps = [],
    install = ["lib/go/thrift"],
    module = "github.com/apache/thrift",
    replace = "github.com/apache/thrift",
    sum = "h1:vhEK14bs3+zQEDMftA8BVc7uwAIkoqK/kGtoqs8kjks=",
    version = "v0.0.0-20151001171628-53dd39833a08",
    deps = [],
)

go_get(
    name = "github.com_asaskevich_EventBus",
    download_deps = [],
    install = [""],
    module = "github.com/asaskevich/EventBus",
    sum = "h1:Shem5lRG4gJyrrg9YMIl7dOQazyWCq0Daz4LjompZ28=",
    version = "v0.0.0-20180315140547-d46933a94f05",
    deps = [],
)

go_get(
    name = "github.com_asaskevich_govalidator",
    download_deps = [],
    install = [""],
    module = "github.com/asaskevich/govalidator",
    sum = "h1:zV3ejI06GQ59hwDQAvmK1qxOQGB3WuVTRoY0okPTAv0=",
    version = "v0.0.0-20200108200545-475eaeb16496",
    deps = [],
)

go_get(
    name = "github.com_aws_aws-sdk-go",
    download_deps = [],
    install = [
        "aws/awserr",
        "internal/ini",
        "internal/shareddefaults",
        "aws/credentials",
        "aws/endpoints",
        "internal/sdkio",
        "aws",
        "aws/client/metadata",
        "aws/awsutil",
        "aws/request",
        "internal/sdkrand",
        "aws/client",
        "aws/corehandlers",
        "aws/credentials/processcreds",
        "internal/strings",
        "internal/sdkmath",
        "private/protocol",
        "private/protocol/rest",
        "aws/signer/v4",
        "private/protocol/query/queryutil",
        "private/protocol/xml/xmlutil",
        "private/protocol/query",
        "service/sts",
        "service/sts/stsiface",
        "aws/credentials/stscreds",
        "aws/csm",
        "internal/sdkuri",
        "aws/ec2metadata",
        "aws/credentials/ec2rolecreds",
        "private/protocol/json/jsonutil",
        "aws/credentials/endpointcreds",
        "aws/defaults",
        "aws/session",
        "private/protocol/ec2query",
        "service/ec2",
        "service/iam",
        "service/iam/iamiface",
        "service/cloudformation",
        "aws/arn",
        "internal/s3err",
        "private/protocol/eventstream",
        "private/protocol/eventstream/eventstreamapi",
        "private/protocol/restxml",
        "service/s3/internal/arn",
        "service/s3",
        "service/s3/s3iface",
        "service/s3/s3manager",
        "service/autoscaling",
        "private/protocol/jsonrpc",
        "private/protocol/restjson",
        "service/eks",
        "service/elb",
        "service/ec2/ec2iface",
        "service/route53",
        "service/route53/route53iface",
    ],
    module = "github.com/aws/aws-sdk-go",
    sum = "h1:NkmnHFVEMTRYTleRLm5xUaL1mHKKkYQl4rCd+jzD58c=",
    version = "v1.28.0",
    deps = [":github.com_jmespath_go-jmespath"],
)

go_get(
    name = "github.com_banzaicloud_anchore-image-validator",
    download_deps = [],
    install = ["pkg/apis/security/v1alpha1"],
    module = "github.com/banzaicloud/anchore-image-validator",
    sum = "h1:wlvTg+btN2gyZrKDOoCpAz+tJLnc072UbTfFTtEdnZk=",
    version = "v0.0.0-20190823121528-918b9fa6af62",
    deps = [":k8s.io_apimachinery"],
)

go_get(
    name = "github.com_banzaicloud_bank-vaults_pkg_sdk",
    download_deps = [],
    install = [
        "vault",
        "auth",
        "db",
        "tls",
    ],
    module = "github.com/banzaicloud/bank-vaults/pkg/sdk",
    sum = "h1:YBo1RVbiq/o14Z8x2K/hH0qF2iFhe1OBphoPwLggTa4=",
    version = "v0.3.1",
    deps = [
        ":emperror.dev_errors",
        ":github.com_fsnotify_fsnotify",
        ":github.com_hashicorp_hcl",
        ":github.com_hashicorp_vault_api",
        ":github.com_hashicorp_vault_sdk",
        ":github.com_json-iterator_go",
        ":github.com_mitchellh_mapstructure",
        ":github.com_sirupsen_logrus",
        ":github.com_spf13_cast",
        ":github.com_spf13_viper",
        ":k8s.io_api",
        ":k8s.io_client-go",
        ":sigs.k8s.io_controller-runtime",
    ],
)

go_get(
    name = "github.com_banzaicloud_gin-utilz",
    download_deps = [],
    install = ["auth"],
    module = "github.com/banzaicloud/gin-utilz",
    sum = "h1:j6wLdZSlw36g63yklk11ihnIKqoU2zFfXIcaL1dumuM=",
    version = "v0.2.0",
    deps = [
        ":emperror.dev_errors",
        ":github.com_dgrijalva_jwt-go",
        ":github.com_gin-gonic_gin",
    ],
)

go_get(
    name = "github.com_banzaicloud_go-gin-prometheus",
    download_deps = [],
    install = [""],
    module = "github.com/banzaicloud/go-gin-prometheus",
    sum = "h1:zetacN1k3svVc7DtzGfpHA49uO0G8cwsiP5qSuUUQKs=",
    version = "v0.1.0",
    deps = [
        ":github.com_gin-gonic_gin",
        ":github.com_prometheus_client_golang",
        ":github.com_sirupsen_logrus",
    ],
)

go_get(
    name = "github.com_banzaicloud_istio-operator",
    download_deps = [],
    install = [
        "pkg/util",
        "pkg/apis/istio/v1beta1",
    ],
    module = "github.com/banzaicloud/istio-operator",
    sum = "h1:nz020decQ0q8AkYzrnc/NZTbcuzw9B8N+s7LW5DA+3E=",
    version = "v0.0.0-20200330114955-d15bdd228ae4",
    deps = [
        ":k8s.io_api",
        ":k8s.io_apimachinery",
        ":sigs.k8s.io_controller-runtime",
    ],
)

go_get(
    name = "github.com_banzaicloud_k8s-objectmatcher",
    download_deps = [],
    install = ["patch"],
    module = "github.com/banzaicloud/k8s-objectmatcher",
    sum = "h1:HhXkOWg4xmAW203p3G4Av9ylUfkUJF1+8MK90XQAaJw=",
    version = "v1.3.2",
    deps = [
        ":github.com_goph_emperror",
        ":github.com_json-iterator_go",
        ":github.com_pkg_errors",
        ":k8s.io_api",
        ":k8s.io_apimachinery",
    ],
)

go_get(
    name = "github.com_banzaicloud_logging-operator_pkg_sdk",
    download_deps = [],
    install = [
        "model/types",
        "model/filter",
        "model/common",
        "model/output",
        "api/v1beta1",
    ],
    module = "github.com/banzaicloud/logging-operator/pkg/sdk",
    sum = "h1:96DCkcHkbDgaCLpvzrdyQ/lzzntOSp/Wklg6MTYI5GQ=",
    version = "v0.3.15",
    deps = [
        ":emperror.dev_errors",
        ":github.com_banzaicloud_operator-tools",
        ":github.com_spf13_cast",
        ":k8s.io_api",
        ":k8s.io_apimachinery",
        ":sigs.k8s.io_controller-runtime",
    ],
)

go_get(
    name = "github.com_banzaicloud_logrus-runtime-formatter",
    download_deps = [],
    install = [""],
    module = "github.com/banzaicloud/logrus-runtime-formatter",
    sum = "h1:WeGZeO6I+lI1IG2vm1iXTeMrHLcMUjfoYm+rubBM5c4=",
    version = "v0.0.0-20180617171254-12df4a18567f",
    deps = [":github.com_sirupsen_logrus"],
)

go_get(
    name = "github.com_banzaicloud_operator-tools",
    download_deps = [],
    install = [
        "pkg/secret",
        "pkg/utils",
        "pkg/volume",
    ],
    module = "github.com/banzaicloud/operator-tools",
    sum = "h1:Nv+nhPs2EjO5L4YjgSsuMVJKl81xMZh/T3zGNBKahFE=",
    version = "v0.10.2",
    deps = [
        ":emperror.dev_errors",
        ":github.com_go-logr_logr",
        ":github.com_iancoleman_orderedmap",
        ":github.com_spf13_cast",
        ":k8s.io_api",
        ":k8s.io_apimachinery",
        ":sigs.k8s.io_controller-runtime",
    ],
)

go_get(
    name = "github.com_beorn7_perks",
    download_deps = [],
    install = ["quantile"],
    module = "github.com/beorn7/perks",
    sum = "h1:VlbKKnNfV8bJzeqoa4cOKqO6bYr3WgKZxO8Z16+hsOM=",
    version = "v1.0.1",
    deps = [],
)

go_get(
    name = "github.com_cenkalti_backoff_v3",
    download_deps = [],
    install = [""],
    module = "github.com/cenkalti/backoff/v3",
    sum = "h1:ske+9nBpD9qZsTBoF41nW5L+AIuFBKMeze18XQ3eG1c=",
    version = "v3.0.0",
    deps = [],
)

go_get(
    name = "github.com_cespare_xxhash_v2",
    download_deps = [],
    install = [""],
    module = "github.com/cespare/xxhash/v2",
    sum = "h1:6MnRN8NT7+YBpUIWxHtefFZOKTAPgGjpQSxqLNn0+qY=",
    version = "v2.1.1",
    deps = [],
)

go_get(
    name = "github.com_chai2010_gettext-go",
    download_deps = [],
    install = [
        "gettext/mo",
        "gettext/plural",
        "gettext/po",
        "gettext",
    ],
    module = "github.com/chai2010/gettext-go",
    sum = "h1:7aWHqerlJ41y6FOsEUvknqgXnGmJyJSbjhAWq5pO4F8=",
    version = "v0.0.0-20160711120539-c6fed771bfd5",
    deps = [],
)

go_get(
    name = "github.com_containerd_containerd",
    download_deps = [],
    install = [
        "errdefs",
        "content",
        "log",
        "platforms",
        "images",
        "remotes",
        "labels",
        "reference",
        "archive/compression",
        "remotes/docker/schema1",
        "version",
        "remotes/docker",
        "filters",
        "sys",
        "content/local",
    ],
    module = "github.com/containerd/containerd",
    sum = "h1:ForxmXkA6tPIvffbrDAcPUIB32QgXkt2XFj+F0UxetA=",
    version = "v1.3.2",
    deps = [
        ":github.com_docker_distribution",
        ":github.com_opencontainers_go-digest",
        ":github.com_opencontainers_image-spec",
        ":github.com_opencontainers_runc",
        ":github.com_pkg_errors",
        ":github.com_sirupsen_logrus",
        ":golang.org_x_net",
        ":golang.org_x_sync",
        ":golang.org_x_sys",
        ":google.golang.org_grpc",
    ],
)

go_get(
    name = "github.com_coreos_go-oidc",
    download_deps = [],
    install = [""],
    module = "github.com/coreos/go-oidc",
    sum = "h1:sdJrfw8akMnCuUlaZU3tE/uYXFgfqom8DBE9so9EBsM=",
    version = "v2.1.0+incompatible",
    deps = [
        ":github.com_pquerna_cachecontrol",
        ":golang.org_x_oauth2",
        ":gopkg.in_square_go-jose.v2",
    ],
)

go_get(
    name = "github.com_cyphar_filepath-securejoin",
    download_deps = [],
    install = [""],
    module = "github.com/cyphar/filepath-securejoin",
    sum = "h1:jCwT2GTP+PY5nBz3c/YL5PAIbusElVrPujOBSCj8xRg=",
    version = "v0.2.2",
    deps = [":github.com_pkg_errors"],
)

go_get(
    name = "github.com_davecgh_go-spew",
    download_deps = [],
    install = ["spew"],
    module = "github.com/davecgh/go-spew",
    sum = "h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=",
    version = "v1.1.1",
    deps = [],
)

go_get(
    name = "github.com_deislabs_oras",
    download_deps = [],
    install = [
        "pkg/auth",
        "pkg/auth/docker",
        "pkg/content",
        "pkg/context",
        "pkg/artifact",
        "pkg/oras",
    ],
    module = "github.com/deislabs/oras",
    sum = "h1:If674KraJVpujYR00rzdi0QAmW4BxzMJPVAZJKuhQ0c=",
    version = "v0.8.1",
    deps = [
        ":github.com_containerd_containerd",
        ":github.com_docker_cli",
        ":github.com_docker_docker",
        ":github.com_opencontainers_go-digest",
        ":github.com_opencontainers_image-spec",
        ":github.com_pkg_errors",
        ":github.com_sirupsen_logrus",
        ":golang.org_x_sync",
    ],
)

go_get(
    name = "github.com_dgrijalva_jwt-go",
    download_deps = [],
    install = [
        "",
        "request",
    ],
    module = "github.com/dgrijalva/jwt-go",
    sum = "h1:7qlOGliEKZXTDg6OTjfoBKDXWrumCAMpl/TFQ4/5kLM=",
    version = "v3.2.0+incompatible",
    deps = [],
)

go_get(
    name = "github.com_dimchansky_utfbom",
    download_deps = [],
    install = [""],
    module = "github.com/dimchansky/utfbom",
    sum = "h1:FcM3g+nofKgUteL8dm/UpdRXNC9KmADgTpLKsu0TRo4=",
    version = "v1.1.0",
    deps = [],
)

go_get(
    name = "github.com_docker_cli",
    download_deps = [],
    install = [
        "cli/config/types",
        "cli/config/credentials",
        "cli/config/configfile",
        "cli/config",
    ],
    module = "github.com/docker/cli",
    sum = "h1:FwssHbCDJD025h+BchanCwE1Q8fyMgqDr2mOQAWOLGw=",
    version = "v0.0.0-20200130152716-5d0cf8839492",
    deps = [
        ":github.com_docker_docker",
        ":github.com_docker_docker-credential-helpers",
        ":github.com_pkg_errors",
    ],
)

go_get(
    name = "github.com_docker_distribution",
    download_deps = [],
    install = [
        "digestset",
        "reference",
        "registry/api/errcode",
        "",
        "registry/api/v2",
        "registry/client/auth/challenge",
        "registry/client/transport",
        "metrics",
        "registry/storage/cache",
        "registry/storage/cache/memory",
        "registry/client",
        "registry/client/auth",
    ],
    module = "github.com/docker/distribution",
    sum = "h1:a5mlkVzth6W5A4fOsS3D2EO5BUmsJpcB+cRlLU7cSug=",
    version = "v2.7.1+incompatible",
    deps = [
        ":github.com_docker_go-metrics",
        ":github.com_gorilla_mux",
        ":github.com_opencontainers_go-digest",
        ":github.com_opencontainers_image-spec",
    ],
)

go_get(
    name = "github.com_docker_docker",
    download_deps = [],
    install = [
        "pkg/term",
        "pkg/homedir",
        "api/types/blkiodev",
        "api/types/mount",
        "api/types/strslice",
        "api/types/container",
        "api/types/versions",
        "api/types/filters",
        "errdefs",
        "api/types/network",
        "api/types/registry",
        "api/types/swarm/runtime",
        "api/types/swarm",
        "api/types",
        "pkg/ioutils",
        "pkg/jsonmessage",
        "pkg/stringid",
        "pkg/tarsum",
        "registry/resumable",
        "rootless",
        "registry",
    ],
    module = "github.com/docker/docker",
    sum = "h1:KXS1Jg+ddGcWA8e1N7cupxaHHZhit5rB9tfDU+mfjyY=",
    version = "v1.4.2-0.20200203170920-46ec8731fbce",
    deps = [
        ":github.com_containerd_containerd",
        ":github.com_docker_distribution",
        ":github.com_docker_go-connections",
        ":github.com_docker_go-units",
        ":github.com_gogo_protobuf",
        ":github.com_morikuni_aec",
        ":github.com_opencontainers_image-spec",
        ":github.com_pkg_errors",
        ":github.com_sirupsen_logrus",
        ":golang.org_x_sys",
        ":google.golang.org_grpc",
    ],
)

go_get(
    name = "github.com_docker_docker-credential-helpers",
    download_deps = [],
    install = [
        "credentials",
        "client",
    ],
    module = "github.com/docker/docker-credential-helpers",
    sum = "h1:zI2p9+1NQYdnG6sMU26EX4aVGlqbInSQxQXLvzJ4RPQ=",
    version = "v0.6.3",
    deps = [],
)

go_get(
    name = "github.com_docker_go-connections",
    download_deps = [],
    install = [
        "nat",
        "tlsconfig",
    ],
    module = "github.com/docker/go-connections",
    sum = "h1:El9xVISelRB7BuFusrZozjnkIM5YnzCViNKohAFqRJQ=",
    version = "v0.4.0",
    deps = [":github.com_pkg_errors"],
)

go_get(
    name = "github.com_docker_go-metrics",
    download_deps = [],
    install = [""],
    module = "github.com/docker/go-metrics",
    sum = "h1:yWHOI+vFjEsAakUTSrtqc/SAHrhSkmn48pqjidZX3QA=",
    version = "v0.0.0-20180209012529-399ea8c73916",
    deps = [":github.com_prometheus_client_golang"],
)

go_get(
    name = "github.com_docker_go-units",
    download_deps = [],
    install = [""],
    module = "github.com/docker/go-units",
    sum = "h1:3uh0PgVws3nIA0Q+MwDC8yjEPf9zjRfZZWXZYDct3Tw=",
    version = "v0.4.0",
    deps = [],
)

go_get(
    name = "github.com_docker_spdystream",
    download_deps = [],
    install = [
        "spdy",
        "",
    ],
    module = "github.com/docker/spdystream",
    sum = "h1:cenwrSVm+Z7QLSV/BsnenAOcDXdX4cMv4wP0B/5QbPg=",
    version = "v0.0.0-20160310174837-449fdfce4d96",
    deps = [],
)

go_get(
    name = "github.com_emicklei_go-restful",
    download_deps = [],
    install = [
        "log",
        "",
    ],
    module = "github.com/emicklei/go-restful",
    sum = "h1:spTtZBk5DYEvbxMVutUuTyh1Ao2r4iyvLdACqsl/Ljk=",
    version = "v2.9.5+incompatible",
    deps = [],
)

go_get(
    name = "github.com_evanphx_json-patch",
    download_deps = [],
    install = [""],
    module = "github.com/evanphx/json-patch",
    sum = "h1:ouOWdg56aJriqS0huScTkVXPC5IcNrDCXZ6OoTAWu7M=",
    version = "v4.5.0+incompatible",
    deps = [":github.com_pkg_errors"],
)

go_get(
    name = "github.com_exponent-io_jsonpath",
    download_deps = [],
    install = [""],
    module = "github.com/exponent-io/jsonpath",
    sum = "h1:105gxyaGwCFad8crR9dcMQWvV9Hvulu6hwUh4tWPJnM=",
    version = "v0.0.0-20151013193312-d6023ce2651d",
    deps = [],
)

go_get(
    name = "github.com_facebookgo_clock",
    download_deps = [],
    install = [""],
    module = "github.com/facebookgo/clock",
    sum = "h1:yDWHCSQ40h88yih2JAcL6Ls/kVkSE8GFACTGVnMPruw=",
    version = "v0.0.0-20150410010913-600d898af40a",
    deps = [],
)

go_get(
    name = "github.com_fatih_color",
    download_deps = [],
    install = [""],
    module = "github.com/fatih/color",
    sum = "h1:DkWD4oS2D8LGGgTQ6IvwJJXSL5Vp2ffcQg58nFV38Ys=",
    version = "v1.7.0",
    deps = [
        ":github.com_mattn_go-colorable",
        ":github.com_mattn_go-isatty",
    ],
)

go_get(
    name = "github.com_fsnotify_fsnotify",
    download_deps = [],
    install = [""],
    module = "github.com/fsnotify/fsnotify",
    sum = "h1:IXs+QLmnXW2CcXuY+8Mzv/fWEsPGWxqefPtCP5CnV9I=",
    version = "v1.4.7",
    deps = [":golang.org_x_sys"],
)

go_get(
    name = "github.com_ghodss_yaml",
    download_deps = [],
    install = [""],
    module = "github.com/ghodss/yaml",
    sum = "h1:wQHKEahhL6wmXdzwWG11gIVCkOv05bNOh+Rxn0yngAk=",
    version = "v1.0.0",
    deps = [":gopkg.in_yaml.v2"],
)

go_get(
    name = "github.com_gin-contrib_cors",
    download_deps = [],
    install = [""],
    module = "github.com/gin-contrib/cors",
    sum = "h1:PolezCc89peu+NgkIWt9OB01Kbzt6IP0J/JvkG6xxlg=",
    version = "v1.3.0",
    deps = [":github.com_gin-gonic_gin"],
)

go_get(
    name = "github.com_gin-contrib_sse",
    download_deps = [],
    install = [""],
    module = "github.com/gin-contrib/sse",
    sum = "h1:Y/yl/+YNO8GZSjAhjMsSuLt29uWRFHdHYUb5lYOV9qE=",
    version = "v0.1.0",
    deps = [],
)

go_get(
    name = "github.com_gin-gonic_gin",
    download_deps = [],
    install = [
        "internal/json",
        "binding",
        "render",
        "",
    ],
    module = "github.com/gin-gonic/gin",
    sum = "h1:fi+bqFAx/oLK54somfCtEZs9HeH1LHVoEPUgARpTqyc=",
    version = "v1.5.0",
    deps = [
        ":github.com_gin-contrib_sse",
        ":github.com_golang_protobuf",
        ":github.com_mattn_go-isatty",
        ":github.com_ugorji_go_codec",
        ":gopkg.in_go-playground_validator.v9",
        ":gopkg.in_yaml.v2",
    ],
)

go_get(
    name = "github.com_go-kit_kit",
    download_deps = [],
    install = [
        "endpoint",
        "log",
        "transport",
        "transport/http",
        "sd",
        "sd/lb",
        "transport/grpc",
        "tracing/opencensus",
    ],
    module = "github.com/go-kit/kit",
    sum = "h1:dXFJfIHVvUcpSgDOV+Ne6t7jXri8Tfv2uOLHUZ2XNuo=",
    version = "v0.10.0",
    deps = [
        ":github.com_go-logfmt_logfmt",
        ":go.opencensus.io",
        ":google.golang.org_grpc",
    ],
)

go_get(
    name = "github.com_go-logfmt_logfmt",
    download_deps = [],
    install = [""],
    module = "github.com/go-logfmt/logfmt",
    sum = "h1:TrB8swr/68K7m9CcGut2g3UOihhbcbiMAYiuTXdEih4=",
    version = "v0.5.0",
    deps = [],
)

go_get(
    name = "github.com_go-logr_logr",
    download_deps = [],
    install = [""],
    module = "github.com/go-logr/logr",
    sum = "h1:M1Tv3VzNlEHg6uyACnRdtrploV2P7wZqH8BoQMtz0cg=",
    version = "v0.1.0",
    deps = [],
)

go_get(
    name = "github.com_go-openapi_jsonpointer",
    download_deps = [],
    install = [""],
    module = "github.com/go-openapi/jsonpointer",
    sum = "h1:gihV7YNZK1iK6Tgwwsxo2rJbD1GTbdm72325Bq8FI3w=",
    version = "v0.19.3",
    deps = [":github.com_go-openapi_swag"],
)

go_get(
    name = "github.com_go-openapi_jsonreference",
    download_deps = [],
    install = [""],
    module = "github.com/go-openapi/jsonreference",
    sum = "h1:5cxNfTy0UVC3X8JL5ymxzyoUZmo8iZb+jeTWn7tUa8o=",
    version = "v0.19.3",
    deps = [
        ":github.com_PuerkitoBio_purell",
        ":github.com_go-openapi_jsonpointer",
    ],
)

go_get(
    name = "github.com_go-openapi_spec",
    download_deps = [],
    install = [""],
    module = "github.com/go-openapi/spec",
    sum = "h1:0XRyw8kguri6Yw4SxhsQA/atC88yqrk0+G4YhI2wabc=",
    version = "v0.19.3",
    deps = [
        ":github.com_go-openapi_jsonpointer",
        ":github.com_go-openapi_jsonreference",
        ":github.com_go-openapi_swag",
    ],
)

go_get(
    name = "github.com_go-openapi_swag",
    download_deps = [],
    install = [""],
    module = "github.com/go-openapi/swag",
    sum = "h1:lTz6Ys4CmqqCQmZPBlbQENR1/GucA2bzYTE12Pw4tFY=",
    version = "v0.19.5",
    deps = [
        ":github.com_mailru_easyjson",
        ":gopkg.in_yaml.v2",
    ],
)

go_get(
    name = "github.com_go-playground_locales",
    download_deps = [],
    install = [
        "currency",
        "",
    ],
    module = "github.com/go-playground/locales",
    sum = "h1:2FITxuFt/xuCNP1Acdhv62OzaCiviiE4kotfhkmOqEc=",
    version = "v0.12.1",
    deps = [],
)

go_get(
    name = "github.com_go-playground_universal-translator",
    download_deps = [],
    install = [""],
    module = "github.com/go-playground/universal-translator",
    sum = "h1:X++omBR/4cE2MNg91AoC3rmGrCjJ8eAeUP/K/EKx4DM=",
    version = "v0.16.0",
    deps = [":github.com_go-playground_locales"],
)

go_get(
    name = "github.com_go-sql-driver_mysql",
    download_deps = [],
    install = [""],
    module = "github.com/go-sql-driver/mysql",
    sum = "h1:ozyZYNQW3x3HtqT1jira07DN2PArx2v7/mN66gGcHOs=",
    version = "v1.5.0",
    deps = [],
)

go_get(
    name = "github.com_gobwas_glob",
    download_deps = [],
    install = [
        "util/runes",
        "util/strings",
        "match",
        "syntax/lexer",
        "syntax/ast",
        "compiler",
        "syntax",
        "",
    ],
    module = "github.com/gobwas/glob",
    sum = "h1:A4xDbljILXROh+kObIiy5kIaPYD8e96x1tgBhUI5J+Y=",
    version = "v0.2.3",
    deps = [],
)

go_get(
    name = "github.com_gofrs_flock",
    download_deps = [],
    install = [""],
    module = "github.com/gofrs/flock",
    sum = "h1:DP+LD/t0njgoPBvT5MJLeliUIVQR03hiKR6vezdwHlc=",
    version = "v0.7.1",
    deps = [],
)

go_get(
    name = "github.com_gofrs_uuid",
    download_deps = [],
    install = [""],
    module = "github.com/gofrs/uuid",
    sum = "h1:y12jRkkFxsd7GpqdSZ+/KCs/fJbqpEXSGd4+jfEaewE=",
    version = "v3.2.0+incompatible",
    deps = [],
)

go_get(
    name = "github.com_gogo_protobuf",
    download_deps = [],
    install = [
        "proto",
        "sortkeys",
    ],
    module = "github.com/gogo/protobuf",
    sum = "h1:DqDEcV5aeaTmdFBePNpYsp3FlcVH/2ISVVM9Qf8PSls=",
    version = "v1.3.1",
    deps = [],
)

go_get(
    name = "github.com_golang_groupcache",
    download_deps = [],
    install = ["lru"],
    module = "github.com/golang/groupcache",
    sum = "h1:5ZkaAPbicIKTF2I64qf5Fh8Aa83Q/dnOafMYV0OMwjA=",
    version = "v0.0.0-20191227052852-215e87163ea7",
    deps = [],
)

go_get(
    name = "github.com_golang_mock",
    download_deps = [],
    install = ["gomock"],
    module = "github.com/golang/mock",
    sum = "h1:GV+pQPG/EUUbkh47niozDcADz6go/dUwhVzdUQHIVRw=",
    version = "v1.4.3",
    deps = [],
)

go_get(
    name = "github.com_golang_protobuf",
    download_deps = [],
    install = [
        "proto",
        "ptypes/any",
        "ptypes/duration",
        "ptypes/timestamp",
        "ptypes",
        "protoc-gen-go/descriptor",
    ],
    module = "github.com/golang/protobuf",
    sum = "h1:F768QJ1E9tib+q5Sc8MkdJi1RxLTbRcTf8LJV56aRls=",
    version = "v1.3.5",
    deps = [],
)

go_get(
    name = "github.com_golang_snappy",
    download_deps = [],
    install = [""],
    module = "github.com/golang/snappy",
    sum = "h1:Qgr9rKW7uDUkrbSmQeiDsGa8SjGyCOGtuasMWwvp2P4=",
    version = "v0.0.1",
    deps = [],
)

go_get(
    name = "github.com_google_btree",
    download_deps = [],
    install = [""],
    module = "github.com/google/btree",
    sum = "h1:0udJVsspx3VBr5FwtLhQQtuAsVc79tTq0ocGIPAU6qo=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "github.com_google_go-cmp",
    download_deps = [],
    install = [
        "cmp/internal/diff",
        "cmp/internal/flags",
        "cmp/internal/function",
        "cmp/internal/value",
        "cmp",
    ],
    module = "github.com/google/go-cmp",
    sum = "h1:xsAVV57WRhGj6kEIi8ReJzQlHHqcBYCElAvkovg3B/4=",
    version = "v0.4.0",
    deps = [],
)

go_get(
    name = "github.com_google_gofuzz",
    download_deps = [],
    install = [""],
    module = "github.com/google/gofuzz",
    sum = "h1:A8PeW59pxE9IoFRqBp37U+mSNaQoZ46F1f0f863XSXw=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "github.com_google_uuid",
    download_deps = [],
    install = [""],
    module = "github.com/google/uuid",
    sum = "h1:Gkbcsh/GbpXz7lPftLA3P6TYMwjCLYm83jiFQZF/3gY=",
    version = "v1.1.1",
    deps = [],
)

go_get(
    name = "github.com_googleapis_gax-go_v2",
    download_deps = [],
    install = [""],
    module = "github.com/googleapis/gax-go/v2",
    sum = "h1:sjZBwGj9Jlw33ImPtvFviGYvseOtDM7hkSKB7+Tv3SM=",
    version = "v2.0.5",
    deps = [":google.golang.org_grpc"],
)

go_get(
    name = "github.com_googleapis_gnostic",
    download_deps = [],
    install = [
        "extensions",
        "compiler",
        "OpenAPIv2",
    ],
    module = "github.com/googleapis/gnostic",
    sum = "h1:WeAefnSUHlBb0iJKwxFDZdbfGwkd7xRNuV+IpXMJhYk=",
    version = "v0.3.1",
    deps = [
        ":github.com_golang_protobuf",
        ":gopkg.in_yaml.v2",
    ],
)

go_get(
    name = "github.com_goph_emperror",
    download_deps = [],
    install = [""],
    module = "github.com/goph/emperror",
    sum = "h1:yLapQcmEsO0ipe9p5TaN22djm3OFV/TfM/fcYP0/J18=",
    version = "v0.17.2",
    deps = [":github.com_pkg_errors"],
)

go_get(
    name = "github.com_gorilla_context",
    download_deps = [],
    install = [""],
    module = "github.com/gorilla/context",
    sum = "h1:AWwleXJkX/nhcU9bZSnZoi3h/qGYqQAGhq6zZe/aQW8=",
    version = "v1.1.1",
    deps = [],
)

go_get(
    name = "github.com_gorilla_mux",
    download_deps = [],
    install = [""],
    module = "github.com/gorilla/mux",
    sum = "h1:gnP5JzjVOuiZD07fKKToCAOjS0yOpj/qPETTXCCS6hw=",
    version = "v1.7.3",
    deps = [],
)

go_get(
    name = "github.com_gorilla_securecookie",
    download_deps = [],
    install = [""],
    module = "github.com/gorilla/securecookie",
    sum = "h1:miw7JPhV+b/lAHSXz4qd/nN9jRiAFV5FwjeKyCS8BvQ=",
    version = "v1.1.1",
    deps = [],
)

go_get(
    name = "github.com_gorilla_sessions",
    download_deps = [],
    install = [""],
    module = "github.com/gorilla/sessions",
    sum = "h1:S7P+1Hm5V/AT9cjEcUD5uDaQSX0OE577aCXgoaKpYbQ=",
    version = "v1.2.0",
    deps = [":github.com_gorilla_securecookie"],
)

go_get(
    name = "github.com_gosimple_slug",
    download_deps = [],
    install = [""],
    module = "github.com/gosimple/slug",
    sum = "h1:r5vDcYrFz9BmfIAMC829un9hq7hKM4cHUrsv36LbEqs=",
    version = "v1.9.0",
    deps = [":github.com_rainycape_unidecode"],
)

go_get(
    name = "github.com_gosuri_uitable",
    download_deps = [],
    install = [
        "util/strutil",
        "util/wordwrap",
        "",
    ],
    module = "github.com/gosuri/uitable",
    sum = "h1:IG2xLKRvErL3uhY6e1BylFzG+aJiwQviDDTfOKeKTpY=",
    version = "v0.0.4",
    deps = [
        ":github.com_fatih_color",
        ":github.com_mattn_go-runewidth",
    ],
)

go_get(
    name = "github.com_gregjones_httpcache",
    download_deps = [],
    install = [
        "",
        "diskcache",
    ],
    module = "github.com/gregjones/httpcache",
    sum = "h1:pdN6V1QBWetyv/0+wjACpqVH+eVULgEjkurDLq3goeM=",
    version = "v0.0.0-20180305231024-9cad4c3443a7",
    deps = [":github.com_peterbourgon_diskv"],
)

go_get(
    name = "github.com_hashicorp_errwrap",
    download_deps = [],
    install = [""],
    module = "github.com/hashicorp/errwrap",
    sum = "h1:hLrqtEDnRye3+sgx6z4qVLNuviH3MR5aQ0ykNJa/UYA=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "github.com_hashicorp_go-cleanhttp",
    download_deps = [],
    install = [""],
    module = "github.com/hashicorp/go-cleanhttp",
    sum = "h1:dH3aiDG9Jvb5r5+bYHsikaOUIpcM0xvgMXVoDkXMzJM=",
    version = "v0.5.1",
    deps = [],
)

go_get(
    name = "github.com_hashicorp_go-multierror",
    download_deps = [],
    install = [""],
    module = "github.com/hashicorp/go-multierror",
    sum = "h1:iVjPR7a6H0tWELX5NxNe7bYopibicUzc7uPribsnS6o=",
    version = "v1.0.0",
    deps = [":github.com_hashicorp_errwrap"],
)

go_get(
    name = "github.com_hashicorp_go-retryablehttp",
    download_deps = [],
    install = [""],
    module = "github.com/hashicorp/go-retryablehttp",
    sum = "h1:1BZvpawXoJCWX6pNtow9+rpEj+3itIlutiqnntI6jOE=",
    version = "v0.5.4",
    deps = [":github.com_hashicorp_go-cleanhttp"],
)

go_get(
    name = "github.com_hashicorp_go-rootcerts",
    download_deps = [],
    install = [""],
    module = "github.com/hashicorp/go-rootcerts",
    sum = "h1:DMo4fmknnz0E0evoNYnV48RjWndOsmd6OW+09R3cEP8=",
    version = "v1.0.1",
    deps = [":github.com_mitchellh_go-homedir"],
)

go_get(
    name = "github.com_hashicorp_go-sockaddr",
    download_deps = [],
    install = [""],
    module = "github.com/hashicorp/go-sockaddr",
    sum = "h1:ztczhD1jLxIRjVejw8gFomI1BQZOe2WoVOu0SyteCQc=",
    version = "v1.0.2",
    deps = [],
)

go_get(
    name = "github.com_hashicorp_golang-lru",
    download_deps = [],
    install = [
        "simplelru",
        "",
    ],
    module = "github.com/hashicorp/golang-lru",
    sum = "h1:0hERBMJE1eitiLkihrMvRVBYAkpHzc/J3QdDN+dAcgU=",
    version = "v0.5.1",
    deps = [],
)

go_get(
    name = "github.com_hashicorp_hcl",
    download_deps = [],
    install = [
        "hcl/strconv",
        "hcl/token",
        "hcl/ast",
        "hcl/scanner",
        "hcl/parser",
        "json/token",
        "json/scanner",
        "json/parser",
        "",
        "hcl/printer",
    ],
    module = "github.com/hashicorp/hcl",
    sum = "h1:0Anlzjpi4vEasTeNFn2mLJgTSwt0+6sfsiTG8qcWGx4=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "github.com_hashicorp_vault_api",
    download_deps = [],
    install = [""],
    module = "github.com/hashicorp/vault/api",
    sum = "h1:j08Or/wryXT4AcHj1oCbMd7IijXcKzYUGw59LGu9onU=",
    version = "v1.0.4",
    deps = [
        ":github.com_hashicorp_errwrap",
        ":github.com_hashicorp_go-cleanhttp",
        ":github.com_hashicorp_go-multierror",
        ":github.com_hashicorp_go-retryablehttp",
        ":github.com_hashicorp_go-rootcerts",
        ":github.com_hashicorp_hcl",
        ":github.com_hashicorp_vault_sdk",
        ":github.com_mitchellh_mapstructure",
        ":golang.org_x_net",
        ":golang.org_x_time",
        ":gopkg.in_square_go-jose.v2",
    ],
)

go_get(
    name = "github.com_hashicorp_vault_sdk",
    download_deps = [],
    install = [
        "helper/consts",
        "helper/hclutil",
        "helper/compressutil",
        "helper/jsonutil",
        "helper/strutil",
        "helper/parseutil",
    ],
    module = "github.com/hashicorp/vault/sdk",
    sum = "h1:mOEPeOhT7jl0J4AMl1E705+BcmeRs1VmKNb9F0sMLy8=",
    version = "v0.1.13",
    deps = [
        ":github.com_golang_snappy",
        ":github.com_hashicorp_errwrap",
        ":github.com_hashicorp_go-multierror",
        ":github.com_hashicorp_go-sockaddr",
        ":github.com_hashicorp_hcl",
        ":github.com_mitchellh_mapstructure",
        ":github.com_pierrec_lz4",
        ":github.com_ryanuber_go-glob",
    ],
)

go_get(
    name = "github.com_heptio_ark",
    download_deps = [],
    install = [
        "pkg/apis/ark/v1",
        "pkg/generated/clientset/versioned/scheme",
        "pkg/cloudprovider",
    ],
    module = "github.com/heptio/ark",
    sum = "h1:LcyHwxub/geleuL6xEUk1xOBa6Fa8uT+/a7h2HEYBmc=",
    version = "v0.9.3",
    deps = [
        ":github.com_pkg_errors",
        ":github.com_sirupsen_logrus",
        ":k8s.io_api",
        ":k8s.io_apimachinery",
    ],
)

go_get(
    name = "github.com_huandu_xstrings",
    download_deps = [],
    install = [""],
    module = "github.com/huandu/xstrings",
    sum = "h1:yPeWdRnmynF7p+lLYz0H2tthW9lqhMJrQV/U7yy4wX0=",
    version = "v1.2.0",
    deps = [],
)

go_get(
    name = "github.com_iancoleman_orderedmap",
    download_deps = [],
    install = [""],
    module = "github.com/iancoleman/orderedmap",
    sum = "h1:i462o439ZjprVSFSZLZxcsoAe592sZB1rci2Z8j4wdk=",
    version = "v0.0.0-20190318233801-ac98e3ecb4b0",
    deps = [],
)

go_get(
    name = "github.com_imdario_mergo",
    download_deps = [],
    install = [""],
    module = "github.com/imdario/mergo",
    sum = "h1:Y+UAYTZ7gDEuOfhxKWy+dvb5dRQ6rJjFSdX2HZY1/gI=",
    version = "v0.3.7",
    deps = [],
)

go_get(
    name = "github.com_jinzhu_copier",
    download_deps = [],
    install = [""],
    module = "github.com/jinzhu/copier",
    sum = "h1:zPPuIq2jAWWPTrGt70eK/BSch+gFAGrNzecsoENgu2o=",
    version = "v0.0.0-20190924061706-b57f9002281a",
    deps = [],
)

go_get(
    name = "github.com_jinzhu_gorm",
    download_deps = [],
    install = [
        "",
        "dialects/mysql",
        "dialects/postgres",
        "dialects/sqlite",
    ],
    module = "github.com/jinzhu/gorm",
    replace = "github.com/jinzhu/gorm",
    sum = "h1:lDSDtsCt5AGGSKTs8AHlSDbbgif4G4+CKJ8ETBDVHTA=",
    version = "v1.9.1",
    deps = [
        ":github.com_go-sql-driver_mysql",
        ":github.com_jinzhu_inflection",
        ":github.com_lib_pq",
        ":github.com_mattn_go-sqlite3",
    ],
)

go_get(
    name = "github.com_jinzhu_inflection",
    download_deps = [],
    install = [""],
    module = "github.com/jinzhu/inflection",
    sum = "h1:K317FqzuhWc8YvSVlFMCCUb36O/S9MCKRDI7QkRKD/E=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "github.com_jinzhu_now",
    download_deps = [],
    install = [""],
    module = "github.com/jinzhu/now",
    sum = "h1:g39TucaRWyV3dwDO++eEc6qf8TVIQ/Da48WmqjZ3i7E=",
    version = "v1.1.1",
    deps = [],
)

go_get(
    name = "github.com_jmespath_go-jmespath",
    download_deps = [],
    install = [""],
    module = "github.com/jmespath/go-jmespath",
    sum = "h1:pmfjZENx5imkbgOkpRUYLnmbU7UEFbjtDA2hxJ1ichM=",
    version = "v0.0.0-20180206201540-c2b33e8439af",
    deps = [],
)

go_get(
    name = "github.com_jmoiron_sqlx",
    download_deps = [],
    install = [
        "reflectx",
        "",
    ],
    module = "github.com/jmoiron/sqlx",
    sum = "h1:41Ip0zITnmWNR/vHV+S4m+VoUivnWY5E4OJfLZjCJMA=",
    version = "v1.2.0",
    deps = [],
)

go_get(
    name = "github.com_jonboulle_clockwork",
    download_deps = [],
    install = [""],
    module = "github.com/jonboulle/clockwork",
    sum = "h1:J2SLSdy7HgElq8ekSl2Mxh6vrRNFxqbXGenYH2I02Vs=",
    version = "v0.2.0",
    deps = [],
)

go_get(
    name = "github.com_json-iterator_go",
    download_deps = [],
    install = [""],
    module = "github.com/json-iterator/go",
    sum = "h1:9yzud/Ht36ygwatGx56VwCZtlI/2AD15T1X2sjSuGns=",
    version = "v1.1.9",
    deps = [
        ":github.com_modern-go_concurrent",
        ":github.com_modern-go_reflect2",
    ],
)

go_get(
    name = "github.com_leodido_go-urn",
    download_deps = [],
    install = [""],
    module = "github.com/leodido/go-urn",
    sum = "h1:Sm1gr51B1kKyfD2BlRcLSiEkffoG96g6TPv6eRoEiB8=",
    version = "v1.1.0",
    deps = [],
)

go_get(
    name = "github.com_lestrrat-go_backoff",
    download_deps = [],
    install = [""],
    module = "github.com/lestrrat-go/backoff",
    sum = "h1:nR+UgAhdhwfw2i+xznuHRlj81oMYa7u3lXun0xcsXUU=",
    version = "v1.0.0",
    deps = [":github.com_pkg_errors"],
)

go_get(
    name = "github.com_lib_pq",
    download_deps = [],
    install = [
        "oid",
        "scram",
        "",
        "hstore",
    ],
    module = "github.com/lib/pq",
    sum = "h1:/qkRGz8zljWiDcFvgpwUpwIAPu3r07TDvs3Rws+o/pU=",
    version = "v1.3.0",
    deps = [],
)

go_get(
    name = "github.com_liggitt_tabwriter",
    download_deps = [],
    install = [""],
    module = "github.com/liggitt/tabwriter",
    sum = "h1:9TO3cAIGXtEhnIaL+V+BEER86oLrvS+kWobKpbJuye0=",
    version = "v0.0.0-20181228230101-89fcab3d43de",
    deps = [],
)

go_get(
    name = "github.com_lithammer_shortuuid_v3",
    download_deps = [],
    install = [""],
    module = "github.com/lithammer/shortuuid/v3",
    sum = "h1:uj4xhotfY92Y1Oa6n6HUiFn87CdoEHYUlTy0+IgbLrs=",
    version = "v3.0.4",
    deps = [":github.com_google_uuid"],
)

go_get(
    name = "github.com_magiconair_properties",
    download_deps = [],
    install = [""],
    module = "github.com/magiconair/properties",
    sum = "h1:ZC2Vc7/ZFkGmsVC9KvOjumD+G5lXy2RtTKyzRKO2BQ4=",
    version = "v1.8.1",
    deps = [],
)

go_get(
    name = "github.com_mailru_easyjson",
    download_deps = [],
    install = [
        "jlexer",
        "buffer",
        "jwriter",
    ],
    module = "github.com/mailru/easyjson",
    sum = "h1:aizVhC/NAAcKWb+5QsU1iNOZb4Yws5UO2I+aIprQITM=",
    version = "v0.7.0",
    deps = [],
)

go_get(
    name = "github.com_mattn_go-colorable",
    download_deps = [],
    install = [""],
    module = "github.com/mattn/go-colorable",
    sum = "h1:UVL0vNpWh04HeJXV0KLcaT7r06gOH2l4OW6ddYRUIY4=",
    version = "v0.0.9",
    deps = [":github.com_mattn_go-isatty"],
)

go_get(
    name = "github.com_mattn_go-ieproxy",
    download_deps = [],
    install = [""],
    module = "github.com/mattn/go-ieproxy",
    sum = "h1:oNAwILwmgWKFpuU+dXvI6dl9jG2mAWAZLX3r9s0PPiw=",
    version = "v0.0.0-20190702010315-6dee0af9227d",
    deps = [],
)

go_get(
    name = "github.com_mattn_go-isatty",
    download_deps = [],
    install = [""],
    module = "github.com/mattn/go-isatty",
    sum = "h1:d5US/mDsogSGW37IV293h//ZFaeajb69h+EHFsv2xGg=",
    version = "v0.0.9",
    deps = [],
)

go_get(
    name = "github.com_mattn_go-runewidth",
    download_deps = [],
    install = [""],
    module = "github.com/mattn/go-runewidth",
    sum = "h1:2BvfKmzob6Bmd4YsL0zygOqfdFnK7GR4QL06Do4/p7Y=",
    version = "v0.0.4",
    deps = [],
)

go_get(
    name = "github.com_mattn_go-sqlite3",
    download_deps = [],
    install = [""],
    module = "github.com/mattn/go-sqlite3",
    sum = "h1:gXHsfypPkaMZrKbD5209QV9jbUTJKjyR5WD3HYQSd+U=",
    version = "v2.0.3+incompatible",
    deps = [],
)

go_get(
    name = "github.com_matttproud_golang_protobuf_extensions",
    download_deps = [],
    install = ["pbutil"],
    module = "github.com/matttproud/golang_protobuf_extensions",
    sum = "h1:I0XW9+e1XWDxdcEniV4rQAIOPUGDq67JSCiRCgGCZLI=",
    version = "v1.0.2-0.20181231171920-c182affec369",
    deps = [":github.com_golang_protobuf"],
)

go_get(
    name = "github.com_microcosm-cc_bluemonday",
    download_deps = [],
    install = [""],
    module = "github.com/microcosm-cc/bluemonday",
    sum = "h1:5lPfLTTAvAbtS0VqT+94yOtFnGfUWYyx0+iToC3Os3s=",
    version = "v1.0.2",
    deps = [":golang.org_x_net"],
)

go_get(
    name = "github.com_mitchellh_copystructure",
    download_deps = [],
    install = [""],
    module = "github.com/mitchellh/copystructure",
    sum = "h1:Laisrj+bAB6b/yJwB5Bt3ITZhGJdqmxquMKeZ+mmkFQ=",
    version = "v1.0.0",
    deps = [":github.com_mitchellh_reflectwalk"],
)

go_get(
    name = "github.com_mitchellh_go-homedir",
    download_deps = [],
    install = [""],
    module = "github.com/mitchellh/go-homedir",
    sum = "h1:lukF9ziXFxDFPkA1vsr5zpc1XuPDn/wFntq5mG+4E0Y=",
    version = "v1.1.0",
    deps = [],
)

go_get(
    name = "github.com_mitchellh_go-wordwrap",
    download_deps = [],
    install = [""],
    module = "github.com/mitchellh/go-wordwrap",
    sum = "h1:6GlHJ/LTGMrIJbwgdqdl2eEH8o+Exx/0m8ir9Gns0u4=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "github.com_mitchellh_mapstructure",
    download_deps = [],
    install = [""],
    module = "github.com/mitchellh/mapstructure",
    sum = "h1:fmNYVwqnSfB9mZU6OS2O6GsXM+wcskZDuKQzvN1EDeE=",
    version = "v1.1.2",
    deps = [],
)

go_get(
    name = "github.com_mitchellh_reflectwalk",
    download_deps = [],
    install = [""],
    module = "github.com/mitchellh/reflectwalk",
    sum = "h1:9D+8oIskB4VJBN5SFlmc27fSlIBZaov1Wpk/IfikLNY=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "github.com_modern-go_concurrent",
    download_deps = [],
    install = [""],
    module = "github.com/modern-go/concurrent",
    sum = "h1:TRLaZ9cD/w8PVh93nsPXa1VrQ6jlwL5oN8l14QlcNfg=",
    version = "v0.0.0-20180306012644-bacd9c7ef1dd",
    deps = [],
)

go_get(
    name = "github.com_modern-go_reflect2",
    download_deps = [],
    install = [""],
    module = "github.com/modern-go/reflect2",
    sum = "h1:9f412s+6RmYXLWZSEzVVgPGK7C2PphHj5RJrvfx9AWI=",
    version = "v1.0.1",
    deps = [":github.com_modern-go_concurrent"],
)

go_get(
    name = "github.com_moogar0880_problems",
    download_deps = [],
    install = [""],
    module = "github.com/moogar0880/problems",
    sum = "h1:bktLhq8NDG/czU2ZziYNigBFksx13RaYe5AVdNmHDT4=",
    version = "v0.1.1",
    deps = [],
)

go_get(
    name = "github.com_morikuni_aec",
    download_deps = [],
    install = [""],
    module = "github.com/morikuni/aec",
    sum = "h1:nP9CBfwrvYnBRgY6qfDQkygYDmYwOilePFkwzv4dU8A=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "github.com_mxk_go-flowrate",
    download_deps = [],
    install = ["flowrate"],
    module = "github.com/mxk/go-flowrate",
    sum = "h1:y5//uYreIhSUg3J1GEMiLbxo1LJaP8RfCpH6pymGZus=",
    version = "v0.0.0-20140419014527-cca7078d478f",
    deps = [],
)

go_get(
    name = "github.com_oklog_run",
    download_deps = [],
    install = [""],
    module = "github.com/oklog/run",
    sum = "h1:GEenZ1cK0+q0+wsJew9qUg/DyD8k3JzYsZAi5gYi2mA=",
    version = "v1.1.0",
    deps = [],
)

go_get(
    name = "github.com_oklog_ulid",
    download_deps = [],
    install = [""],
    module = "github.com/oklog/ulid",
    sum = "h1:EGfNDEx6MqHz8B3uNV6QAib1UR2Lm97sHi3ocA6ESJ4=",
    version = "v1.3.1",
    deps = [],
)

go_get(
    name = "github.com_onsi_gomega",
    download_deps = [],
    install = [
        "format",
        "gbytes",
        "types",
        "internal/assertion",
        "internal/oraclematcher",
        "internal/asyncassertion",
        "internal/testingtsupport",
        "matchers/support/goraph/node",
        "matchers/support/goraph/edge",
        "matchers/support/goraph/util",
        "matchers/support/goraph/bipartitegraph",
        "matchers",
        "",
        "gexec",
    ],
    module = "github.com/onsi/gomega",
    sum = "h1:R1uwffexN6Pr340GtYRIdZmAiN4J+iw6WG4wog1DUXg=",
    version = "v1.9.0",
    deps = [
        ":golang.org_x_net",
        ":golang.org_x_xerrors",
        ":gopkg.in_yaml.v2",
    ],
)

go_get(
    name = "github.com_opencontainers_go-digest",
    download_deps = [],
    install = [""],
    module = "github.com/opencontainers/go-digest",
    sum = "h1:WzifXhOVOEOuFYOJAW6aQqW0TooG2iki3E3Ii+WN7gQ=",
    version = "v1.0.0-rc1",
    deps = [],
)

go_get(
    name = "github.com_opencontainers_image-spec",
    download_deps = [],
    install = [
        "specs-go",
        "specs-go/v1",
    ],
    module = "github.com/opencontainers/image-spec",
    sum = "h1:JMemWkRwHx4Zj+fVxWoMCFm/8sYGGrUVojFA6h/TRcI=",
    version = "v1.0.1",
    deps = [":github.com_opencontainers_go-digest"],
)

go_get(
    name = "github.com_opencontainers_runc",
    download_deps = [],
    install = [
        "libcontainer/user",
        "libcontainer/system",
    ],
    module = "github.com/opencontainers/runc",
    sum = "h1:/k06BMULKF5hidyoZymkoDCzdJzltZpz/UU4LguQVtc=",
    version = "v1.0.0-rc9",
    deps = [":golang.org_x_sys"],
)

go_get(
    name = "github.com_opentracing_opentracing-go",
    download_deps = [],
    install = [
        "log",
        "",
        "ext",
    ],
    module = "github.com/opentracing/opentracing-go",
    sum = "h1:pWlfV3Bxv7k65HYwkikxat0+s3pV4bsqf19k25Ur8rU=",
    version = "v1.1.0",
    deps = [],
)

go_get(
    name = "github.com_oracle_oci-go-sdk",
    download_deps = [],
    install = [
        "common",
        "containerengine",
        "core",
        "identity",
        "objectstorage",
    ],
    module = "github.com/oracle/oci-go-sdk",
    sum = "h1:XKS5XdG4ZGCXsvS0WcqBn/qOGo4EIfL4egx9jo3ackM=",
    version = "v2.0.0+incompatible",
    deps = [],
)

go_get(
    name = "github.com_patrickmn_go-cache",
    download_deps = [],
    install = [""],
    module = "github.com/patrickmn/go-cache",
    sum = "h1:HRMgzkcYKYpi3C8ajMPV8OFXaaRUnok+kx1WdO15EQc=",
    version = "v2.1.0+incompatible",
    deps = [],
)

go_get(
    name = "github.com_pborman_uuid",
    download_deps = [],
    install = [""],
    module = "github.com/pborman/uuid",
    sum = "h1:J7Q5mO4ysT1dv8hyrUGHb9+ooztCXu1D8MY8DZYsu3g=",
    version = "v1.2.0",
    deps = [":github.com_google_uuid"],
)

go_get(
    name = "github.com_pelletier_go-toml",
    download_deps = [],
    install = [""],
    module = "github.com/pelletier/go-toml",
    sum = "h1:aetoXYr0Tv7xRU/V4B4IZJ2QcbtMUFoNb3ORp7TzIK4=",
    version = "v1.6.0",
    deps = [],
)

go_get(
    name = "github.com_peterbourgon_diskv",
    download_deps = [],
    install = [""],
    module = "github.com/peterbourgon/diskv",
    sum = "h1:UBdAOUP5p4RWqPBg048CAvpKN+vxiaj6gdUUzhl4XmI=",
    version = "v2.0.1+incompatible",
    deps = [":github.com_google_btree"],
)

go_get(
    name = "github.com_pierrec_lz4",
    download_deps = [],
    install = [
        "internal/xxh32",
        "",
    ],
    module = "github.com/pierrec/lz4",
    sum = "h1:2xWsjqPFWcplujydGg4WmhC/6fZqK42wMM8aXeqhl0I=",
    version = "v2.0.5+incompatible",
    deps = [],
)

go_get(
    name = "github.com_pkg_errors",
    download_deps = [],
    install = [""],
    module = "github.com/pkg/errors",
    sum = "h1:FEBLx1zS214owpjy7qsBeixbURkuhQAwrK5UwLGTwt4=",
    version = "v0.9.1",
    deps = [],
)

go_get(
    name = "github.com_pmezard_go-difflib",
    download_deps = [],
    install = ["difflib"],
    module = "github.com/pmezard/go-difflib",
    sum = "h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "github.com_pquerna_cachecontrol",
    download_deps = [],
    install = [
        "cacheobject",
        "",
    ],
    module = "github.com/pquerna/cachecontrol",
    sum = "h1:0XM1XL/OFFJjXsYXlG30spTkV/E9+gmd5GD1w2HE8xM=",
    version = "v0.0.0-20171018203845-0dec1b30a021",
    deps = [],
)

go_get(
    name = "github.com_prometheus_client_golang",
    download_deps = [],
    install = [
        "prometheus/internal",
        "prometheus",
        "prometheus/promhttp",
        "api",
        "api/prometheus/v1",
    ],
    module = "github.com/prometheus/client_golang",
    sum = "h1:FFSuS004yOQEtDdTq+TAOLP5xUq63KqAFYyOi8zA+Y8=",
    version = "v1.4.1",
    deps = [
        ":github.com_beorn7_perks",
        ":github.com_cespare_xxhash_v2",
        ":github.com_golang_protobuf",
        ":github.com_json-iterator_go",
        ":github.com_prometheus_client_model",
        ":github.com_prometheus_common",
        ":github.com_prometheus_procfs",
    ],
)

go_get(
    name = "github.com_prometheus_client_model",
    download_deps = [],
    install = ["go"],
    module = "github.com/prometheus/client_model",
    sum = "h1:uq5h0d+GuxiXLJLNABMgp2qUWDPiLvgCzz2dUR+/W/M=",
    version = "v0.2.0",
    deps = [":github.com_golang_protobuf"],
)

go_get(
    name = "github.com_prometheus_common",
    download_deps = [],
    install = [
        "internal/bitbucket.org/ww/goautoneg",
        "model",
        "expfmt",
    ],
    module = "github.com/prometheus/common",
    sum = "h1:KOMtN28tlbam3/7ZKEYKHhKoJZYYj3gMH4uc62x7X7U=",
    version = "v0.9.1",
    deps = [
        ":github.com_golang_protobuf",
        ":github.com_matttproud_golang_protobuf_extensions",
        ":github.com_prometheus_client_model",
    ],
)

go_get(
    name = "github.com_prometheus_procfs",
    download_deps = [],
    install = [
        "internal/fs",
        "internal/util",
        "",
    ],
    module = "github.com/prometheus/procfs",
    sum = "h1:DksSrntiTPE63NQuxGcFa1OS/odKfwJu3PJHrhKAy7Q=",
    version = "v0.0.9",
    deps = [],
)

go_get(
    name = "github.com_qor_assetfs",
    download_deps = [],
    install = [""],
    module = "github.com/qor/assetfs",
    sum = "h1:JRpyNNSRAkwNHd4WgyPcalTAhxOCh3eFNMoQkxWhjSw=",
    version = "v0.0.0-20170713023933-ff57fdc13a14",
    deps = [],
)

go_get(
    name = "github.com_qor_auth",
    download_deps = [],
    install = [
        "claims",
        "auth_identity",
        "",
    ],
    module = "github.com/qor/auth",
    replace = "github.com/banzaicloud/auth",
    sum = "h1:EUl1EMnJ2AfRGq6xn/SKdRY39ZULGyFj0XJDImv1aF4=",
    version = "v0.1.3",
    deps = [
        ":github.com_dgrijalva_jwt-go",
        ":github.com_jinzhu_copier",
        ":github.com_jinzhu_gorm",
        ":github.com_qor_mailer",
        ":github.com_qor_qor",
        ":github.com_qor_redirect_back",
        ":github.com_qor_render",
        ":github.com_qor_responder",
        ":github.com_qor_session",
    ],
)

go_get(
    name = "github.com_qor_mailer",
    download_deps = [],
    install = [
        "",
        "logger",
    ],
    module = "github.com/qor/mailer",
    sum = "h1:xD5EB1F5pmi1J48juHXdlOdLrhu/i7c7zvHdzgLMEJ4=",
    version = "v0.0.0-20170814094430-1e6ac7106955",
    deps = [
        ":github.com_qor_assetfs",
        ":github.com_qor_render",
        ":gopkg.in_gomail.v2",
    ],
)

go_get(
    name = "github.com_qor_middlewares",
    download_deps = [],
    install = [""],
    module = "github.com/qor/middlewares",
    sum = "h1:+WCc1IigwWpWBxMFsmLUsIF230TakGHstDajd8aKDAc=",
    version = "v0.0.0-20170822143614-781378b69454",
    deps = [],
)

go_get(
    name = "github.com_qor_qor",
    download_deps = [],
    install = [
        "",
        "utils",
    ],
    module = "github.com/qor/qor",
    sum = "h1:0z+LJ7Efz/a+SYR2Wr/CfSyLuFgzSVVFyu9hqGEY74Y=",
    version = "v0.0.0-20190319081902-186b0237364b",
    deps = [
        ":github.com_gosimple_slug",
        ":github.com_jinzhu_gorm",
        ":github.com_jinzhu_now",
        ":github.com_microcosm-cc_bluemonday",
    ],
)

go_get(
    name = "github.com_qor_redirect_back",
    download_deps = [],
    install = [""],
    module = "github.com/qor/redirect_back",
    sum = "h1:p8fFfW3hQzNMqXxr+LBrr82MIXoUi1HCXbeM8oCgfnw=",
    version = "v0.0.0-20170907030740-b4161ed6f848",
    deps = [
        ":github.com_qor_middlewares",
        ":github.com_qor_qor",
        ":github.com_qor_session",
    ],
)

go_get(
    name = "github.com_qor_render",
    download_deps = [],
    install = [""],
    module = "github.com/qor/render",
    sum = "h1:4aq245m+d7ulq2cMQitfRP32qII5MwgE5tEe+UjfcEw=",
    version = "v0.0.0-20171201033449-63566e46f01b",
    deps = [
        ":github.com_qor_assetfs",
        ":github.com_qor_qor",
    ],
)

go_get(
    name = "github.com_qor_responder",
    download_deps = [],
    install = [""],
    module = "github.com/qor/responder",
    sum = "h1:g77aB9KGnuMC2kNSUWO2KN8k2RzgbI0HqOvWA+AlKX8=",
    version = "v0.0.0-20160314063933-ecae0be66c1a",
    deps = [],
)

go_get(
    name = "github.com_qor_session",
    download_deps = [],
    install = [
        "",
        "gorilla",
        "manager",
    ],
    module = "github.com/qor/session",
    sum = "h1:8l21EEdlZ9R0AA3FbeUAANc5NAx8Y3tn1VKbyAgjYlI=",
    version = "v0.0.0-20170907035918-8206b0adab70",
    deps = [
        ":github.com_gorilla_context",
        ":github.com_gorilla_sessions",
        ":github.com_qor_middlewares",
        ":github.com_qor_qor",
    ],
)

go_get(
    name = "github.com_rainycape_unidecode",
    download_deps = [],
    install = [""],
    module = "github.com/rainycape/unidecode",
    sum = "h1:ta7tUOvsPHVHGom5hKW5VXNc2xZIkfCKP8iaqOyYtUQ=",
    version = "v0.0.0-20150907023854-cb7f23ec59be",
    deps = [],
)

go_get(
    name = "github.com_robfig_cron",
    download_deps = [],
    install = [""],
    module = "github.com/robfig/cron",
    sum = "h1:ZjScXvvxeQ63Dbyxy76Fj3AT3Ut0aKsyd2/tl3DTMuQ=",
    version = "v1.2.0",
    deps = [],
)

go_get(
    name = "github.com_rubenv_sql-migrate",
    download_deps = [],
    install = [
        "sqlparse",
        "",
    ],
    module = "github.com/rubenv/sql-migrate",
    sum = "h1:xkBtI5JktwbW/vf4vopBbhYsRFTGfQWHYXzC0/qYwxI=",
    version = "v0.0.0-20200212082348-64f95ea68aa3",
    deps = [":gopkg.in_gorp.v1"],
)

go_get(
    name = "github.com_russross_blackfriday",
    download_deps = [],
    install = [""],
    module = "github.com/russross/blackfriday",
    sum = "h1:HyvC0ARfnZBqnXwABFeSZHpKvJHJJfPz81GNueLj0oo=",
    version = "v1.5.2",
    deps = [],
)

go_get(
    name = "github.com_ryanuber_go-glob",
    download_deps = [],
    install = [""],
    module = "github.com/ryanuber/go-glob",
    sum = "h1:iQh3xXAumdQ+4Ufa5b25cRpC5TYKlno6hsv6Cb3pkBk=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "github.com_sagikazarmark_appkit",
    download_deps = [],
    install = [
        "errors",
        "transport/http",
        "endpoint",
        "run",
    ],
    module = "github.com/sagikazarmark/appkit",
    sum = "h1:/Nv3L9SfbAa/2uM1+jWBPAnfo52Ei/ylwm/Ow1moHkA=",
    version = "v0.8.0",
    deps = [
        ":github.com_go-kit_kit",
        ":github.com_moogar0880_problems",
    ],
)

go_get(
    name = "github.com_sagikazarmark_kitx",
    download_deps = [],
    install = [
        "endpoint",
        "transport/http",
        "correlation",
        "transport",
    ],
    module = "github.com/sagikazarmark/kitx",
    sum = "h1:19phhwBe4iVOy1Kam4dqGuxFT59/Wlp0xI239PhpaiQ=",
    version = "v0.12.0",
    deps = [
        ":github.com_go-kit_kit",
        ":github.com_moogar0880_problems",
        ":github.com_pkg_errors",
        ":google.golang.org_grpc",
    ],
)

go_get(
    name = "github.com_sagikazarmark_ocmux",
    download_deps = [],
    install = [""],
    module = "github.com/sagikazarmark/ocmux",
    sum = "h1:YmxEdmIFCeM9NO99U3DBvLRq+cIy2gQJ4zDyifhOqWU=",
    version = "v0.2.0",
    deps = [
        ":github.com_gorilla_mux",
        ":go.opencensus.io",
    ],
)

go_get(
    name = "github.com_sirupsen_logrus",
    download_deps = [],
    install = [""],
    module = "github.com/sirupsen/logrus",
    sum = "h1:UBcNElsrwanuuMsnGSlYmtmgbb23qDR5dG+6X6Oo89I=",
    version = "v1.6.0",
    deps = [":golang.org_x_sys"],
)

go_get(
    name = "github.com_spf13_afero",
    download_deps = [],
    install = [
        "mem",
        "",
    ],
    module = "github.com/spf13/afero",
    sum = "h1:5jhuqJyZCZf2JRofRvN/nIFgIWNzPa3/Vz8mYylgbWc=",
    version = "v1.2.2",
    deps = [":golang.org_x_text"],
)

go_get(
    name = "github.com_spf13_cast",
    download_deps = [],
    install = [""],
    module = "github.com/spf13/cast",
    sum = "h1:nFm6S0SMdyzrzcmThSipiEubIDy8WEXKNZ0UOgiRpng=",
    version = "v1.3.1",
    deps = [],
)

go_get(
    name = "github.com_spf13_cobra",
    download_deps = [],
    install = [""],
    module = "github.com/spf13/cobra",
    sum = "h1:6m/oheQuQ13N9ks4hubMG6BnvwOeaJrqSPLahSnczz8=",
    version = "v1.0.0",
    deps = [":github.com_spf13_pflag"],
)

go_get(
    name = "github.com_spf13_jwalterweatherman",
    download_deps = [],
    install = [""],
    module = "github.com/spf13/jwalterweatherman",
    sum = "h1:ue6voC5bR5F8YxI5S67j9i582FU4Qvo2bmqnqMYADFk=",
    version = "v1.1.0",
    deps = [],
)

go_get(
    name = "github.com_spf13_pflag",
    download_deps = [],
    install = [""],
    module = "github.com/spf13/pflag",
    sum = "h1:iy+VFUOCP1a+8yFto/drg2CJ5u0yRoB7fZw3DKv/JXA=",
    version = "v1.0.5",
    deps = [],
)

go_get(
    name = "github.com_spf13_viper",
    download_deps = [],
    install = [""],
    module = "github.com/spf13/viper",
    sum = "h1:xVKxvI7ouOI5I+U9s2eeiUfMaWBVoXA3AWskkrqK0VM=",
    version = "v1.7.0",
    deps = [
        ":github.com_fsnotify_fsnotify",
        ":github.com_hashicorp_hcl",
        ":github.com_magiconair_properties",
        ":github.com_mitchellh_mapstructure",
        ":github.com_pelletier_go-toml",
        ":github.com_spf13_afero",
        ":github.com_spf13_cast",
        ":github.com_spf13_jwalterweatherman",
        ":github.com_spf13_pflag",
        ":github.com_subosito_gotenv",
        ":gopkg.in_ini.v1",
        ":gopkg.in_yaml.v2",
    ],
)

go_get(
    name = "github.com_stretchr_objx",
    download_deps = [],
    install = [""],
    module = "github.com/stretchr/objx",
    sum = "h1:Hbg2NidpLE8veEBkEZTL3CvlkUIVzuU9jDplZO54c48=",
    version = "v0.2.0",
    deps = [],
)

go_get(
    name = "github.com_stretchr_testify",
    download_deps = [],
    install = [
        "assert",
        "mock",
        "require",
        "suite",
    ],
    module = "github.com/stretchr/testify",
    sum = "h1:hDPOHmpOpP40lSULcqw7IrRb/u7w6RpDC9399XyoNd0=",
    version = "v1.6.1",
    deps = [
        ":github.com_davecgh_go-spew",
        ":github.com_pmezard_go-difflib",
        ":github.com_stretchr_objx",
        ":gopkg.in_yaml.v3",
    ],
)

go_get(
    name = "github.com_subosito_gotenv",
    download_deps = [],
    install = [""],
    module = "github.com/subosito/gotenv",
    sum = "h1:Slr1R9HxAlEKefgq5jn9U+DnETlIUa6HfgEzj0g5d7s=",
    version = "v1.2.0",
    deps = [],
)

go_get(
    name = "github.com_technosophos_moniker",
    download_deps = [],
    install = [""],
    module = "github.com/technosophos/moniker",
    sum = "h1:DNVk+NIkGS0RbLkjQOLCJb/759yfCysThkMbl7EXxyY=",
    version = "v0.0.0-20180509230615-a5dbd03a2245",
    deps = [],
)

go_get(
    name = "github.com_uber-go_mapdecode",
    download_deps = [],
    install = [
        "internal/mapstructure",
        "",
    ],
    module = "github.com/uber-go/mapdecode",
    sum = "h1:euUEFM9KnuCa1OBixz1xM+FIXmpixyay5DLymceOVrU=",
    version = "v1.0.0",
    deps = [":go.uber.org_multierr"],
)

go_get(
    name = "github.com_uber-go_tally",
    download_deps = [],
    install = [""],
    module = "github.com/uber-go/tally",
    sum = "h1:9hLSgNBP28CjIaDmAuRTq9qV+UZY+9PcvAkXO4nNMwg=",
    version = "v3.3.15+incompatible",
    deps = [],
)

go_get(
    name = "github.com_uber_tchannel-go",
    download_deps = [],
    install = [
        "internal/argreader",
        "thrift/arg2",
        "relay",
        "tnet",
        "tos",
        "trand",
        "typed",
        "",
    ],
    module = "github.com/uber/tchannel-go",
    sum = "h1:2Hxvd+LaQyb9Rza+KoobJU6lhabIo1+cImk/rdLfiuQ=",
    version = "v1.18.0",
    deps = [
        ":github.com_opentracing_opentracing-go",
        ":go.uber.org_atomic",
        ":go.uber.org_multierr",
        ":golang.org_x_net",
        ":golang.org_x_sys",
    ],
)

go_get(
    name = "github.com_ugorji_go_codec",
    download_deps = [],
    install = [""],
    module = "github.com/ugorji/go/codec",
    sum = "h1:2SvQaVZ1ouYrrKKwoSk2pzd4A9evlKJb9oTL+OaLUSs=",
    version = "v1.1.7",
    deps = [],
)

go_get(
    name = "github.com_vmware_govmomi",
    download_deps = [],
    install = [
        "vim25/debug",
        "vim25/progress",
        "vim25/types",
        "vim25/xml",
        "vim25/soap",
        "vim25/methods",
        "vim25",
        "vim25/mo",
        "property",
        "session",
        "",
        "list",
        "nfc",
        "task",
        "object",
        "find",
    ],
    module = "github.com/vmware/govmomi",
    sum = "h1:fIg5KkUxu4b/y1BmmtNFO9pyN10CkVzvriaJyynAdE8=",
    version = "v0.22.0",
    deps = [],
)

go_get(
    name = "github.com_xeipuuv_gojsonpointer",
    download_deps = [],
    install = [""],
    module = "github.com/xeipuuv/gojsonpointer",
    sum = "h1:J9EGpcZtP0E/raorCMxlFGSTBrsSlaDGf3jU/qvAE2c=",
    version = "v0.0.0-20180127040702-4e3ac2762d5f",
    deps = [],
)

go_get(
    name = "github.com_xeipuuv_gojsonreference",
    download_deps = [],
    install = [""],
    module = "github.com/xeipuuv/gojsonreference",
    sum = "h1:EzJWgHovont7NscjpAxXsDA8S8BMYve8Y5+7cuRE7R0=",
    version = "v0.0.0-20180127040603-bd5ef7bd5415",
    deps = [":github.com_xeipuuv_gojsonpointer"],
)

go_get(
    name = "github.com_xeipuuv_gojsonschema",
    download_deps = [],
    install = [""],
    module = "github.com/xeipuuv/gojsonschema",
    sum = "h1:ngVtJC9TY/lg0AA/1k48FYhBrhRoFlEmWzsehpNAaZg=",
    version = "v1.1.0",
    deps = [":github.com_xeipuuv_gojsonreference"],
)

go_get(
    name = "go.opencensus.io",
    download_deps = [],
    install = [
        "",
        "internal",
        "trace/internal",
        "trace/tracestate",
        "trace",
        "trace/propagation",
        "plugin/ochttp/propagation/b3",
        "resource",
        "metric/metricdata",
        "tag",
        "stats/internal",
        "stats",
        "internal/tagencoding",
        "metric/metricproducer",
        "stats/view",
        "plugin/ochttp",
        "plugin/ocgrpc",
    ],
    module = "go.opencensus.io",
    sum = "h1:75k/FF0Q2YM8QYo07VPddOLBslDt1MZOdEslOHvmzAs=",
    version = "v0.22.2",
    deps = [
        ":github.com_golang_groupcache",
        ":google.golang.org_grpc",
    ],
)

go_get(
    name = "go.uber.org_atomic",
    download_deps = [],
    install = [""],
    module = "go.uber.org/atomic",
    sum = "h1:Ezj3JGmsOnG1MoRWQkPBsKLe9DwWD9QeXzTRzzldNVk=",
    version = "v1.6.0",
    deps = [],
)

go_get(
    name = "go.uber.org_cadence",
    download_deps = [],
    install = [
        ".gen/go/shared",
        ".gen/go/cadence",
        ".gen/go/cadence/workflowserviceclient",
        ".gen/go/cadence/workflowservicetest",
        "internal/common",
        "internal/common/backoff",
        "internal/common/cache",
        "internal/common/metrics",
        "internal/common/util",
        "internal",
        "encoded",
        "workflow",
        "client",
        "",
        "activity",
        "worker",
        "mocks",
        "testsuite",
    ],
    module = "go.uber.org/cadence",
    sum = "h1:NtAV+JTBFyOrUVTqI+acbfqn68Yrzj4+sX4NPACTt60=",
    version = "v0.9.0",
    deps = [
        ":github.com_apache_thrift",
        ":github.com_facebookgo_clock",
        ":github.com_golang_mock",
        ":github.com_opentracing_opentracing-go",
        ":github.com_pborman_uuid",
        ":github.com_robfig_cron",
        ":github.com_stretchr_testify",
        ":github.com_uber-go_tally",
        ":go.uber.org_atomic",
        ":go.uber.org_thriftrw",
        ":go.uber.org_yarpc",
        ":go.uber.org_zap",
        ":golang.org_x_time",
    ],
)

go_get(
    name = "go.uber.org_multierr",
    download_deps = [],
    install = [""],
    module = "go.uber.org/multierr",
    sum = "h1:KCa4XfM8CWFCpxXRGok+Q0SS/0XBhMDbHHGABQLvD2A=",
    version = "v1.5.0",
    deps = [":go.uber.org_atomic"],
)

go_get(
    name = "go.uber.org_net_metrics",
    download_deps = [],
    install = [
        "push",
        "",
        "tallypush",
        "bucket",
    ],
    module = "go.uber.org/net/metrics",
    sum = "h1:iRLPuVecNYf/wIV+mQaA4IgN8ghifu3q1B4IT6HfwyY=",
    version = "v1.3.0",
    deps = [
        ":github.com_prometheus_client_golang",
        ":github.com_prometheus_client_model",
        ":github.com_uber-go_tally",
        ":go.uber.org_atomic",
    ],
)

go_get(
    name = "go.uber.org_thriftrw",
    download_deps = [],
    install = [
        "thriftreflect",
        "wire",
        "version",
        "internal/envelope/exception",
        "protocol/binary",
        "protocol",
        "envelope",
    ],
    module = "go.uber.org/thriftrw",
    sum = "h1:jdVWqCcoT8V9eh4XGaLdGsbFxXtX2huYcBbeXWBjS/w=",
    version = "v1.23.0",
    deps = [
        ":go.uber.org_multierr",
        ":go.uber.org_zap",
    ],
)

go_get(
    name = "go.uber.org_yarpc",
    download_deps = [],
    install = [
        "yarpcerrors",
        "api/transport",
        "internal/inboundcall",
        "api/encoding",
        "internal/introspection",
        "api/middleware",
        "internal",
        "internal/errorsync",
        "internal/firstoutboundmiddleware",
        "internal/humanize",
        "internal/inboundmiddleware",
        "internal/digester",
        "internal/observability",
        "internal/outboundmiddleware",
        "internal/request",
        "pkg/lifecycle",
        "",
        "encoding/thrift/internal",
        "internal/bufferpool",
        "pkg/encoding",
        "pkg/errors",
        "pkg/procedure",
        "encoding/thrift",
        "api/backoff",
        "api/peer",
        "internal/backoff",
        "internal/iopool",
        "internal/yarpcerrors",
        "peer",
        "peer/abstractpeer",
        "peer/hostport",
        "transport/tchannel/internal",
        "internal/interpolate",
        "internal/config",
        "yarpcconfig",
        "transport/tchannel",
    ],
    module = "go.uber.org/yarpc",
    sum = "h1:lyOdulfJ6i9C2jdFd+6Qoj1RiDwv64hCHOZYXD8ofhU=",
    version = "v1.45.0",
    deps = [
        ":github.com_opentracing_opentracing-go",
        ":github.com_uber-go_mapdecode",
        ":github.com_uber-go_tally",
        ":github.com_uber_tchannel-go",
        ":go.uber.org_atomic",
        ":go.uber.org_multierr",
        ":go.uber.org_net_metrics",
        ":go.uber.org_thriftrw",
        ":go.uber.org_zap",
        ":golang.org_x_net",
        ":gopkg.in_yaml.v2",
    ],
)

go_get(
    name = "go.uber.org_zap",
    download_deps = [],
    install = [
        "buffer",
        "internal/bufferpool",
        "internal/color",
        "internal/exit",
        "zapcore",
        "",
    ],
    module = "go.uber.org/zap",
    sum = "h1:nYDKopTbvAPq/NrUVZwT15y2lpROBiLLyoRTbXOYWOo=",
    version = "v1.14.1",
    deps = [
        ":go.uber.org_atomic",
        ":go.uber.org_multierr",
    ],
)

go_get(
    name = "golang.org_x_crypto",
    download_deps = [],
    install = [
        "ed25519",
        "pbkdf2",
        "ssh/terminal",
        "pkcs12/internal/rc2",
        "pkcs12",
        "internal/subtle",
        "chacha20",
        "curve25519",
        "poly1305",
        "blowfish",
        "ssh/internal/bcrypt_pbkdf",
        "ssh",
        "scrypt",
        "openpgp/errors",
        "openpgp/armor",
        "cast5",
        "openpgp/elgamal",
        "openpgp/s2k",
        "openpgp/packet",
        "openpgp",
        "openpgp/clearsign",
        "bcrypt",
    ],
    module = "golang.org/x/crypto",
    sum = "h1:/Tl7pH94bvbAAHBdZJT947M/+gp0+CqQXDtMRC0fseo=",
    version = "v0.0.0-20200220183623-bac4c82f6975",
    deps = [":golang.org_x_sys"],
)

go_get(
    name = "golang.org_x_net",
    download_deps = [],
    install = [
        "idna",
        "http/httpguts",
        "http2/hpack",
        "http2",
        "context/ctxhttp",
        "internal/timeseries",
        "trace",
        "context",
        "bpf",
        "internal/iana",
        "internal/socket",
        "ipv4",
        "ipv6",
        "html/atom",
        "html",
        "publicsuffix",
        "html/charset",
    ],
    module = "golang.org/x/net",
    sum = "h1:QPwSajcTUrFriMF1nJ3XzgoqakqQEsnZf9LdXdi2nkI=",
    version = "v0.0.0-20200421231249-e086a090c8fd",
    deps = [
        ":golang.org_x_sys",
        ":golang.org_x_text",
    ],
)

go_get(
    name = "golang.org_x_oauth2",
    download_deps = [],
    install = [
        "internal",
        "",
        "jws",
        "jwt",
        "google",
    ],
    module = "golang.org/x/oauth2",
    sum = "h1:TzXSXBo42m9gQenoE3b9BGiEpg5IG2JkU5FkPIawgtw=",
    version = "v0.0.0-20200107190931-bf48bf16ab8d",
    deps = [
        ":cloud.google.com_go_compute_metadata",
        ":golang.org_x_net",
    ],
)

go_get(
    name = "golang.org_x_sync",
    download_deps = [],
    install = [
        "errgroup",
        "semaphore",
    ],
    module = "golang.org/x/sync",
    sum = "h1:vcxGaoTs7kV8m5Np9uUNQin4BrLOthgV7252N8V+FwY=",
    version = "v0.0.0-20190911185100-cd5d95a43a6e",
    deps = [],
)

go_get(
    name = "golang.org_x_sys",
    download_deps = [],
    install = ["unix"],
    module = "golang.org/x/sys",
    sum = "h1:xhmwyvizuTgC2qz7ZlMluP20uW+C3Rm0FD/WLDX8884=",
    version = "v0.0.0-20200323222414-85ca7c5b95cd",
    deps = [],
)

go_get(
    name = "golang.org_x_text",
    download_deps = [],
    install = [
        "transform",
        "unicode/bidi",
        "secure/bidirule",
        "unicode/norm",
        "encoding/internal/identifier",
        "encoding",
        "encoding/internal",
        "internal/utf8internal",
        "runes",
        "encoding/unicode",
        "width",
        "encoding/charmap",
        "encoding/japanese",
        "encoding/korean",
        "encoding/simplifiedchinese",
        "encoding/traditionalchinese",
        "internal/tag",
        "internal/language",
        "internal/language/compact",
        "language",
        "encoding/htmlindex",
    ],
    module = "golang.org/x/text",
    sum = "h1:tW2bmiBqwgJj/UpqtC8EpXEZVYOwU0yG4iWbprSVAcs=",
    version = "v0.3.2",
    deps = [],
)

go_get(
    name = "golang.org_x_time",
    download_deps = [],
    install = ["rate"],
    module = "golang.org/x/time",
    sum = "h1:NusfzzA6yGQ+ua51ck7E3omNUX/JuqbFSaRGqU8CcLI=",
    version = "v0.0.0-20200416051211-89c76fbcd5d1",
    deps = [],
)

go_get(
    name = "golang.org_x_xerrors",
    download_deps = [],
    install = [
        "internal",
        "",
    ],
    module = "golang.org/x/xerrors",
    sum = "h1:E7g+9GITq07hpfrRu66IVDexMakfv52eLZ2CXBWiKr4=",
    version = "v0.0.0-20191204190536-9bdfabe68543",
    deps = [],
)

go_get(
    name = "google.golang.org_api",
    download_deps = [],
    install = [
        "internal/third_party/uritemplates",
        "googleapi",
        "internal/gensupport",
        "internal",
        "option",
        "googleapi/transport",
        "transport/http/internal/propagation",
        "transport/http",
        "cloudresourcemanager/v1",
        "transport/grpc",
        "transport",
        "container/v1",
        "serviceusage/v1",
        "iterator",
        "storage/v1",
        "compute/v1",
        "support/bundler",
    ],
    module = "google.golang.org/api",
    sum = "h1:yzlyyDW/J0w8yNFJIhiAJy4kq74S+1DOLdawELNxFMA=",
    version = "v0.15.0",
    deps = [
        ":github.com_googleapis_gax-go_v2",
        ":go.opencensus.io",
        ":golang.org_x_oauth2",
        ":golang.org_x_sync",
        ":google.golang.org_grpc",
    ],
)

go_module_download(
    name = "google.golang.org_genproto",
    module = "google.golang.org/genproto",
    sum = "h1:1mbrb1tUU+Zmt5C94IGKADBTJZjZXAd+BubWi7r9EiI=",
    tag = "download",
    version = "v0.0.0-20200212174721-66ed5ce911ce",
    deps = [],
)

go_get(
    name = "google.golang.org_genproto_googleapis_rpc_status",
    src = ":_google.golang.org_genproto#download",
    install = ["googleapis/rpc/status"],
    module = "google.golang.org/genproto",
    deps = [":github.com_golang_protobuf"],
)

go_get(
    name = "google.golang.org_genproto_googleapis_api_annotations",
    src = ":_google.golang.org_genproto#download",
    install = ["googleapis/api/annotations"],
    module = "google.golang.org/genproto",
    deps = [":github.com_golang_protobuf"],
)

go_get(
    name = "google.golang.org_genproto_googleapis_iam_credentials_v1",
    src = ":_google.golang.org_genproto#download",
    install = ["googleapis/iam/credentials/v1"],
    module = "google.golang.org/genproto",
    deps = [
        ":github.com_golang_protobuf",
        ":google.golang.org_genproto_googleapis_api_annotations",
        ":google.golang.org_grpc",
    ],
)

go_get(
    name = "google.golang.org_genproto_googleapis_type_expr",
    src = ":_google.golang.org_genproto#download",
    install = ["googleapis/type/expr"],
    module = "google.golang.org/genproto",
    deps = [":github.com_golang_protobuf"],
)

go_get(
    name = "google.golang.org_genproto_googleapis_iam_v1",
    src = ":_google.golang.org_genproto#download",
    install = ["googleapis/iam/v1"],
    module = "google.golang.org/genproto",
    deps = [
        ":github.com_golang_protobuf",
        ":google.golang.org_genproto_googleapis_api_annotations",
        ":google.golang.org_genproto_googleapis_type_expr",
        ":google.golang.org_grpc",
    ],
)

go_get(
    name = "google.golang.org_genproto_googleapis_rpc_code",
    src = ":_google.golang.org_genproto#download",
    install = ["googleapis/rpc/code"],
    module = "google.golang.org/genproto",
    deps = [":github.com_golang_protobuf"],
)

go_get(
    name = "google.golang.org_genproto_googleapis_devtools_clouderrorreporting_v1beta1",
    src = ":_google.golang.org_genproto#download",
    install = ["googleapis/devtools/clouderrorreporting/v1beta1"],
    module = "google.golang.org/genproto",
    deps = [
        ":github.com_golang_protobuf",
        ":google.golang.org_genproto_googleapis_api_annotations",
        ":google.golang.org_grpc",
    ],
)

go_get(
    name = "google.golang.org_grpc",
    download_deps = [],
    install = [
        "backoff",
        "grpclog",
        "connectivity",
        "credentials/internal",
        "internal",
        "credentials",
        "metadata",
        "attributes",
        "serviceconfig",
        "resolver",
        "balancer",
        "balancer/base",
        "internal/grpcrand",
        "balancer/roundrobin",
        "codes",
        "encoding",
        "encoding/proto",
        "internal/backoff",
        "internal/balancerload",
        "binarylog/grpc_binarylog_v1",
        "status",
        "internal/binarylog",
        "internal/buffer",
        "internal/channelz",
        "internal/envconfig",
        "internal/grpcsync",
        "internal/resolver/dns",
        "internal/resolver/passthrough",
        "internal/syscall",
        "keepalive",
        "peer",
        "stats",
        "tap",
        "internal/transport",
        "naming",
        "",
        "balancer/grpclb/grpc_lb_v1",
        "balancer/grpclb",
        "credentials/alts/internal",
        "credentials/alts/internal/proto/grpc_gcp",
        "credentials/alts/internal/authinfo",
        "credentials/alts/internal/conn",
        "credentials/alts/internal/handshaker",
        "credentials/alts/internal/handshaker/service",
        "credentials/alts",
        "credentials/oauth",
        "credentials/google",
        "health/grpc_health_v1",
    ],
    module = "google.golang.org/grpc",
    sum = "h1:zvIju4sqAGvwKspUQOhwnpcqSbzi7/H6QomNNjTL4sk=",
    version = "v1.27.1",
    deps = [
        ":github.com_golang_protobuf",
        ":github.com_google_go-cmp",
        ":golang.org_x_net",
        ":golang.org_x_oauth2",
        ":google.golang.org_genproto_googleapis_rpc_status",
    ],
)

go_get(
    name = "gopkg.in_go-playground_validator.v9",
    download_deps = [],
    install = [""],
    module = "gopkg.in/go-playground/validator.v9",
    sum = "h1:SvGtYmN60a5CVKTOzMSyfzWDeZRxRuGvRQyEAKbw1xc=",
    version = "v9.29.1",
    deps = [
        ":github.com_go-playground_universal-translator",
        ":github.com_leodido_go-urn",
    ],
)

go_get(
    name = "gopkg.in_gomail.v2",
    download_deps = [],
    install = [""],
    module = "gopkg.in/gomail.v2",
    sum = "h1:n7WqCuqOuCbNr617RXOY0AWRXxgwEyPp2z+p0+hgMuE=",
    version = "v2.0.0-20160411212932-81ebce5c23df",
    deps = [],
)

go_get(
    name = "gopkg.in_gorp.v1",
    download_deps = [],
    install = [""],
    module = "gopkg.in/gorp.v1",
    sum = "h1:j3DWlAyGVv8whO7AcIWznQ2Yj7yJkn34B8s63GViAAw=",
    version = "v1.7.2",
    deps = [],
)

go_get(
    name = "gopkg.in_inf.v0",
    download_deps = [],
    install = [""],
    module = "gopkg.in/inf.v0",
    sum = "h1:73M5CoZyi3ZLMOyDlQh031Cx6N9NDJ2Vvfl76EDAgDc=",
    version = "v0.9.1",
    deps = [],
)

go_get(
    name = "gopkg.in_ini.v1",
    download_deps = [],
    install = [""],
    module = "gopkg.in/ini.v1",
    sum = "h1:AQvPpx3LzTDM0AjnIRlVFwFFGC+npRopjZxLJj6gdno=",
    version = "v1.51.0",
    deps = [],
)

go_get(
    name = "gopkg.in_resty.v1",
    download_deps = [],
    install = [""],
    module = "gopkg.in/resty.v1",
    sum = "h1:CuXP0Pjfw9rOuY6EP+UvtNvt5DSqHpIxILZKT/quCZI=",
    version = "v1.12.0",
    deps = [":golang.org_x_net"],
)

go_get(
    name = "gopkg.in_square_go-jose.v2",
    download_deps = [],
    install = [
        "cipher",
        "json",
        "",
        "jwt",
    ],
    module = "gopkg.in/square/go-jose.v2",
    sum = "h1:SK5KegNXmKmqE342YYN2qPHEnUYeoMiXXl1poUlI+o4=",
    version = "v2.3.1",
    deps = [":golang.org_x_crypto"],
)

go_get(
    name = "gopkg.in_yaml.v2",
    download_deps = [],
    install = [""],
    module = "gopkg.in/yaml.v2",
    sum = "h1:obN1ZagJSUGI0Ek/LBmuj4SNLPfIny3KsKFopxRdj10=",
    version = "v2.2.8",
    deps = [],
)

go_get(
    name = "gopkg.in_yaml.v3",
    download_deps = [],
    install = [""],
    module = "gopkg.in/yaml.v3",
    sum = "h1:dUUwHk2QECo/6vqA44rthZ8ie2QXMNeKRTHCNY2nXvo=",
    version = "v3.0.0-20200313102051-9f266ea9e77c",
    deps = [],
)

go_get(
    name = "helm.sh_helm_v3",
    download_deps = [],
    install = [
        "pkg/chart",
        "internal/ignore",
        "internal/sympath",
        "pkg/chart/loader",
        "pkg/chartutil",
        "pkg/helmpath/xdg",
        "pkg/helmpath",
        "internal/experimental/registry",
        "internal/third_party/k8s.io/kubernetes/deployment/util",
        "pkg/kube",
        "pkg/cli",
        "pkg/provenance",
        "internal/urlutil",
        "internal/tlsutil",
        "internal/version",
        "pkg/plugin",
        "pkg/getter",
        "pkg/repo",
        "internal/resolver",
        "internal/third_party/dep/fs",
        "pkg/downloader",
        "pkg/engine",
        "pkg/kube/fake",
        "pkg/lint/support",
        "pkg/lint/rules",
        "pkg/lint",
        "pkg/postrender",
        "pkg/time",
        "pkg/release",
        "pkg/releaseutil",
        "pkg/storage/driver",
        "pkg/storage",
        "pkg/action",
        "pkg/strvals",
        "pkg/cli/values",
    ],
    module = "helm.sh/helm/v3",
    replace = "github.com/pepov/helm/v3",
    sum = "h1:ml+rfMcwaH83F3euH3TyL7AJIM6Xi24Y3mWawODmtH8=",
    version = "v3.0.0-20200519142412-df65965032e2",
    deps = [
        ":github.com_BurntSushi_toml",
        ":github.com_Masterminds_semver_v3",
        ":github.com_Masterminds_sprig_v3",
        ":github.com_asaskevich_govalidator",
        ":github.com_banzaicloud_k8s-objectmatcher",
        ":github.com_containerd_containerd",
        ":github.com_cyphar_filepath-securejoin",
        ":github.com_deislabs_oras",
        ":github.com_docker_go-units",
        ":github.com_evanphx_json-patch",
        ":github.com_gobwas_glob",
        ":github.com_gosuri_uitable",
        ":github.com_mitchellh_copystructure",
        ":github.com_opencontainers_go-digest",
        ":github.com_opencontainers_image-spec",
        ":github.com_pkg_errors",
        ":github.com_sirupsen_logrus",
        ":github.com_spf13_pflag",
        ":github.com_xeipuuv_gojsonschema",
        ":golang.org_x_crypto",
        ":k8s.io_api",
        ":k8s.io_apiextensions-apiserver",
        ":k8s.io_apimachinery",
        ":k8s.io_cli-runtime",
        ":k8s.io_client-go",
        ":k8s.io_kubectl",
        ":sigs.k8s.io_yaml",
    ],
)

go_get(
    name = "k8s.io_api",
    download_deps = [],
    install = [
        "core/v1",
        "admissionregistration/v1",
        "admissionregistration/v1beta1",
        "apps/v1",
        "apps/v1beta1",
        "apps/v1beta2",
        "auditregistration/v1alpha1",
        "authentication/v1",
        "authentication/v1beta1",
        "authorization/v1",
        "authorization/v1beta1",
        "autoscaling/v1",
        "autoscaling/v2beta1",
        "autoscaling/v2beta2",
        "batch/v1",
        "batch/v1beta1",
        "batch/v2alpha1",
        "certificates/v1beta1",
        "coordination/v1",
        "coordination/v1beta1",
        "discovery/v1alpha1",
        "discovery/v1beta1",
        "events/v1beta1",
        "extensions/v1beta1",
        "flowcontrol/v1alpha1",
        "networking/v1",
        "networking/v1beta1",
        "node/v1alpha1",
        "node/v1beta1",
        "policy/v1beta1",
        "rbac/v1",
        "rbac/v1alpha1",
        "rbac/v1beta1",
        "scheduling/v1",
        "scheduling/v1alpha1",
        "scheduling/v1beta1",
        "settings/v1alpha1",
        "storage/v1",
        "storage/v1alpha1",
        "storage/v1beta1",
        "admission/v1",
        "admission/v1beta1",
        "imagepolicy/v1alpha1",
    ],
    module = "k8s.io/api",
    replace = "k8s.io/api",
    sum = "h1:EkVieIbn1sC8YCDwckLKLpf+LoVofXYW72+LTZWo4aQ=",
    version = "v0.17.5",
    deps = [
        ":github.com_gogo_protobuf",
        ":k8s.io_apimachinery",
    ],
)

go_get(
    name = "k8s.io_apiextensions-apiserver",
    download_deps = [],
    install = [
        "pkg/features",
        "pkg/apis/apiextensions",
        "pkg/apis/apiextensions/v1beta1",
        "pkg/apis/apiextensions/v1",
        "pkg/client/clientset/clientset/scheme",
        "pkg/client/clientset/clientset/typed/apiextensions/v1beta1",
    ],
    module = "k8s.io/apiextensions-apiserver",
    replace = "k8s.io/apiextensions-apiserver",
    sum = "h1:1MvO6pRopn9ZHweFEVFxnWDRpMd3ZE7SPY156qDnOeI=",
    version = "v0.17.5",
    deps = [
        ":github.com_gogo_protobuf",
        ":k8s.io_apimachinery",
        ":k8s.io_apiserver",
        ":k8s.io_client-go",
        ":k8s.io_component-base",
        ":k8s.io_utils",
    ],
)

go_get(
    name = "k8s.io_apimachinery",
    download_deps = [],
    install = [
        "pkg/api/resource",
        "third_party/forked/golang/reflect",
        "pkg/conversion",
        "pkg/selection",
        "pkg/fields",
        "pkg/util/sets",
        "pkg/util/errors",
        "pkg/util/validation/field",
        "pkg/util/validation",
        "pkg/labels",
        "pkg/conversion/queryparams",
        "pkg/runtime/schema",
        "pkg/util/json",
        "pkg/util/naming",
        "pkg/util/runtime",
        "pkg/runtime",
        "pkg/types",
        "pkg/util/intstr",
        "pkg/util/net",
        "pkg/watch",
        "pkg/apis/meta/v1",
        "pkg/api/errors",
        "pkg/runtime/serializer/streaming",
        "pkg/version",
        "pkg/runtime/serializer/recognizer",
        "pkg/util/framer",
        "pkg/util/yaml",
        "pkg/runtime/serializer/json",
        "pkg/runtime/serializer/protobuf",
        "pkg/apis/meta/v1/unstructured",
        "pkg/runtime/serializer/versioning",
        "pkg/runtime/serializer",
        "pkg/util/clock",
        "pkg/api/meta",
        "pkg/apis/meta/v1beta1",
        "pkg/apis/meta/internalversion",
        "pkg/util/wait",
        "pkg/api/equality",
        "pkg/util/mergepatch",
        "third_party/forked/golang/json",
        "pkg/util/strategicpatch",
        "pkg/util/duration",
        "pkg/apis/meta/v1/unstructured/unstructuredscheme",
        "pkg/apis/meta/v1/validation",
        "pkg/api/validation",
        "pkg/util/cache",
        "pkg/util/diff",
        "pkg/util/httpstream",
        "third_party/forked/golang/netutil",
        "pkg/util/httpstream/spdy",
        "pkg/util/remotecommand",
        "pkg/util/rand",
        "pkg/apis/meta/internalversion/scheme",
        "pkg/util/proxy",
        "pkg/util/jsonmergepatch",
    ],
    module = "k8s.io/apimachinery",
    replace = "k8s.io/apimachinery",
    sum = "h1:QAjfgeTtSGksdkgyaPrIb4lhU16FWMIzxKejYD5S0gc=",
    version = "v0.17.5",
    deps = [
        ":github.com_davecgh_go-spew",
        ":github.com_docker_spdystream",
        ":github.com_evanphx_json-patch",
        ":github.com_gogo_protobuf",
        ":github.com_google_go-cmp",
        ":github.com_google_gofuzz",
        ":github.com_hashicorp_golang-lru",
        ":github.com_json-iterator_go",
        ":github.com_modern-go_reflect2",
        ":github.com_mxk_go-flowrate",
        ":golang.org_x_net",
        ":gopkg.in_inf.v0",
        ":k8s.io_klog",
        ":k8s.io_kube-openapi",
        ":sigs.k8s.io_yaml",
    ],
)

go_get(
    name = "k8s.io_apiserver",
    download_deps = [],
    install = [
        "pkg/util/feature",
        "pkg/features",
        "pkg/authentication/serviceaccount",
        "pkg/authentication/user",
        "pkg/authentication/authenticator",
    ],
    module = "k8s.io/apiserver",
    replace = "k8s.io/apiserver",
    sum = "h1:8RJUTT2TzVZnCl3B+6YgYtaMTIcjwo11z40yZbA85ds=",
    version = "v0.17.5",
    deps = [
        ":k8s.io_apimachinery",
        ":k8s.io_component-base",
    ],
)

go_get(
    name = "k8s.io_cli-runtime",
    download_deps = [],
    install = [
        "pkg/printers",
        "pkg/kustomize/k8sdeps/kv",
        "pkg/kustomize/k8sdeps/configmapandsecret",
        "pkg/kustomize/k8sdeps/kunstruct",
        "pkg/kustomize/k8sdeps/transformer/hash",
        "pkg/kustomize/k8sdeps/transformer/patch",
        "pkg/kustomize/k8sdeps/transformer",
        "pkg/kustomize/k8sdeps/validator",
        "pkg/kustomize/k8sdeps",
        "pkg/kustomize",
        "pkg/resource",
        "pkg/genericclioptions",
    ],
    module = "k8s.io/cli-runtime",
    replace = "k8s.io/cli-runtime",
    sum = "h1:WIlIsuLppbKU6Oixlx1LTAz+e62wj1guQWLphhvZWZg=",
    version = "v0.17.5",
    deps = [
        ":github.com_evanphx_json-patch",
        ":github.com_liggitt_tabwriter",
        ":github.com_pkg_errors",
        ":github.com_spf13_cobra",
        ":github.com_spf13_pflag",
        ":golang.org_x_text",
        ":k8s.io_api",
        ":k8s.io_apimachinery",
        ":k8s.io_client-go",
        ":sigs.k8s.io_kustomize",
        ":sigs.k8s.io_yaml",
    ],
)

go_get(
    name = "k8s.io_client-go",
    download_deps = [],
    install = [
        "pkg/version",
        "pkg/apis/clientauthentication",
        "pkg/apis/clientauthentication/v1alpha1",
        "pkg/apis/clientauthentication/v1beta1",
        "tools/clientcmd/api",
        "transport",
        "util/connrotation",
        "plugin/pkg/client/auth/exec",
        "rest/watch",
        "tools/metrics",
        "util/keyutil",
        "util/cert",
        "util/flowcontrol",
        "rest",
        "kubernetes/scheme",
        "discovery",
        "restmapper",
        "tools/auth",
        "tools/clientcmd/api/v1",
        "tools/clientcmd/api/latest",
        "util/homedir",
        "tools/clientcmd",
        "dynamic",
        "kubernetes/typed/admissionregistration/v1",
        "kubernetes/typed/admissionregistration/v1beta1",
        "kubernetes/typed/apps/v1",
        "kubernetes/typed/apps/v1beta1",
        "kubernetes/typed/apps/v1beta2",
        "kubernetes/typed/auditregistration/v1alpha1",
        "kubernetes/typed/authentication/v1",
        "kubernetes/typed/authentication/v1beta1",
        "kubernetes/typed/authorization/v1",
        "kubernetes/typed/authorization/v1beta1",
        "kubernetes/typed/autoscaling/v1",
        "kubernetes/typed/autoscaling/v2beta1",
        "kubernetes/typed/autoscaling/v2beta2",
        "kubernetes/typed/batch/v1",
        "kubernetes/typed/batch/v1beta1",
        "kubernetes/typed/batch/v2alpha1",
        "kubernetes/typed/certificates/v1beta1",
        "kubernetes/typed/coordination/v1",
        "kubernetes/typed/coordination/v1beta1",
        "tools/reference",
        "kubernetes/typed/core/v1",
        "kubernetes/typed/discovery/v1alpha1",
        "kubernetes/typed/discovery/v1beta1",
        "kubernetes/typed/events/v1beta1",
        "kubernetes/typed/extensions/v1beta1",
        "kubernetes/typed/flowcontrol/v1alpha1",
        "kubernetes/typed/networking/v1",
        "kubernetes/typed/networking/v1beta1",
        "kubernetes/typed/node/v1alpha1",
        "kubernetes/typed/node/v1beta1",
        "kubernetes/typed/policy/v1beta1",
        "kubernetes/typed/rbac/v1",
        "kubernetes/typed/rbac/v1alpha1",
        "kubernetes/typed/rbac/v1beta1",
        "kubernetes/typed/scheduling/v1",
        "kubernetes/typed/scheduling/v1alpha1",
        "kubernetes/typed/scheduling/v1beta1",
        "kubernetes/typed/settings/v1alpha1",
        "kubernetes/typed/storage/v1",
        "kubernetes/typed/storage/v1alpha1",
        "kubernetes/typed/storage/v1beta1",
        "kubernetes",
        "third_party/forked/golang/template",
        "util/jsonpath",
        "discovery/cached/disk",
        "tools/pager",
        "util/retry",
        "tools/cache",
        "tools/portforward",
        "tools/watch",
        "transport/spdy",
        "scale/scheme",
        "scale/scheme/appsint",
        "scale/scheme/appsv1beta1",
        "scale/scheme/appsv1beta2",
        "scale/scheme/autoscalingv1",
        "scale/scheme/extensionsint",
        "scale/scheme/extensionsv1beta1",
        "scale",
        "util/exec",
        "tools/remotecommand",
        "informers/internalinterfaces",
        "listers/admissionregistration/v1",
        "informers/admissionregistration/v1",
        "listers/admissionregistration/v1beta1",
        "informers/admissionregistration/v1beta1",
        "informers/admissionregistration",
        "listers/apps/v1",
        "informers/apps/v1",
        "listers/apps/v1beta1",
        "informers/apps/v1beta1",
        "listers/apps/v1beta2",
        "informers/apps/v1beta2",
        "informers/apps",
        "listers/auditregistration/v1alpha1",
        "informers/auditregistration/v1alpha1",
        "informers/auditregistration",
        "listers/autoscaling/v1",
        "informers/autoscaling/v1",
        "listers/autoscaling/v2beta1",
        "informers/autoscaling/v2beta1",
        "listers/autoscaling/v2beta2",
        "informers/autoscaling/v2beta2",
        "informers/autoscaling",
        "listers/batch/v1",
        "informers/batch/v1",
        "listers/batch/v1beta1",
        "informers/batch/v1beta1",
        "listers/batch/v2alpha1",
        "informers/batch/v2alpha1",
        "informers/batch",
        "listers/certificates/v1beta1",
        "informers/certificates/v1beta1",
        "informers/certificates",
        "listers/coordination/v1",
        "informers/coordination/v1",
        "listers/coordination/v1beta1",
        "informers/coordination/v1beta1",
        "informers/coordination",
        "listers/core/v1",
        "informers/core/v1",
        "informers/core",
        "listers/discovery/v1alpha1",
        "informers/discovery/v1alpha1",
        "listers/discovery/v1beta1",
        "informers/discovery/v1beta1",
        "informers/discovery",
        "listers/events/v1beta1",
        "informers/events/v1beta1",
        "informers/events",
        "listers/extensions/v1beta1",
        "informers/extensions/v1beta1",
        "informers/extensions",
        "listers/flowcontrol/v1alpha1",
        "informers/flowcontrol/v1alpha1",
        "informers/flowcontrol",
        "listers/networking/v1",
        "informers/networking/v1",
        "listers/networking/v1beta1",
        "informers/networking/v1beta1",
        "informers/networking",
        "listers/node/v1alpha1",
        "informers/node/v1alpha1",
        "listers/node/v1beta1",
        "informers/node/v1beta1",
        "informers/node",
        "listers/policy/v1beta1",
        "informers/policy/v1beta1",
        "informers/policy",
        "listers/rbac/v1",
        "informers/rbac/v1",
        "listers/rbac/v1alpha1",
        "informers/rbac/v1alpha1",
        "listers/rbac/v1beta1",
        "informers/rbac/v1beta1",
        "informers/rbac",
        "listers/scheduling/v1",
        "informers/scheduling/v1",
        "listers/scheduling/v1alpha1",
        "informers/scheduling/v1alpha1",
        "listers/scheduling/v1beta1",
        "informers/scheduling/v1beta1",
        "informers/scheduling",
        "listers/settings/v1alpha1",
        "informers/settings/v1alpha1",
        "informers/settings",
        "listers/storage/v1",
        "informers/storage/v1",
        "listers/storage/v1alpha1",
        "informers/storage/v1alpha1",
        "listers/storage/v1beta1",
        "informers/storage/v1beta1",
        "informers/storage",
        "informers",
        "metadata",
        "metadata/metadatalister",
        "metadata/metadatainformer",
        "tools/record/util",
        "tools/record",
        "plugin/pkg/client/auth/gcp",
        "util/workqueue",
        "testing",
        "discovery/fake",
        "kubernetes/typed/admissionregistration/v1/fake",
        "kubernetes/typed/admissionregistration/v1beta1/fake",
        "kubernetes/typed/apps/v1/fake",
        "kubernetes/typed/apps/v1beta1/fake",
        "kubernetes/typed/apps/v1beta2/fake",
        "kubernetes/typed/auditregistration/v1alpha1/fake",
        "kubernetes/typed/authentication/v1/fake",
        "kubernetes/typed/authentication/v1beta1/fake",
        "kubernetes/typed/authorization/v1/fake",
        "kubernetes/typed/authorization/v1beta1/fake",
        "kubernetes/typed/autoscaling/v1/fake",
        "kubernetes/typed/autoscaling/v2beta1/fake",
        "kubernetes/typed/autoscaling/v2beta2/fake",
        "kubernetes/typed/batch/v1/fake",
        "kubernetes/typed/batch/v1beta1/fake",
        "kubernetes/typed/batch/v2alpha1/fake",
        "kubernetes/typed/certificates/v1beta1/fake",
        "kubernetes/typed/coordination/v1/fake",
        "kubernetes/typed/coordination/v1beta1/fake",
        "kubernetes/typed/core/v1/fake",
        "kubernetes/typed/discovery/v1alpha1/fake",
        "kubernetes/typed/discovery/v1beta1/fake",
        "kubernetes/typed/events/v1beta1/fake",
        "kubernetes/typed/extensions/v1beta1/fake",
        "kubernetes/typed/flowcontrol/v1alpha1/fake",
        "kubernetes/typed/networking/v1/fake",
        "kubernetes/typed/networking/v1beta1/fake",
        "kubernetes/typed/node/v1alpha1/fake",
        "kubernetes/typed/node/v1beta1/fake",
        "kubernetes/typed/policy/v1beta1/fake",
        "kubernetes/typed/rbac/v1/fake",
        "kubernetes/typed/rbac/v1alpha1/fake",
        "kubernetes/typed/rbac/v1beta1/fake",
        "kubernetes/typed/scheduling/v1/fake",
        "kubernetes/typed/scheduling/v1alpha1/fake",
        "kubernetes/typed/scheduling/v1beta1/fake",
        "kubernetes/typed/settings/v1alpha1/fake",
        "kubernetes/typed/storage/v1/fake",
        "kubernetes/typed/storage/v1alpha1/fake",
        "kubernetes/typed/storage/v1beta1/fake",
        "kubernetes/fake",
    ],
    module = "k8s.io/client-go",
    replace = "k8s.io/client-go",
    sum = "h1:Sm/9AQ415xPAX42JLKbJZnreXFgD2rVfDUDwOTm0gzA=",
    version = "v0.17.5",
    deps = [
        ":github.com_davecgh_go-spew",
        ":github.com_evanphx_json-patch",
        ":github.com_golang_groupcache",
        ":github.com_golang_protobuf",
        ":github.com_googleapis_gnostic",
        ":github.com_gregjones_httpcache",
        ":github.com_imdario_mergo",
        ":github.com_peterbourgon_diskv",
        ":github.com_spf13_pflag",
        ":golang.org_x_crypto",
        ":golang.org_x_net",
        ":golang.org_x_oauth2",
        ":golang.org_x_time",
        ":k8s.io_api",
        ":k8s.io_apimachinery",
        ":k8s.io_klog",
        ":k8s.io_utils",
    ],
)

go_get(
    name = "k8s.io_cluster-bootstrap",
    download_deps = [],
    install = [
        "token/api",
        "token/util",
        "util/secrets",
    ],
    module = "k8s.io/cluster-bootstrap",
    replace = "k8s.io/cluster-bootstrap",
    sum = "h1:LjdKuGxrDCdCRZBXp94R5WQC3XD+3TQRv0AY3yMTDBU=",
    version = "v0.17.5",
    deps = [
        ":k8s.io_api",
        ":k8s.io_apimachinery",
        ":k8s.io_klog",
    ],
)

go_get(
    name = "k8s.io_component-base",
    download_deps = [],
    install = [
        "featuregate",
        "version",
        "config",
        "config/v1alpha1",
        "cli/flag",
    ],
    module = "k8s.io/component-base",
    replace = "k8s.io/component-base",
    sum = "h1:f4QKFRH1OIuWbpWwDm+vGvFQmrXmAbtPF8PREdtkIGE=",
    version = "v0.17.5",
    deps = [
        ":github.com_spf13_pflag",
        ":k8s.io_apimachinery",
        ":k8s.io_klog",
        ":k8s.io_utils",
    ],
)

go_get(
    name = "k8s.io_helm",
    download_deps = [],
    install = [
        "pkg/ignore",
        "pkg/proto/hapi/chart",
        "pkg/proto/hapi/version",
        "pkg/sympath",
        "pkg/version",
        "pkg/chartutil",
        "pkg/proto/hapi/release",
        "pkg/releaseutil",
        "pkg/manifest",
        "pkg/proto/hapi/services",
        "pkg/engine",
        "pkg/renderutil",
        "pkg/storage/errors",
        "pkg/helm",
        "pkg/kube",
        "pkg/storage/driver",
        "pkg/storage",
        "pkg/tiller/environment",
        "pkg/helm/portforwarder",
        "pkg/helm/helmpath",
        "pkg/helm/environment",
        "pkg/plugin",
        "pkg/urlutil",
        "pkg/tlsutil",
        "pkg/getter",
        "pkg/provenance",
        "pkg/repo",
        "pkg/strvals",
        "cmd/helm/installer",
        "internal/third_party/dep/fs",
        "pkg/resolver",
        "pkg/downloader",
    ],
    module = "k8s.io/helm",
    replace = "github.com/banzaicloud/helm",
    sum = "h1:JOyw3e97+ArUdmzLNMuu7RukJuGNs9FLyWtiMAyztiI=",
    version = "v2.7.1-0.20200228123321-c4355aab74fc+incompatible",
    deps = [
        ":github.com_BurntSushi_toml",
        ":github.com_Masterminds_semver",
        ":github.com_Masterminds_sprig",
        ":github.com_cyphar_filepath-securejoin",
        ":github.com_evanphx_json-patch",
        ":github.com_ghodss_yaml",
        ":github.com_gobwas_glob",
        ":github.com_golang_protobuf",
        ":github.com_jmoiron_sqlx",
        ":github.com_lib_pq",
        ":github.com_pkg_errors",
        ":github.com_rubenv_sql-migrate",
        ":github.com_spf13_pflag",
        ":golang.org_x_crypto",
        ":golang.org_x_net",
        ":google.golang.org_grpc",
        ":k8s.io_api",
        ":k8s.io_apiextensions-apiserver",
        ":k8s.io_apimachinery",
        ":k8s.io_cli-runtime",
        ":k8s.io_client-go",
        ":k8s.io_kubectl",
        ":k8s.io_kubernetes",
    ],
)

go_get(
    name = "k8s.io_klog",
    download_deps = [],
    install = [""],
    module = "k8s.io/klog",
    sum = "h1:Pt+yjF5aB1xDSVbau4VsWe+dQNzA0qv1LlXdC2dF6Q8=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "k8s.io_kube-openapi",
    download_deps = [],
    install = [
        "pkg/util/proto",
        "pkg/common",
        "pkg/util/proto/validation",
    ],
    module = "k8s.io/kube-openapi",
    sum = "h1:jocF7XFucw2pEiv2wS7wk2FRFCjDFGV1oa4TMs0SAT0=",
    version = "v0.0.0-20200316234421-82d701f24f9d",
    deps = [
        ":github.com_emicklei_go-restful",
        ":github.com_go-openapi_spec",
        ":github.com_googleapis_gnostic",
        ":gopkg.in_yaml.v2",
    ],
)

go_get(
    name = "k8s.io_kube-proxy",
    download_deps = [],
    install = ["config/v1alpha1"],
    module = "k8s.io/kube-proxy",
    replace = "k8s.io/kube-proxy",
    sum = "h1:K+1Hi1BgeELDPvmBihOdYAuDUwqnQd68pKstcDqW2kY=",
    version = "v0.17.5",
    deps = [
        ":k8s.io_apimachinery",
        ":k8s.io_component-base",
    ],
)

go_get(
    name = "k8s.io_kubectl",
    download_deps = [],
    install = [
        "pkg/scheme",
        "pkg/util/openapi",
        "pkg/util/openapi/validation",
        "pkg/util/interrupt",
        "pkg/util/term",
        "pkg/util/templates",
        "pkg/validation",
        "pkg/cmd/util",
        "pkg/rawhttp",
        "pkg/generated",
        "pkg/util/i18n",
        "pkg/cmd/get",
    ],
    module = "k8s.io/kubectl",
    replace = "k8s.io/kubectl",
    sum = "h1:ZPHJzD5Vvs0jk1Rdx23npBm3z5PgVEER+b/VE8fxbKU=",
    version = "v0.17.5",
    deps = [
        ":github.com_MakeNowJust_heredoc",
        ":github.com_chai2010_gettext-go",
        ":github.com_docker_docker",
        ":github.com_evanphx_json-patch",
        ":github.com_exponent-io_jsonpath",
        ":github.com_go-openapi_spec",
        ":github.com_googleapis_gnostic",
        ":github.com_liggitt_tabwriter",
        ":github.com_mitchellh_go-wordwrap",
        ":github.com_russross_blackfriday",
        ":github.com_spf13_cobra",
        ":github.com_spf13_pflag",
        ":golang.org_x_sys",
        ":gopkg.in_yaml.v2",
        ":k8s.io_api",
        ":k8s.io_apimachinery",
        ":k8s.io_cli-runtime",
        ":k8s.io_client-go",
        ":k8s.io_component-base",
        ":k8s.io_klog",
        ":k8s.io_kube-openapi",
        ":k8s.io_utils",
        ":vbom.ml_util",
    ],
)

go_get(
    name = "k8s.io_kubelet",
    download_deps = [],
    install = ["config/v1beta1"],
    module = "k8s.io/kubelet",
    replace = "k8s.io/kubelet",
    sum = "h1:2P+HWjbe3fjB0f5JUhbYrqRYp3CjhmfgxBF9NSV+i8k=",
    version = "v0.17.5",
    deps = [
        ":k8s.io_api",
        ":k8s.io_apimachinery",
    ],
)

go_get(
    name = "k8s.io_kubernetes",
    download_deps = [],
    install = [
        "pkg/apis/core",
        "pkg/apis/storage",
        "pkg/features",
        "pkg/apis/storage/util",
        "pkg/api/legacyscheme",
        "pkg/api/v1/pod",
        "pkg/apis/autoscaling",
        "pkg/apis/apps",
        "pkg/util/parsers",
        "pkg/apis/core/v1",
        "pkg/apis/core/install",
        "pkg/api/service",
        "pkg/apis/core/helper",
        "pkg/fieldpath",
        "pkg/apis/core/pods",
        "pkg/apis/core/v1/helper",
        "pkg/capabilities",
        "pkg/master/ports",
        "pkg/apis/scheduling",
        "pkg/kubelet/types",
        "pkg/security/apparmor",
        "pkg/apis/core/validation",
        "pkg/serviceaccount",
        "pkg/util/hash",
        "pkg/util/taints",
        "pkg/controller",
        "pkg/util/labels",
        "pkg/controller/deployment/util",
        "cmd/kubeadm/app/apis/kubeadm",
        "pkg/api/v1/resource",
        "pkg/scheduler/algorithm/priorities/util",
        "pkg/scheduler/nodeinfo",
    ],
    module = "k8s.io/kubernetes",
    sum = "h1:ZPZDhS4wRIPSKahglFHK5Q8wcUH/Q2QE5AbopkuLYyU=",
    version = "v1.17.5",
    deps = [
        ":github.com_davecgh_go-spew",
        ":github.com_docker_distribution",
        ":github.com_golang_groupcache",
        ":github.com_pkg_errors",
        ":golang.org_x_oauth2",
        ":gopkg.in_square_go-jose.v2",
        ":k8s.io_api",
        ":k8s.io_apiextensions-apiserver",
        ":k8s.io_apimachinery",
        ":k8s.io_apiserver",
        ":k8s.io_client-go",
        ":k8s.io_cluster-bootstrap",
        ":k8s.io_component-base",
        ":k8s.io_klog",
        ":k8s.io_kube-proxy",
        ":k8s.io_kubelet",
        ":k8s.io_utils",
    ],
)

go_get(
    name = "k8s.io_utils",
    download_deps = [],
    install = [
        "integer",
        "pointer",
        "buffer",
        "trace",
        "exec",
        "net",
        "path",
    ],
    module = "k8s.io/utils",
    sum = "h1:GiPwtSzdP43eI1hpPCbROQCCIgCuiMMNF8YUVLF3vJo=",
    version = "v0.0.0-20191114184206-e782cd3c129f",
    deps = [":k8s.io_klog"],
)

go_get(
    name = "logur.dev_adapter_logrus",
    download_deps = [],
    install = [""],
    module = "logur.dev/adapter/logrus",
    sum = "h1:qBw5g87m++zoFOq6K30J+UnQC7bZrPAwpmoF3G7Oxo8=",
    version = "v0.4.1",
    deps = [
        ":github.com_sirupsen_logrus",
        ":logur.dev_logur",
    ],
)

go_get(
    name = "logur.dev_adapter_zap",
    download_deps = [],
    install = [
        "internal/keyvals",
        "",
    ],
    module = "logur.dev/adapter/zap",
    sum = "h1:lZFwPNU3SQamoxejhkU2v/TTt/vC/eeYQBO9MIPKXmc=",
    version = "v0.4.1",
    deps = [
        ":go.uber.org_zap",
        ":logur.dev_logur",
    ],
)

go_get(
    name = "logur.dev_integration_watermill",
    download_deps = [],
    install = [""],
    module = "logur.dev/integration/watermill",
    sum = "h1:y9MhXxJB7Igm712L24mSEUvEdsfa8KQ7ZTT4vObROUg=",
    version = "v0.4.2",
    deps = [
        ":github.com_ThreeDotsLabs_watermill",
        ":logur.dev_logur",
    ],
)

go_get(
    name = "logur.dev_integration_zap",
    download_deps = [],
    install = [""],
    module = "logur.dev/integration/zap",
    sum = "h1:fz3Bjuqn0mcxfas7ff12rS+frH1KnBsZI3ye6+8e5T8=",
    version = "v0.3.2",
    deps = [
        ":go.uber.org_zap",
        ":logur.dev_logur",
    ],
)

go_get(
    name = "logur.dev_logur",
    download_deps = [],
    install = [
        "",
        "conformance",
        "logtesting",
    ],
    module = "logur.dev/logur",
    sum = "h1:q4MxivaiTXiDHrQyeCH5WkwBLUrd6rM2lZlyztYvi4o=",
    version = "v0.16.2",
    deps = [],
)

go_get(
    name = "sigs.k8s.io_controller-runtime",
    download_deps = [],
    install = [
        "pkg/client/apiutil",
        "pkg/client",
        "pkg/log",
        "pkg/internal/log",
        "pkg/client/config",
        "pkg/scheme",
        "pkg/runtime/scheme",
    ],
    module = "sigs.k8s.io/controller-runtime",
    sum = "h1:pyXbUfoTo+HA3jeIfr0vgi+1WtmNh0CwlcnQGLXwsSw=",
    version = "v0.5.2",
    deps = [
        ":github.com_evanphx_json-patch",
        ":github.com_go-logr_logr",
        ":golang.org_x_time",
        ":k8s.io_apimachinery",
        ":k8s.io_client-go",
    ],
)

go_get(
    name = "sigs.k8s.io_kubefed",
    download_deps = [],
    install = [
        "pkg/apis/core/common",
        "pkg/apis/core/v1beta1",
        "pkg/apis/core/v1alpha1",
        "pkg/apis/multiclusterdns/v1alpha1",
        "pkg/apis/scheduling/v1alpha1",
        "pkg/apis",
        "pkg/client/generic/scheme",
        "pkg/client/generic",
        "pkg/controller/util",
        "pkg/apis/core/typeconfig",
        "pkg/kubefedctl/options",
        "pkg/kubefedctl/util",
        "pkg/kubefedctl/enable",
        "pkg/kubefedctl/federate",
        "pkg/kubefedctl/orphaning",
        "pkg/version",
        "pkg/kubefedctl",
    ],
    module = "sigs.k8s.io/kubefed",
    sum = "h1:nzaQ4HDReHLECXMv7iszHBLx3+GO3/Iwlw7dkS71qCw=",
    version = "v0.2.0-alpha.1",
    deps = [
        ":github.com_evanphx_json-patch",
        ":github.com_ghodss_yaml",
        ":github.com_json-iterator_go",
        ":github.com_pkg_errors",
        ":github.com_spf13_cobra",
        ":github.com_spf13_pflag",
        ":k8s.io_api",
        ":k8s.io_apiextensions-apiserver",
        ":k8s.io_apimachinery",
        ":k8s.io_client-go",
        ":k8s.io_component-base",
        ":k8s.io_klog",
        ":k8s.io_kube-openapi",
        ":k8s.io_kubectl",
        ":sigs.k8s.io_controller-runtime",
        ":sigs.k8s.io_yaml",
    ],
)

go_get(
    name = "sigs.k8s.io_kustomize",
    download_deps = [],
    install = [
        "pkg/gvk",
        "pkg/image",
        "pkg/patch",
        "pkg/types",
        "pkg/ifc",
        "pkg/internal/error",
        "pkg/resid",
        "pkg/resource",
        "pkg/resmap",
        "pkg/expansion",
        "pkg/transformers/config/defaultconfig",
        "pkg/transformers/config",
        "pkg/transformers",
        "pkg/ifc/transformer",
        "pkg/factory",
        "pkg/constants",
        "pkg/fs",
        "pkg/git",
        "pkg/loader",
        "pkg/patch/transformer",
        "pkg/target",
        "pkg/commands/build",
    ],
    module = "sigs.k8s.io/kustomize",
    sum = "h1:JUufWFNlI44MdtnjUqVnvh29rR37PQFzPbLXqhyOyX0=",
    version = "v2.0.3+incompatible",
    deps = [
        ":github.com_evanphx_json-patch",
        ":github.com_ghodss_yaml",
        ":github.com_go-openapi_spec",
        ":github.com_pkg_errors",
        ":github.com_spf13_cobra",
        ":k8s.io_kube-openapi",
    ],
)

go_get(
    name = "sigs.k8s.io_testing_frameworks",
    download_deps = [],
    install = [
        "integration/addr",
        "integration/internal",
        "integration",
    ],
    module = "sigs.k8s.io/testing_frameworks",
    sum = "h1:vK0+tvjF0BZ/RYFeZ1E6BYBwHJJXhjuZ3TdsEKH+UQM=",
    version = "v0.1.2",
    deps = [":github.com_onsi_gomega"],
)

go_get(
    name = "sigs.k8s.io_yaml",
    download_deps = [],
    install = [""],
    module = "sigs.k8s.io/yaml",
    sum = "h1:4A07+ZFc2wgJwo8YNlQpr1rVlgUDlxXHhPJciaPY5gs=",
    version = "v1.1.0",
    deps = [":gopkg.in_yaml.v2"],
)

go_get(
    name = "vbom.ml_util",
    download_deps = [],
    install = ["sortorder"],
    module = "vbom.ml/util",
    sum = "h1:MksmcCZQWAQJCTA5T0jgI/0sJ51AVm4Z41MrmfczEoc=",
    version = "v0.0.0-20160121211510-db5cfe13f5cc",
    deps = [],
)
