/*
Copyright 2026 The Argoproj Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package grpc

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

func getFakeSensorAndTrigger() (*v1alpha1.Sensor, *v1alpha1.Trigger) {
	trigger := &v1alpha1.Trigger{
		Template: &v1alpha1.TriggerTemplate{
			Name: "fake-grpc-trigger",
			GRPC: &v1alpha1.GRPCTrigger{
				URL:      "localhost:0",
				Method:   "/test.EchoService/Echo",
				Insecure: true,
				Schema: []v1alpha1.GRPCSchemaField{
					{Name: "message", Number: 1, Type: "string"},
				},
				Payload: []v1alpha1.TriggerParameter{
					{Dest: "message"},
				},
			},
		},
	}
	sensor := &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-sensor", Namespace: "fake"},
		Spec:       v1alpha1.SensorSpec{Triggers: []v1alpha1.Trigger{*trigger}},
	}
	return sensor, trigger
}

func TestNewGRPCTrigger_InsecureConnectionAndCaching(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	logger := logging.NewArgoEventsLogger()

	gt, err := NewGRPCTrigger(clients, sensor, trigger, logger)
	require.NoError(t, err)
	require.NotNil(t, gt.Conn)

	cachedConn, ok := clients.Load(trigger.Template.Name)
	require.True(t, ok)
	assert.Same(t, gt.Conn, cachedConn)

	// Calling it again with the same trigger name must reuse the cached conn.
	gt2, err := NewGRPCTrigger(clients, sensor, trigger, logger)
	require.NoError(t, err)
	assert.Same(t, gt.Conn, gt2.Conn)
}

func TestNewGRPCTrigger_RequiresTLSOrInsecure(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	trigger.Template.GRPC.Insecure = false
	trigger.Template.GRPC.TLS = nil
	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()

	_, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either tls or insecure")
}

func TestGRPCTrigger_GetTriggerType(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)
	assert.Equal(t, v1alpha1.TriggerTypeGRPC, gt.GetTriggerType())
}

func TestGRPCTrigger_FetchResource(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)

	res, err := gt.FetchResource(context.TODO())
	require.NoError(t, err)
	fetched, ok := res.(*v1alpha1.GRPCTrigger)
	require.True(t, ok)
	assert.Equal(t, "localhost:0", fetched.URL)
}

func TestGRPCTrigger_ApplyResourceParameters(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)

	defaultURL := "localhost:9999"
	trigger.Template.GRPC.Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dep",
				DataKey:        "url",
				Value:          &defaultURL,
			},
			Dest: "url",
		},
	}

	events := map[string]*v1alpha1.Event{
		"fake-dep": {
			Context: &v1alpha1.EventContext{
				ID:              "1",
				DataContentType: v1alpha1.MediaTypeJSON,
			},
			Data: []byte(`{"url":"localhost:8888"}`),
		},
	}

	res, err := gt.ApplyResourceParameters(events, trigger.Template.GRPC)
	require.NoError(t, err)
	updated, ok := res.(*v1alpha1.GRPCTrigger)
	require.True(t, ok)
	assert.Equal(t, "localhost:8888", updated.URL)
}

func TestNewGRPCTrigger_TLSPath(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	trigger.Template.GRPC.Insecure = false
	// TLSConfig.Enabled with no cert secrets set still yields a valid empty
	// tls.Config (see sharedutil.GetTLSConfig) — enough to exercise the TLS
	// branch of NewGRPCTrigger without needing real certificates.
	trigger.Template.GRPC.TLS = &v1alpha1.TLSConfig{Enabled: true}

	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)
	assert.NotNil(t, gt.Conn)
}

// newTestEchoServer starts an in-process gRPC server exposing
// "/test.EchoService/Echo" without any compiled stubs: the handler decodes
// the raw request into a dynamic message built from the same schema the
// client uses, and returns replyErr (nil for success).
func newTestEchoServer(t *testing.T, schema []v1alpha1.GRPCSchemaField, replyErr error) (addr string, capturedMessage func() *dynamicpb.Message, stop func()) {
	t.Helper()

	desc, err := buildRequestDescriptor(schema)
	require.NoError(t, err)

	var captured *dynamicpb.Message

	serviceDesc := &grpc.ServiceDesc{
		ServiceName: "test.EchoService",
		HandlerType: (*any)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Echo",
				Handler: func(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
					req := dynamicpb.NewMessage(desc)
					if err := dec(req); err != nil {
						return nil, err
					}
					captured = req
					if replyErr != nil {
						return nil, replyErr
					}
					return dynamicpb.NewMessage(emptyResponseDescriptor), nil
				},
			},
		},
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	server.RegisterService(serviceDesc, struct{}{})

	go func() {
		_ = server.Serve(lis)
	}()

	return lis.Addr().String(), func() *dynamicpb.Message { return captured }, server.Stop
}

func TestGRPCTrigger_Execute_Success(t *testing.T) {
	schema := []v1alpha1.GRPCSchemaField{
		{Name: "message", Number: 1, Type: "string"},
	}
	addr, capturedMessage, stop := newTestEchoServer(t, schema, nil)
	defer stop()

	sensor, trigger := getFakeSensorAndTrigger()
	trigger.Template.GRPC.URL = addr
	trigger.Template.GRPC.Schema = schema

	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)

	events := map[string]*v1alpha1.Event{
		"fake-dep": {
			Context: &v1alpha1.EventContext{ID: "1", DataContentType: v1alpha1.MediaTypeJSON},
			Data:    []byte(`{"message":"hello"}`),
		},
	}
	trigger.Template.GRPC.Payload = []v1alpha1.TriggerParameter{
		{
			Src:  &v1alpha1.TriggerParameterSource{DependencyName: "fake-dep", DataKey: "message"},
			Dest: "message",
		},
	}

	result, err := gt.Execute(context.TODO(), events, trigger.Template.GRPC)
	require.NoError(t, err)

	st, ok := result.(*status.Status)
	require.True(t, ok)
	assert.Equal(t, codes.OK, st.Code())

	msg := capturedMessage()
	require.NotNil(t, msg)
	assert.Equal(t, "hello", msg.Get(msg.Descriptor().Fields().ByName("message")).String())
}

func TestGRPCTrigger_Execute_NonOKStatusIsNotAGoError(t *testing.T) {
	schema := []v1alpha1.GRPCSchemaField{
		{Name: "message", Number: 1, Type: "string"},
	}
	addr, _, stop := newTestEchoServer(t, schema, status.Error(codes.NotFound, "no such thing"))
	defer stop()

	sensor, trigger := getFakeSensorAndTrigger()
	trigger.Template.GRPC.URL = addr
	trigger.Template.GRPC.Schema = schema
	defaultMessage := "hello"
	trigger.Template.GRPC.Payload = []v1alpha1.TriggerParameter{
		{
			Src:  &v1alpha1.TriggerParameterSource{DependencyName: "fake-dep", Value: &defaultMessage},
			Dest: "message",
		},
	}

	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)

	result, err := gt.Execute(context.TODO(), map[string]*v1alpha1.Event{}, trigger.Template.GRPC)
	require.NoError(t, err, "a non-OK gRPC status must not surface as a Go error from Execute")

	st, ok := result.(*status.Status)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestGRPCTrigger_ApplyPolicy(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)

	gt.Trigger.Policy = &v1alpha1.TriggerPolicy{
		Status: &v1alpha1.StatusPolicy{Allow: []int32{int32(codes.OK)}},
	}
	assert.NoError(t, gt.ApplyPolicy(context.TODO(), status.New(codes.OK, "")))

	gt.Trigger.Policy = &v1alpha1.TriggerPolicy{
		Status: &v1alpha1.StatusPolicy{Allow: []int32{int32(codes.OK)}},
	}
	err = gt.ApplyPolicy(context.TODO(), status.New(codes.NotFound, "no such thing"))
	assert.Error(t, err)
}

func TestGRPCTrigger_ApplyPolicy_NoPolicyMeansNoCheck(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)

	assert.NoError(t, gt.ApplyPolicy(context.TODO(), status.New(codes.Internal, "boom")))
}
