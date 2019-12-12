package common

import (
	"context"
	"github.com/argoproj/argo-events/gateways"
	"google.golang.org/grpc/metadata"
)

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
