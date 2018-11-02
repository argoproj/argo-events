package artifact

import (
	"encoding/json"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	sv1alphav1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/store"
)

// StartConfig runs a configuration
func (s3ce *S3ConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer s3ce.GatewayConfig.GatewayCleanup(config, &errMessage, err)

	s3ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("starting configuration...")
	artifact := config.Data.Config.(*common.S3Artifact)
	s3ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *artifact).Msg("artifact configuration")

	go s3ce.listenToEvents(artifact, config)

	for {
		select {
		case <-s3ce.StartChan:
			s3ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running")
			config.Active = true

		case data := <-s3ce.DataCh:
			s3ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway-processor")
			s3ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src: config.Data.Src,
				Payload: data,
			})

		case <-s3ce.StopChan:
			s3ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("stopping configuration")
			config.Active = false
			return nil

		case <-s3ce.DoneCh:
			s3ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is complete")
			return nil

		case err := <-s3ce.ErrChan:
			return err
		}
	}
	return nil
}

// listenEvents listens to minio bucket notifications
func (s3ce *S3ConfigExecutor) listenToEvents(artifact *common.S3Artifact, config *gateways.ConfigContext) {
	defer s3ce.DefaultConfigExecutor.CloseChannels()

	creds, err := store.GetCredentials(s3ce.GatewayConfig.Clientset, s3ce.GatewayConfig.Namespace,  &sv1alphav1.ArtifactLocation{
		S3: artifact,
	})
	if err != nil {
		s3ce.ErrChan <- err
	}
	minioClient, err := store.NewMinioClient(artifact, *creds)
	if err != nil {
		s3ce.ErrChan <- err
	}

	event := s3ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, s3ce.GatewayConfig.Clientset)
	if err != nil {
		s3ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		s3ce.ErrChan <- err
	}

	// at this point configuration is sucessfully running
	s3ce.StartChan <- struct {}{}

	// Listen for bucket notifications
	for notificationInfo := range minioClient.ListenBucketNotification(artifact.S3Bucket.Bucket, artifact.Filter.Prefix, artifact.Filter.Suffix, []string{
		string(artifact.Event),
	}, s3ce.DoneCh) {
		if notificationInfo.Err != nil {
			s3ce.ErrChan <- notificationInfo.Err
			break
		} else {
			payload, err := json.Marshal(notificationInfo.Records[0])
			if err != nil {
				s3ce.ErrChan <- err
				break
			}
			s3ce.DataCh <- payload
		}
	}
}
