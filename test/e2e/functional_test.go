//go:build functional
// +build functional

package e2e

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/argoproj/argo-events/test/e2e/fixtures"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/suite"
)

type FunctionalSuite struct {
	fixtures.E2ESuite
}

const (
	LogEventSourceStarted     = "Eventing server started."
	LogSensorStarted          = "Sensor started."
	LogPublishEventSuccessful = "succeeded to publish an event"
	LogTriggerActionFailed    = "failed to execute a trigger"
)

func LogTriggerActionSuccessful(triggerName string) string {
	return fmt.Sprintf("successfully processed trigger '%s'", triggerName)
}

func (s *FunctionalSuite) e(baseURL string) *httpexpect.Expect {
	return httpexpect.
		WithConfig(httpexpect.Config{
			BaseURL:  baseURL,
			Reporter: httpexpect.NewRequireReporter(s.T()),
			Printers: []httpexpect.Printer{
				httpexpect.NewDebugPrinter(NewHttpLogger(), true),
			},
			Client: &http.Client{
				Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			},
		}).
		Builder(func(req *httpexpect.Request) {})
}

func (s *FunctionalSuite) TestCreateCalendarEventSource() {
	t1 := s.Given().EventSource("@testdata/es-calendar.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady().
		Then().
		ExpectEventSourcePodLogContains(LogPublishEventSuccessful, nil)

	defer t1.When().DeleteEventSource()

	t2 := s.Given().Sensor("@testdata/sensor-log.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger"), nil)

	defer t2.When().DeleteSensor()
}

func (s *FunctionalSuite) TestCreateCalendarEventSourceWithHA() {
	t1 := s.Given().EventSource("@testdata/es-calendar-ha.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady().
		Wait(3*time.Second).
		Then().
		ExpectEventSourcePodLogContains(LogPublishEventSuccessful, nil)

	defer t1.When().DeleteEventSource()

	t2 := s.Given().Sensor("@testdata/sensor-log-ha.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Wait(3*time.Second).
		Then().
		ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger"), nil)
	defer t2.When().DeleteSensor()
}

func (s *FunctionalSuite) TestMetricsWithCalendar() {
	t1 := s.Given().EventSource("@testdata/es-calendar-metrics.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady().
		Then().
		ExpectEventSourcePodLogContains(LogEventSourceStarted, nil).
		EventSourcePodPortForward(7777, 7777)

	defer t1.TerminateAllPodPortForwards()
	defer t1.When().DeleteEventSource()

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful, nil)

	// EventSource POD metrics
	s.e("http://localhost:7777").GET("/metrics").
		Expect().
		Status(200).
		Body().
		Contains("argo_events_event_service_running_total").
		Contains("argo_events_events_sent_total").
		Contains("argo_events_event_processing_duration_milliseconds")

	t2 := s.Given().Sensor("@testdata/sensor-log-metrics.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted, nil).
		SensorPodPortForward(7778, 7777)

	defer t2.TerminateAllPodPortForwards()
	defer t2.When().DeleteSensor()

	t2.ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger"), nil)

	// Sensor POD metrics
	s.e("http://localhost:7778").GET("/metrics").
		Expect().
		Status(200).
		Body().
		Contains("argo_events_action_triggered_total").
		Contains("argo_events_action_duration_milliseconds")
}

func (s *FunctionalSuite) TestMetricsWithWebhook() {
	t1 := s.Given().EventSource("@testdata/es-test-metrics-webhook.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady().
		Then().
		ExpectEventSourcePodLogContains(LogEventSourceStarted, nil).
		EventSourcePodPortForward(12000, 12000).
		EventSourcePodPortForward(7777, 7777)

	defer t1.TerminateAllPodPortForwards()
	defer t1.When().DeleteEventSource()

	t2 := s.Given().Sensor("@testdata/sensor-test-metrics.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted, nil).
		SensorPodPortForward(7778, 7777)

	defer t2.TerminateAllPodPortForwards()
	defer t2.When().DeleteSensor()

	time.Sleep(3 * time.Second)

	s.e("http://localhost:12000").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful, nil)

	// Post something invalid
	s.e("http://localhost:12000").POST("/example").WithBytes([]byte("Invalid JSON")).
		Expect().
		Status(400)

	// EventSource POD metrics
	s.e("http://localhost:7777").GET("/metrics").
		Expect().
		Status(200).
		Body().
		Contains("argo_events_event_service_running_total").
		Contains("argo_events_events_sent_total").
		Contains("argo_events_event_processing_duration_milliseconds").
		Contains("argo_events_events_processing_failed_total")

	// Expect to see 1 success and 1 failure
	t2.ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger"), nil).
		ExpectSensorPodLogContains(LogTriggerActionFailed, nil)

	// Sensor POD metrics
	s.e("http://localhost:7778").GET("/metrics").
		Expect().
		Status(200).
		Body().
		Contains("argo_events_action_triggered_total").
		Contains("argo_events_action_duration_milliseconds").
		Contains("argo_events_action_failed_total")
}

func (s *FunctionalSuite) TestResourceEventSource() {
	w1 := s.Given().EventSource("@testdata/es-resource.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady().
		Exec("kubectl", []string{"-n", fixtures.Namespace, "run", "test-pod", "--image", "hello-world", "-l", fixtures.Label + "=" + fixtures.LabelValue}, fixtures.OutputRegexp(`pod/.* created`))

	t1 := w1.Then().
		ExpectEventSourcePodLogContains(LogEventSourceStarted, nil)
	defer t1.When().DeleteEventSource()

	t2 := s.Given().Sensor("@testdata/sensor-resource.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted, nil)
	defer t2.When().DeleteSensor()

	w1.Exec("kubectl", []string{"-n", fixtures.Namespace, "delete", "pod", "test-pod"}, fixtures.OutputRegexp(`pod "test-pod" deleted`))

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful, nil)

	t2.ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger"), nil)
}

func (s *FunctionalSuite) TestMultiDependencyConditions() {

	t1 := s.Given().EventSource("@testdata/es-multi-dep.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady().
		Then().
		ExpectEventSourcePodLogContains(LogEventSourceStarted, nil).
		EventSourcePodPortForward(12001, 12000).
		EventSourcePodPortForward(13001, 13000).
		EventSourcePodPortForward(14001, 14000).
		EventSourcePodPortForward(7777, 7777)

	defer t1.When().DeleteEventSource()
	defer t1.TerminateAllPodPortForwards()

	zeroCount := 0
	oneCount := 1
	twoCount := 2
	t2 := s.Given().Sensor("@testdata/sensor-multi-dep.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted, &oneCount).
		SensorPodPortForward(7778, 7777)

	defer t2.When().DeleteSensor()
	defer t2.TerminateAllPodPortForwards()

	time.Sleep(3 * time.Second)

	// need to verify the conditional logic is working successfully
	// If we trigger test-dep-1 (port 12000) we should see log-trigger-2 but not log-trigger-1
	s.e("http://localhost:12001").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful, &oneCount)

	t2.ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-2"), &oneCount).
		ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), &zeroCount)

	// Then if we trigger test-dep-2 we should see log-trigger-2
	s.e("http://localhost:13001").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	t2.ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), &oneCount)

	// Then we trigger test-dep-2 again and shouldn't see anything
	s.e("http://localhost:13001").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)
	t2.ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), &oneCount)

	// Finally trigger test-dep-3 and we should see log-trigger-1..
	s.e("http://localhost:14001").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)
	t2.ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), &twoCount)
}

// Start Pod with a multidependency condition
// send it one dependency
// verify that if it goes down and comes back up it triggers when sent the other part of the condition
func (s *FunctionalSuite) TestDurableConsumer() {

	t1 := s.Given().EventSource("@testdata/es-durable-consumer.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady().
		Then().
		ExpectEventSourcePodLogContains(LogEventSourceStarted, nil).
		EventSourcePodPortForward(12002, 12000).
		EventSourcePodPortForward(13002, 13000).
		EventSourcePodPortForward(14002, 14000).
		EventSourcePodPortForward(7779, 7777)

	defer t1.When().DeleteEventSource()
	defer t1.TerminateAllPodPortForwards()

	oneCount := 1
	twoCount := 2
	t2 := s.Given().Sensor("@testdata/sensor-durable-consumer.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted, &oneCount).
		SensorPodPortForward(7780, 7777)

	// test-dep-1
	s.e("http://localhost:12002").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful, &oneCount)

	t2.TerminateAllPodPortForwards()

	// delete the Sensor
	t2.When().
		DeleteSensor()

	time.Sleep(6 * time.Second) // need to give this time to be deleted before we create the new one

	t3 := s.Given().Sensor("@testdata/sensor-durable-consumer.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted, &oneCount).
		SensorPodPortForward(7780, 7777)

	defer t3.TerminateAllPodPortForwards()
	defer t3.When().DeleteSensor()
	// test-dep-2
	s.e("http://localhost:13002").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	time.Sleep(60 * time.Second) // takes a little while for the first dependency to get sent to our new consumer

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful, &twoCount)
	t3.ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), &oneCount)

}

func (s *FunctionalSuite) TestMultipleSensors() {
	// Start two sensors which each use "A && B", but staggered in time such that one receives the partial condition
	// Then send the other part of the condition and verify that only one triggers

	// Start EventSource
	t1 := s.Given().EventSource("@testdata/es-multi-sensor.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady().
		Then().
		ExpectEventSourcePodLogContains(LogEventSourceStarted, nil).
		EventSourcePodPortForward(12003, 12000).
		EventSourcePodPortForward(13003, 13000).
		EventSourcePodPortForward(14003, 14000).
		EventSourcePodPortForward(7781, 7777)

	defer t1.When().DeleteEventSource()
	defer t1.TerminateAllPodPortForwards()

	// Start one Sensor
	zeroCount := 0
	oneCount := 1
	twoCount := 2
	t2 := s.Given().Sensor("@testdata/sensor-multi-sensor.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted, &oneCount).
		SensorPodPortForward(7782, 7777)

	defer t2.TerminateAllPodPortForwards()
	defer t2.When().DeleteSensor()

	time.Sleep(3 * time.Second)

	// Trigger first dependency
	// test-dep-1
	s.e("http://localhost:12003").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful, &oneCount)

	// Start second Sensor
	t3 := s.Given().Sensor("@testdata/sensor-multi-sensor-2.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted, &oneCount).
		SensorPodPortForward(7783, 7777)

	defer t3.TerminateAllPodPortForwards()
	defer t3.When().DeleteSensor()

	time.Sleep(3 * time.Second)

	// Trigger second dependency
	// test-dep-2
	s.e("http://localhost:13003").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)
	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful, &twoCount) //todo: could speed things up by instead of looking for count just search for exact string

	// Verify trigger occurs for first Sensor and not second
	t2.ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), nil)
	t3.ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), &zeroCount)

}

func (s *FunctionalSuite) TestTriggerSpecChange() {
	// Start a sensor which uses "A && B"; send A; replace the Sensor with a new spec which uses A; send C and verify that there's no trigger

	// Start EventSource
	t1 := s.Given().EventSource("@testdata/es-trigger-spec-change.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady().
		Then().
		ExpectEventSourcePodLogContains(LogEventSourceStarted, nil).
		EventSourcePodPortForward(12004, 12000).
		EventSourcePodPortForward(13004, 13000).
		EventSourcePodPortForward(14004, 14000).
		EventSourcePodPortForward(7784, 7777)

	defer t1.When().DeleteEventSource()
	defer t1.TerminateAllPodPortForwards()

	// Start one Sensor
	zeroCount := 0
	oneCount := 1
	twoCount := 2
	t2 := s.Given().Sensor("@testdata/sensor-trigger-spec-change.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted, &oneCount).
		SensorPodPortForward(7785, 7777)

	time.Sleep(3 * time.Second)

	// Trigger first dependency
	// test-dep-1
	s.e("http://localhost:12004").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful, &oneCount)

	t2.TerminateAllPodPortForwards()
	t2.When().DeleteSensor()
	time.Sleep(3 * time.Second)

	// Change Sensor's spec
	t2 = s.Given().Sensor("@testdata/sensor-trigger-spec-change-2.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted, &oneCount).
		SensorPodPortForward(7786, 7777)

	defer t2.TerminateAllPodPortForwards()
	defer t2.When().DeleteSensor()

	time.Sleep(3 * time.Second)

	// test-dep-3
	s.e("http://localhost:14004").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful, &twoCount)
	// Verify no Trigger this time since test-dep-1 should have been cleared
	t2.ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), &zeroCount)
}

func TestFunctionalSuite(t *testing.T) {
	suite.Run(t, new(FunctionalSuite))
}
