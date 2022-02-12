module github.com/argoproj/argo-events

go 1.17

retract (
	v1.15.1 // Contains retractions only.
	v1.15.0 // Published accidentally.
)

require (
	cloud.google.com/go/compute v1.1.0
	cloud.google.com/go/pubsub v1.3.1
	github.com/Azure/azure-event-hubs-go/v3 v3.3.7
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible
	github.com/Masterminds/sprig/v3 v3.2.0
	github.com/Shopify/sarama v1.31.1
	github.com/ahmetb/gen-crd-api-reference-docs v0.3.0
	github.com/antonmedv/expr v1.8.8
	github.com/apache/openwhisk-client-go v0.0.0-20190915054138-716c6f973eb2
	github.com/apache/pulsar-client-go v0.1.1
	github.com/argoproj/pkg v0.11.0
	github.com/aws/aws-sdk-go v1.42.50
	github.com/blushft/go-diagrams v0.0.0-20201006005127-c78c821223d9
	github.com/bradleyfalzon/ghinstallation/v2 v2.0.3
	github.com/cloudevents/sdk-go/v2 v2.1.0
	github.com/colinmarc/hdfs v1.1.4-0.20180802165501-48eb8d6c34a9
	github.com/eclipse/paho.mqtt.golang v1.3.5
	github.com/emitter-io/go/v2 v2.0.9
	github.com/fsnotify/fsnotify v1.5.1
	github.com/gavv/httpexpect/v2 v2.2.0
	github.com/gfleury/go-bitbucket-v1 v0.0.0-20210707202713-7d616f7c18ac
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-git/go-git/v5 v5.4.2
	github.com/go-openapi/inflect v0.19.0
	github.com/go-openapi/spec v0.20.4
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-resty/resty/v2 v2.7.0
	github.com/go-swagger/go-swagger v0.29.0
	github.com/gobwas/glob v0.2.4-0.20181002190808-e7a84e9525fe
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.7
	github.com/google/go-github/v31 v31.0.0
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/imdario/mergo v0.3.12
	github.com/itchyny/gojq v0.12.6
	github.com/joncalhoun/qson v0.0.0-20200422171543-84433dcd3da0
	github.com/ktrysmt/go-bitbucket v0.9.32
	github.com/minio/minio-go/v7 v7.0.15
	github.com/mitchellh/mapstructure v1.4.3
	github.com/nats-io/go-nats v1.7.2
	github.com/nats-io/graft v0.0.0-20200605173148-348798afea05
	github.com/nats-io/nats.go v1.13.1-0.20220121202836-972a071d373d
	github.com/nats-io/stan.go v0.10.2
	github.com/nsqio/go-nsq v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.1
	github.com/radovskyb/watcher v1.0.7
	github.com/robfig/cron/v3 v3.0.1
	github.com/slack-go/slack v0.10.1
	github.com/smartystreets/goconvey v1.7.2
	github.com/spf13/cobra v1.3.0
	github.com/spf13/viper v1.10.1
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/testify v1.7.0
	github.com/stripe/stripe-go v70.15.0+incompatible
	github.com/tidwall/gjson v1.13.0
	github.com/tidwall/sjson v1.2.4
	github.com/xanzy/go-gitlab v0.54.4
	github.com/yuin/gopher-lua v0.0.0-20210529063254-f4c35e4016d9
	go.uber.org/ratelimit v0.2.0
	go.uber.org/zap v1.20.0
	golang.org/x/crypto v0.0.0-20220128200615-198e4374d7ed
	google.golang.org/api v0.65.0
	google.golang.org/grpc v1.43.0
	gopkg.in/jcmturner/gokrb5.v5 v5.3.0
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.21.6
	k8s.io/code-generator v0.21.6
	k8s.io/gengo v0.0.0-20211115164449-b448ea381d54
	k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
	sigs.k8s.io/controller-runtime v0.9.7
	sigs.k8s.io/controller-tools v0.7.0
	sigs.k8s.io/yaml v1.2.0
)

require (
	github.com/felixge/httpsnoop v1.0.2 // indirect
	github.com/hashicorp/go-hclog v1.1.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/oliveagle/jsonpath v0.0.0-20180606110733-2e52cf6e6852 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/stringprep v1.0.2 // indirect
)

require (
	cloud.google.com/go v0.100.2 // indirect
	cloud.google.com/go/iam v0.1.1 // indirect
	cloud.google.com/go/kms v1.1.0 // indirect
	github.com/Azure/azure-amqp-common-go/v3 v3.1.0 // indirect
	github.com/Azure/azure-sdk-for-go v52.6.0+incompatible // indirect
	github.com/Azure/go-amqp v0.13.6 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.13 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20210428141323-04723f9f07d7 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/ajg/form v1.5.1 // indirect
	github.com/andres-erbsen/clock v0.0.0-20160526145045-9e14626cd129 // indirect
	github.com/ardielle/ardielle-go v1.5.2 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/awalterschulze/gographviz v0.0.0-20200901124122-0eecad45bd71 // indirect
	github.com/benbjohnson/clock v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cloudfoundry/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/devigned/tab v0.1.1 // indirect
	github.com/doublerebel/bellows v0.0.0-20160303004610-f177d92a03d3
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/eapache/go-resiliency v1.2.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/emicklei/go-restful v2.12.0+incompatible // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/evanphx/json-patch v4.11.0+incompatible // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/fatih/structs v1.0.0 // indirect
	github.com/form3tech-oss/jwt-go v3.2.2+incompatible // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-git/go-billy/v5 v5.3.1 // indirect
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/go-openapi/analysis v0.21.2 // indirect
	github.com/go-openapi/errors v0.20.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/loads v0.21.0 // indirect
	github.com/go-openapi/runtime v0.21.1 // indirect
	github.com/go-openapi/strfmt v0.21.1 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/go-openapi/validate v0.20.3 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gobuffalo/flect v0.2.3 // indirect
	github.com/golang-jwt/jwt/v4 v4.0.0 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-github/v39 v39.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/gax-go/v2 v2.1.1 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.8 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e // indirect
	github.com/huandu/xstrings v1.3.1 // indirect
	github.com/imkira/go-interpol v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/itchyny/timefmt-go v0.1.3 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.2 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jessevdk/go-flags v1.5.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/kevinburke/ssh_config v0.0.0-20201106050909-4977a11b4351 // indirect
	github.com/klauspost/compress v1.14.2 // indirect
	github.com/klauspost/cpuid v1.3.1 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lightstep/tracecontext.go v0.0.0-20181129014701-1757c391b1ac // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/minio/md5-simd v1.1.0 // indirect
	github.com/minio/sha256-simd v0.1.1 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nats-io/gnatsd v1.4.1 // indirect
	github.com/nats-io/nats-server/v2 v2.7.2 // indirect
	github.com/nats-io/nats-streaming-server v0.24.1 // indirect
	github.com/nats-io/nkeys v0.3.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nicksnyder/go-i18n v1.10.1-0.20190510212457-b280125b035a // indirect
	github.com/pelletier/go-toml v1.9.4 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	github.com/rs/xid v1.2.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/smartystreets/assertions v1.2.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.8.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/toqueteos/webbrowser v1.2.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.9.0 // indirect
	github.com/valyala/gozstd v1.7.0 // indirect
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	github.com/xdg-go/scram v1.1.0
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.1.0 // indirect
	github.com/yahoo/athenz v1.8.55 // indirect
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0 // indirect
	github.com/yudai/gojsondiff v1.0.0 // indirect
	github.com/yudai/golcs v0.0.0-20170316035057-ecda9a501e82 // indirect
	go.mongodb.org/mongo-driver v1.8.2 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20211116232009-f0f3c7e86c11 // indirect
	golang.org/x/tools v0.1.8 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220111164026-67b88f271998 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.66.3 // indirect
	gopkg.in/jcmturner/aescts.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/dnsutils.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/goidentity.v2 v2.0.0 // indirect
	gopkg.in/jcmturner/rpc.v0 v0.0.2 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/apiextensions-apiserver v0.22.2 // indirect
	k8s.io/component-base v0.21.6 // indirect
	k8s.io/klog v0.3.0 // indirect
	k8s.io/klog/v2 v2.9.0 // indirect
	k8s.io/utils v0.0.0-20210802155522-efc7438f0176 // indirect
	moul.io/http2curl v1.0.1-0.20190925090545-5cd742060b0e // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
)

replace k8s.io/api => k8s.io/api v0.21.6

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.6

replace k8s.io/apimachinery => k8s.io/apimachinery v0.21.7-rc.0

replace k8s.io/apiserver => k8s.io/apiserver v0.21.6

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.6

replace k8s.io/client-go => k8s.io/client-go v0.21.6

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.21.6

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.6

replace k8s.io/component-base => k8s.io/component-base v0.21.6

replace k8s.io/cri-api => k8s.io/cri-api v0.21.8-rc.0

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.21.6

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.6

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.21.6

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.6

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.6

replace k8s.io/kubectl => k8s.io/kubectl v0.21.6

replace k8s.io/kubelet => k8s.io/kubelet v0.21.6

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.6

replace k8s.io/metrics => k8s.io/metrics v0.21.6

replace k8s.io/node-api => k8s.io/node-api v0.20.4

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.6

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.21.6

replace k8s.io/sample-controller => k8s.io/sample-controller v0.21.6

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible

replace k8s.io/code-generator => k8s.io/code-generator v0.21.7-rc.0

replace k8s.io/controller-manager => k8s.io/controller-manager v0.21.6

replace k8s.io/component-helpers => k8s.io/component-helpers v0.21.6

replace k8s.io/mount-utils => k8s.io/mount-utils v0.21.7-rc.0
