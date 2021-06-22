module github.com/argoproj/argo-events

go 1.15

require (
	cloud.google.com/go v0.54.0
	cloud.google.com/go/pubsub v1.2.0
	github.com/Azure/azure-amqp-common-go/v3 v3.1.0 // indirect
	github.com/Azure/azure-event-hubs-go/v3 v3.3.7
	github.com/Azure/azure-sdk-for-go v52.6.0+incompatible // indirect
	github.com/Azure/go-amqp v0.13.6 // indirect
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible
	github.com/Masterminds/sprig/v3 v3.2.0
	github.com/Shopify/sarama v1.26.1
	github.com/ahmetb/gen-crd-api-reference-docs v0.2.0
	github.com/antonmedv/expr v1.8.8
	github.com/apache/openwhisk-client-go v0.0.0-20190915054138-716c6f973eb2
	github.com/apache/pulsar-client-go v0.1.1
	github.com/argoproj/pkg v0.9.0
	github.com/aws/aws-sdk-go v1.33.16
	github.com/blushft/go-diagrams v0.0.0-20201006005127-c78c821223d9
	github.com/cloudevents/sdk-go/v2 v2.1.0
	github.com/cloudfoundry/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21 // indirect
	github.com/colinmarc/hdfs v1.1.4-0.20180802165501-48eb8d6c34a9
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/emicklei/go-restful v2.12.0+incompatible // indirect
	github.com/emitter-io/go/v2 v2.0.9
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gavv/httpexpect/v2 v2.2.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-git/go-git/v5 v5.3.0
	github.com/go-openapi/inflect v0.19.0
	github.com/go-openapi/spec v0.20.2
	github.com/go-redis/redis v6.15.8+incompatible
	github.com/go-resty/resty/v2 v2.3.0
	github.com/go-swagger/go-swagger v0.25.0
	github.com/gobwas/glob v0.2.4-0.20181002190808-e7a84e9525fe
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.6
	github.com/google/go-github/v31 v31.0.0
	github.com/google/uuid v1.1.2
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/gorilla/mux v1.7.3
	github.com/grpc-ecosystem/grpc-gateway v1.14.6
	github.com/hokaccha/go-prettyjson v0.0.0-20190818114111-108c894c2c0e // indirect
	github.com/imdario/mergo v0.3.12
	github.com/joncalhoun/qson v0.0.0-20200422171543-84433dcd3da0
	github.com/minio/minio-go v1.0.1-0.20190523192347-c6c2912aa552
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/nats-io/gnatsd v1.4.1 // indirect
	github.com/nats-io/go-nats v1.7.2
	github.com/nats-io/graft v0.0.0-20200605173148-348798afea05
	github.com/nats-io/nats-streaming-server v0.17.0 // indirect
	github.com/nats-io/nats.go v1.10.0
	github.com/nats-io/stan.go v0.6.0
	github.com/nicksnyder/go-i18n v1.10.1-0.20190510212457-b280125b035a // indirect
	github.com/nsqio/go-nsq v1.0.8
	github.com/pierrec/lz4 v2.5.0+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/radovskyb/watcher v1.0.7
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/robfig/cron v1.2.0
	github.com/slack-go/slack v0.7.4
	github.com/smartystreets/assertions v0.0.0-20190401211740-f487f9de1cd3 // indirect
	github.com/smartystreets/goconvey v1.6.4
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.0
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/testify v1.7.0
	github.com/stripe/stripe-go v70.15.0+incompatible
	github.com/tidwall/gjson v1.7.5
	github.com/tidwall/sjson v1.1.1
	github.com/xanzy/go-gitlab v0.33.0
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	google.golang.org/api v0.20.0
	google.golang.org/grpc v1.29.1
	gopkg.in/jcmturner/goidentity.v2 v2.0.0 // indirect
	gopkg.in/jcmturner/gokrb5.v5 v5.3.0
	gopkg.in/jcmturner/rpc.v0 v0.0.2 // indirect
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	k8s.io/code-generator v0.20.4
	k8s.io/gengo v0.0.0-20201113003025-83324d819ded
	k8s.io/klog v0.3.0 // indirect
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	sigs.k8s.io/controller-runtime v0.7.0
	sigs.k8s.io/controller-tools v0.6.0
	sigs.k8s.io/yaml v1.2.0
)

replace k8s.io/api => k8s.io/api v0.20.4

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.4

replace k8s.io/apimachinery => k8s.io/apimachinery v0.20.4-rc.0

replace k8s.io/apiserver => k8s.io/apiserver v0.20.4

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.4

replace k8s.io/client-go => k8s.io/client-go v0.20.4

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.20.4

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.20.4

replace k8s.io/component-base => k8s.io/component-base v0.20.4

replace k8s.io/cri-api => k8s.io/cri-api v0.20.4-rc.0

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.20.4

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.20.4

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.20.4

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.20.4

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.20.4

replace k8s.io/kubectl => k8s.io/kubectl v0.20.4

replace k8s.io/kubelet => k8s.io/kubelet v0.20.4

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.20.4

replace k8s.io/metrics => k8s.io/metrics v0.20.4

replace k8s.io/node-api => k8s.io/node-api v0.20.4

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.20.4

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.20.4

replace k8s.io/sample-controller => k8s.io/sample-controller v0.20.4

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible

replace k8s.io/code-generator => k8s.io/code-generator v0.20.4-rc.0

replace k8s.io/controller-manager => k8s.io/controller-manager v0.20.4-rc.0
