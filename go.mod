module github.com/argoproj/argo-events

go 1.13

require (
	cloud.google.com/go v0.56.1-0.20200408183257-fc3b2ad6791b // indirect
	cloud.google.com/go/pubsub v1.2.0
	github.com/Azure/azure-event-hubs-go/v3 v3.2.0
	github.com/Azure/go-amqp v0.12.7 // indirect
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/Shopify/sarama v1.26.1
	github.com/ahmetb/gen-crd-api-reference-docs v0.2.0
	github.com/apache/openwhisk-client-go v0.0.0-20190915054138-716c6f973eb2
	github.com/argoproj/argo v2.5.2+incompatible
	github.com/argoproj/argo-cd v1.5.1
	github.com/argoproj/argo-rollouts v0.7.2
	github.com/argoproj/pkg v0.0.0-20200319004004-f46beff7cd54 // indirect
	github.com/aws/aws-sdk-go v1.30.7
	github.com/cloudevents/sdk-go v1.1.2
	github.com/cloudfoundry/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21 // indirect
	github.com/colinmarc/hdfs v1.1.4-0.20180802165501-48eb8d6c34a9
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/emicklei/go-restful v2.12.0+incompatible // indirect
	github.com/emitter-io/go/v2 v2.0.5
	github.com/fatih/color v1.9.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.7
	github.com/go-openapi/swag v0.19.8 // indirect
	github.com/go-redis/redis v6.15.7+incompatible
	github.com/go-resty/resty/v2 v2.3.0
	github.com/gobwas/glob v0.2.4-0.20181002190808-e7a84e9525fe
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.3.5
	github.com/google/go-cmp v0.4.0
	github.com/google/go-github/v31 v31.0.0
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.4.0 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.5 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e // indirect
	github.com/huandu/xstrings v1.3.0 // indirect
	github.com/imdario/mergo v0.3.9
	github.com/joncalhoun/qson v0.0.0-20200214190643-0e0c0e4268e7
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/k0kubun/colorstring v0.0.0-20150214042306-9440f1994b88 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/klauspost/compress v1.10.4 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/mailru/easyjson v0.7.1 // indirect
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/minio/minio-go v1.0.1-0.20190523192347-c6c2912aa552
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.3.0
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/nats-io/gnatsd v1.4.1 // indirect
	github.com/nats-io/go-nats v1.7.2
	github.com/nats-io/nkeys v0.1.4 // indirect
	github.com/nicksnyder/go-i18n v1.10.1-0.20190510212457-b280125b035a // indirect
	github.com/nlopes/slack v0.6.1-0.20200219171353-c05e07b0a5de
	github.com/nsqio/go-nsq v1.0.8
	github.com/pelletier/go-toml v1.7.0 // indirect
	github.com/pierrec/lz4 v2.5.0+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/radovskyb/watcher v1.0.7
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/robfig/cron v1.2.0
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/sirupsen/logrus v1.5.0
	github.com/smartystreets/assertions v0.0.0-20190401211740-f487f9de1cd3 // indirect
	github.com/smartystreets/goconvey v1.6.4
	github.com/streadway/amqp v0.0.0-20200108173154-1c71cc93ed71
	github.com/stretchr/testify v1.5.1
	github.com/stripe/stripe-go v70.11.0+incompatible
	github.com/tidwall/gjson v1.6.0
	github.com/tidwall/sjson v1.1.1
	github.com/xanzy/go-gitlab v0.31.0
	github.com/yudai/gojsondiff v1.0.0 // indirect
	github.com/yudai/golcs v0.0.0-20170316035057-ecda9a501e82 // indirect
	github.com/yudai/pp v2.0.1+incompatible // indirect
	go.uber.org/zap v1.14.1
	golang.org/x/crypto v0.0.0-20200429183012-4b2356b1ed79
	golang.org/x/sys v0.0.0-20200409092240-59c9f1ba88fa // indirect
	google.golang.org/api v0.21.0
	google.golang.org/grpc v1.28.1
	gopkg.in/ini.v1 v1.55.0 // indirect
	gopkg.in/jcmturner/goidentity.v2 v2.0.0 // indirect
	gopkg.in/jcmturner/gokrb5.v5 v5.3.0
	gopkg.in/jcmturner/rpc.v0 v0.0.2 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	k8s.io/api v0.17.5
	k8s.io/apimachinery v0.17.5
	k8s.io/client-go v0.17.5
	k8s.io/code-generator v0.17.5
	k8s.io/gengo v0.0.0-20190822140433-26a664648505
	k8s.io/kube-openapi v0.0.0-20200316234421-82d701f24f9d
	k8s.io/kubernetes v1.17.5 // indirect
	k8s.io/utils v0.0.0-20200327001022-6496210b90e8 // indirect
	sigs.k8s.io/controller-runtime v0.5.4
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace k8s.io/api => k8s.io/api v0.17.5

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.5

replace k8s.io/apimachinery => k8s.io/apimachinery v0.17.6-beta.0

replace k8s.io/apiserver => k8s.io/apiserver v0.17.5

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.5

replace k8s.io/client-go => k8s.io/client-go v0.17.5

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.5

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.5

replace k8s.io/component-base => k8s.io/component-base v0.17.5

replace k8s.io/cri-api => k8s.io/cri-api v0.17.6-beta.0

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.5

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.5

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.5

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.5

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.5

replace k8s.io/kubectl => k8s.io/kubectl v0.17.5

replace k8s.io/kubelet => k8s.io/kubelet v0.17.5

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.5

replace k8s.io/metrics => k8s.io/metrics v0.17.5

replace k8s.io/node-api => k8s.io/node-api v0.17.5

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.5

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.17.5

replace k8s.io/sample-controller => k8s.io/sample-controller v0.17.5

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.3+incompatible

replace k8s.io/code-generator => k8s.io/code-generator v0.17.5
