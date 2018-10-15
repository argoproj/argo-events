package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	twClient "github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

type EventType string

var (
	All              EventType = "All"
	Tweet            EventType = "Tweet"
	DM               EventType = "DM"
	StatusDeletion   EventType = "StatusDeletion"
	LocationDeletion EventType = "LocationDeletion"
	StatusWithheld   EventType = "StatusWitheld"
	UserWithheld     EventType = "UserWitheld"
	FriendsList      EventType = "FriendList"
	Event            EventType = "Event"
	Other            EventType = "Other"
)

// twitter implements ConfigExecutor
type twitter struct{}

// twitterConfig contains
type twitterConfig struct {
	// ConsumerKey required to access api
	ConsumerKey TwitterAccess `json:"consumerKey" protobuf:"bytes,1,opt,name=consumerKey"`

	// ConsumerSecret required to access api
	ConsumerSecret TwitterAccess `json:"consumerSecret" protobuf:"bytes,2,opt,name=consumerSecret"`

	// AccessToken required to access api
	AccessToken TwitterAccess `json:"accessToken" protobuf:"bytes,3,opt,name=accessToken"`

	// AccessSecret required to access api
	AccessSecret TwitterAccess `json:"accessSecret" protobuf:"bytes,4,opt,name=accessSecret"`

	// FilterParams return Tweets that match one or more filtering predicates
	FilterParams *twClient.StreamFilterParams `json:"filterParams" protobuf:"bytes,5,opt,name=filterParams"`

	// UserParams provide messages specific to the authenticate User and possibly those they follow.
	UserParams *twClient.StreamUserParams `json:"userParams" protobuf:"bytes,6,opt,name=userParams"`

	// FirehoseParams
	FirehoseParams *twClient.StreamFirehoseParams `json:"firehoseParams" protobuf:"bytes,7,opt,name=firehoseParams"`

	// SiteParams
	SiteParams *twClient.StreamSiteParams `json:"siteParams" protobuf:"bytes,8,opt,name=siteParams"`

	// EventTypes are types of event to listen to
	EventTypes []string
}

// TwitterAccess references k8 secret which contains Twitter API authentication information
type TwitterAccess struct {
	// Name of the k8 secret
	Name string
	// Key referring to access token/secret
	Key string
}

func readFromK8Secrets(secretName, key string) (string, error) {
	secret, err := gatewayConfig.Clientset.CoreV1().Secrets(gatewayConfig.Namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	if data, ok := secret.Data[key]; !ok {
		return "", fmt.Errorf("key %s not found in secret %s", key, secretName)
	} else {
		return string(data), nil
	}
}

func encodeAndDispatch(message interface{}, src string) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(message)
	if err != nil {
		gatewayConfig.Log.Error().Err(err).Str("config-key", src).Msg("failed to encode slack event")
	} else {
		gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
			Src:     src,
			Payload: buff.Bytes(),
		})
	}
}

// Runs a gateway configuration
func (t *twitter) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	var twConfig *twitterConfig
	err = yaml.Unmarshal([]byte(config.Data.Config), twConfig)
	if err != nil {
		return err
	}

	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("started configuration")

	consumerKey, err := readFromK8Secrets(twConfig.ConsumerKey.Name, twConfig.ConsumerKey.Key)
	if err != nil {
		errMessage = "failed to read consumer key"
		return err
	}

	consumerSecret, err := readFromK8Secrets(twConfig.ConsumerSecret.Name, twConfig.ConsumerSecret.Key)
	if err != nil {
		errMessage = "failed to read consumer secret"
		return err
	}

	accessToken, err := readFromK8Secrets(twConfig.AccessToken.Name, twConfig.AccessToken.Key)
	if err != nil {
		errMessage = "failed to read access token"
	}

	accessSecret, err := readFromK8Secrets(twConfig.AccessSecret.Name, twConfig.AccessSecret.Key)
	if err != nil {
		errMessage = "failed to get "
	}

	cnf := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := cnf.Client(oauth1.NoContext, token)
	client := twClient.NewClient(httpClient)
	streamSvc := client.Streams

	var stream *twClient.Stream

	if twConfig.FilterParams != nil {
		twConfig.FilterParams.StallWarnings = twClient.Bool(true)
		stream, err = streamSvc.Filter(twConfig.FilterParams)
		if err != nil {
			errMessage = "failed to apply filter params"
			return err
		}
	}
	if twConfig.UserParams != nil {
		twConfig.UserParams.StallWarnings = twClient.Bool(true)
		stream, err = streamSvc.User(twConfig.UserParams)
		if err != nil {
			errMessage = "failed to apply user params"
			return err
		}
	}
	if twConfig.FirehoseParams != nil {
		twConfig.FirehoseParams.StallWarnings = twClient.Bool(true)
		stream, err = streamSvc.Firehose(twConfig.FirehoseParams)
		if err != nil {
			errMessage = "failed to apply firehose params"
			return err
		}
	}
	if twConfig.SiteParams != nil {
		twConfig.SiteParams.StallWarnings = twClient.Bool(true)
		stream, err = streamSvc.Site(twConfig.SiteParams)
		if err != nil {
			errMessage = "failed to apply site params"
			return err
		}
	}

	demux := twClient.NewSwitchDemux()
	for _, eventType := range twConfig.EventTypes {
		switch EventType(eventType) {
		case All:
			demux.All = func(message interface{}) {
				encodeAndDispatch(message, config.Data.Src)
			}
		case Tweet:
			demux.Tweet = func(tweet *twClient.Tweet) {
				encodeAndDispatch(tweet, config.Data.Src)
			}
		case StatusWithheld:
			demux.StatusWithheld = func(statusWithheld *twClient.StatusWithheld) {
				encodeAndDispatch(statusWithheld, config.Data.Src)
			}
		case Event:
			demux.Event = func(event *twClient.Event) {
				encodeAndDispatch(event, config.Data.Src)
			}
		case DM:
			demux.DM = func(dm *twClient.DirectMessage) {
				encodeAndDispatch(dm, config.Data.Src)
			}
		case FriendsList:
			demux.FriendsList = func(friendsList *twClient.FriendsList) {
				encodeAndDispatch(friendsList, config.Data.Src)
			}
		case LocationDeletion:
			demux.LocationDeletion = func(locationDeletion *twClient.LocationDeletion) {
				encodeAndDispatch(locationDeletion, config.Data.Src)
			}
		case StatusDeletion:
			demux.StatusDeletion = func(deletion *twClient.StatusDeletion) {
				encodeAndDispatch(deletion, config.Data.Src)
			}
		case UserWithheld:
			demux.UserWithheld = func(userWithheld *twClient.UserWithheld) {
				encodeAndDispatch(userWithheld, config.Data.Src)
			}
		case Other:
			demux.Other = func(message interface{}) {
				encodeAndDispatch(message, config.Data.Src)
			}
		default:
			errMessage = fmt.Sprintf("unknown event tye %s", eventType)
			return fmt.Errorf("failed to process event type")
		}
	}

	config.Active = true
	// generate k8 event to update configuration state in gateway.
	event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, gatewayConfig.Clientset)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		<-config.StopCh
		config.Active = false
		gatewayConfig.Log.Info().Str("config", config.Data.Src).Msg("stopping the configuration...")
		stream.Stop()
		wg.Done()
	}()

	go func() {
		demux.HandleChan(stream.Messages)
	}()

	wg.Wait()
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is now complete.")
	return nil
}

// Stops a configuration
func (t *twitter) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}

func main() {
	err := gatewayConfig.TransformerReadinessProbe()
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to connect to gateway transformer")
	}
	_, err = gatewayConfig.WatchGatewayEvents(context.Background())
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch k8 events for gateway configuration state updates")
	}
	_, err = gatewayConfig.WatchGatewayConfigMap(context.Background(), &twitter{})
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg("failed to watch gateway configuration updates")
	}
	select {}
}
