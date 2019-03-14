package common

import (
	"context"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"google.golang.org/grpc/metadata"
	"net/http"
)

var Hook = &Webhook{
	Endpoint: "/fake",
	Port:     "12000",
	URL:      "test-url",
}

type FakeHttpWriter struct {
	HeaderStatus int
	Payload      []byte
}

func (f *FakeHttpWriter) Header() http.Header {
	return http.Header{}
}

func (f *FakeHttpWriter) Write(body []byte) (int, error) {
	f.Payload = body
	return len(body), nil
}

func (f *FakeHttpWriter) WriteHeader(status int) {
	f.HeaderStatus = status
}

func GetFakeRouteConfig() *RouteConfig {
	return &RouteConfig{
		Webhook: Hook,
		EventSource: &gateways.EventSource{
			Name: "fake-event-source",
			Data: "hello",
			Id:   "123",
		},
		Log:     common.GetLoggerContext(common.LoggerConf()).Logger(),
		Configs: make(map[string]interface{}),
		StartCh: make(chan struct{}),
	}
}

type FakeGRPCStream struct {
	SentData *gateways.Event
	Ctx      context.Context
}

func (f *FakeGRPCStream) Send(event *gateways.Event) error {
	f.SentData = event
	return nil
}

func (f *FakeGRPCStream) SetHeader(metadata.MD) error {
	return nil
}

func (f *FakeGRPCStream) SendHeader(metadata.MD) error {
	return nil
}

func (f *FakeGRPCStream) SetTrailer(metadata.MD) {
	return
}

func (f *FakeGRPCStream) Context() context.Context {
	return f.Ctx
}

func (f *FakeGRPCStream) SendMsg(m interface{}) error {
	return nil
}

func (f *FakeGRPCStream) RecvMsg(m interface{}) error {
	return nil
}
