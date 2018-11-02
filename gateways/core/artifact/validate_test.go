package artifact

import (
	"testing"
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
)

var (
	configKey = "testConfig"
	configValue = `
s3EventConfig:
    bucket: input
    endpoint: minio-service.argo-events:9000
    event: s3:ObjectCreated:Put
    filter:
        prefix: ""
        suffix: ""
insecure: true
accessKey:
    key: accesskey
    name: artifacts-minio
secretKey:
    key: secretkey
    name: artifacts-minio
`
)

func getConfigContext() *gateways.ConfigContext {
	return &gateways.ConfigContext{
		StopCh: make(chan struct{}),
		Data: &gateways.ConfigData{
			Src: configKey,
		},
	}
}

func TestS3ConfigExecutor_Validate(t *testing.T) {
	i, err := gateways.ParseGatewayConfig(configValue)
	assert.Nil(t, err)
	s3Config := &S3ConfigExecutor{}
	ctx := getConfigContext()
	ctx.Data.Config = i
	err = s3Config.Validate(ctx)
	assert.Nil(t, err)

	badConfig := `
s3EventConfig:
    bucket: input
    endpoint: minio-service.argo-events:9000
    event: s3:ObjectCreated:Put
    filter:
        prefix: ""
        suffix: ""
insecure: true
`

     ctx.Data.Config = badConfig

     err = s3Config.Validate(ctx)
     assert.NotNil(t, err)
}
