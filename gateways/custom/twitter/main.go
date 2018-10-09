package main

import (
	tw "github.com/dghubble/go-twitter/twitter"
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
}

// TwitterAccess references k8 secret which contains Twitter API authentication information
type TwitterAccess struct {
	// Name of the k8 secret
	Name string
	// Key referring to access token/secret
	Key string
}

func main() {
	demux := tw.NewSwitchDemux()
	demux.DM = func(dm *tw.DirectMessage) {

	}
}
