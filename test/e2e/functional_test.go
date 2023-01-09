//go:build functional

package e2e

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/argoproj/argo-events/test/e2e/fixtures"
	"github.com/argoproj/argo-events/test/util"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/suite"
)

type FunctionalSuite struct {
	fixtures.E2ESuite
}

const (
	LogEventSourceStarted     = "Eventing server started."
	LogSensorStarted          = "Sensor started."
	LogPublishEventSuccessful = "Succeeded to publish an event"
	LogTriggerActionFailed    = "Failed to execute a trigger"
)

func LogTriggerActionSuccessful(triggerName string) string {
	return fmt.Sprintf("Successfully processed trigger '%s'", triggerName)
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
		ExpectEventSourcePodLogContains(LogPublishEventSuccessful)

	defer t1.When().DeleteEventSource()

	t2 := s.Given().Sensor("@testdata/sensor-log.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger"))

	defer t2.When().DeleteSensor()
}

func (s *FunctionalSuite) TestCreateCalendarEventSourceWithHA() {
	t1 := s.Given().EventSource("@testdata/es-calendar-ha.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady().
		Wait(3 * time.Second).
		Then().
		ExpectEventSourcePodLogContains(LogPublishEventSuccessful)

	defer t1.When().DeleteEventSource()

	t2 := s.Given().Sensor("@testdata/sensor-log-ha.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Wait(3 * time.Second).
		Then().
		ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger"))
	defer t2.When().DeleteSensor()
}

func (s *FunctionalSuite) TestMetricsWithCalendar() {
	w1 := s.Given().EventSource("@testdata/es-calendar-metrics.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady()

	defer w1.DeleteEventSource()

	w1.Then().
		ExpectEventSourcePodLogContains(LogEventSourceStarted)

	defer w1.Then().EventSourcePodPortForward(17777, 7777).TerminateAllPodPortForwards()

	w1.Then().ExpectEventSourcePodLogContains(LogPublishEventSuccessful)

	// EventSource POD metrics
	s.e("http://localhost:17777").GET("/metrics").
		Expect().
		Status(200).
		Body().
		Contains("argo_events_event_service_running_total").
		Contains("argo_events_events_sent_total").
		Contains("argo_events_event_processing_duration_milliseconds")

	w2 := s.Given().Sensor("@testdata/sensor-log-metrics.yaml").
		When().
		CreateSensor().
		WaitForSensorReady()
	defer w2.DeleteSensor()

	w2.Then().
		ExpectSensorPodLogContains(LogSensorStarted)
	defer w2.Then().SensorPodPortForward(17778, 7777).TerminateAllPodPortForwards()

	w2.Then().ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger"))

	// Sensor POD metrics
	s.e("http://localhost:17778").GET("/metrics").
		Expect().
		Status(200).
		Body().
		Contains("argo_events_action_triggered_total").
		Contains("argo_events_action_duration_milliseconds")
}

func (s *FunctionalSuite) TestMetricsWithWebhook() {
	w1 := s.Given().EventSource("@testdata/es-test-metrics-webhook.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady()

	defer w1.DeleteEventSource()

	w1.Then().ExpectEventSourcePodLogContains(LogEventSourceStarted)

	defer w1.Then().
		EventSourcePodPortForward(12300, 12000).
		EventSourcePodPortForward(7717, 7777).TerminateAllPodPortForwards()

	w2 := s.Given().Sensor("@testdata/sensor-test-metrics.yaml").
		When().
		CreateSensor().
		WaitForSensorReady()

	defer w2.DeleteSensor()
	w2.Then().ExpectSensorPodLogContains(LogSensorStarted)

	defer w2.Then().
		SensorPodPortForward(7718, 7777).TerminateAllPodPortForwards()

	time.Sleep(3 * time.Second)

	s.e("http://localhost:12300").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	w1.Then().ExpectEventSourcePodLogContains(LogPublishEventSuccessful)

	// Post something invalid
	s.e("http://localhost:12300").POST("/example").WithBytes([]byte("Invalid JSON")).
		Expect().
		Status(400)

	// EventSource POD metrics
	s.e("http://localhost:7717").GET("/metrics").
		Expect().
		Status(200).
		Body().
		Contains("argo_events_event_service_running_total").
		Contains("argo_events_events_sent_total").
		Contains("argo_events_event_processing_duration_milliseconds").
		Contains("argo_events_events_processing_failed_total")

	// Expect to see 1 success and 1 failure
	w2.Then().ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger")).
		ExpectSensorPodLogContains(LogTriggerActionFailed)

	// Sensor POD metrics
	s.e("http://localhost:7718").GET("/metrics").
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
		ExpectEventSourcePodLogContains(LogEventSourceStarted)
	defer t1.When().DeleteEventSource()

	t2 := s.Given().Sensor("@testdata/sensor-resource.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted)
	defer t2.When().DeleteSensor()

	w1.Exec("kubectl", []string{"-n", fixtures.Namespace, "delete", "pod", "test-pod"}, fixtures.OutputRegexp(`pod "test-pod" deleted`))

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful)

	t2.ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger"))
}

func (s *FunctionalSuite) TestMultiDependencyConditions() {

	w1 := s.Given().EventSource("@testdata/es-multi-dep.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady()
	defer w1.DeleteEventSource()

	w1.Then().ExpectEventSourcePodLogContains(LogEventSourceStarted)

	defer w1.Then().
		EventSourcePodPortForward(12011, 12000).
		EventSourcePodPortForward(13011, 13000).
		EventSourcePodPortForward(14011, 14000).
		TerminateAllPodPortForwards()

	w2 := s.Given().Sensor("@testdata/sensor-multi-dep.yaml").
		When().
		CreateSensor().
		WaitForSensorReady()
	defer w2.DeleteSensor()

	w2.Then().ExpectSensorPodLogContains(LogSensorStarted, util.PodLogCheckOptionWithCount(1))

	time.Sleep(3 * time.Second)

	// need to verify the conditional logic is working successfully
	// If we trigger test-dep-1 (port 12000) we should see log-trigger-2 but not log-trigger-1
	s.e("http://localhost:12011").POST("/example1").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	w1.Then().ExpectEventSourcePodLogContains(LogPublishEventSuccessful, util.PodLogCheckOptionWithCount(1))

	w2.Then().ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-2"), util.PodLogCheckOptionWithCount(1)).
		ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), util.PodLogCheckOptionWithCount(0))

	// Then if we trigger test-dep-2 we should see log-trigger-2
	s.e("http://localhost:13011").POST("/example2").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	w2.Then().ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), util.PodLogCheckOptionWithCount(1))

	// Then we trigger test-dep-2 again and shouldn't see anything
	s.e("http://localhost:13011").POST("/example2").WithBytes([]byte("{}")).
		Expect().
		Status(200)
	w2.Then().ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), util.PodLogCheckOptionWithCount(1))

	// Finally trigger test-dep-3 and we should see log-trigger-1..
	s.e("http://localhost:14011").POST("/example3").WithBytes([]byte("{}")).
		Expect().
		Status(200)
	w2.Then().ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), util.PodLogCheckOptionWithCount(2))
}

// Start Pod with a multidependency condition
// send it one dependency
// verify that if it goes down and comes back up it triggers when sent the other part of the condition
func (s *FunctionalSuite) TestDurableConsumer() {
	if fixtures.GetBusDriverSpec() == fixtures.E2EEventBusSTAN {
		s.T().SkipNow() // todo: TestDurableConsumer() is being skipped for now due to it not reliably passing with the STAN bus
		// (because when Sensor pod restarts it sometimes takes a little while for the STAN bus to resend the message to the durable consumer)
	}

	w1 := s.Given().EventSource("@testdata/es-durable-consumer.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady()
	defer w1.DeleteEventSource()

	w1.Then().ExpectEventSourcePodLogContains(LogEventSourceStarted)

	defer w1.Then().
		EventSourcePodPortForward(12102, 12000).
		EventSourcePodPortForward(13102, 13000).TerminateAllPodPortForwards()

	w2 := s.Given().Sensor("@testdata/sensor-durable-consumer.yaml").
		When().
		CreateSensor().
		WaitForSensorReady()

	w2.Then().
		ExpectSensorPodLogContains(LogSensorStarted, util.PodLogCheckOptionWithCount(1))

	// test-dep-1
	s.e("http://localhost:12102").POST("/example1").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	w1.Then().ExpectEventSourcePodLogContains(LogPublishEventSuccessful, util.PodLogCheckOptionWithCount(1))

	// delete the Sensor
	w2.DeleteSensor().Then().ExpectNoSensorPodFound()

	w3 := s.Given().Sensor("@testdata/sensor-durable-consumer.yaml").
		When().
		CreateSensor().
		WaitForSensorReady()
	defer w3.DeleteSensor()

	w3.Then().
		ExpectSensorPodLogContains(LogSensorStarted, util.PodLogCheckOptionWithCount(1))

	// test-dep-2
	s.e("http://localhost:13102").POST("/example2").WithBytes([]byte("{}")).
		Expect().
		Status(200)
	w1.Then().ExpectEventSourcePodLogContains(LogPublishEventSuccessful, util.PodLogCheckOptionWithCount(2))
	w3.Then().ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), util.PodLogCheckOptionWithCount(1))
}

func (s *FunctionalSuite) TestMultipleSensors() {
	// Start two sensors which each use "A && B", but staggered in time such that one receives the partial condition
	// Then send the other part of the condition and verify that only one triggers

	// Start EventSource
	w1 := s.Given().EventSource("@testdata/es-multi-sensor.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady()
	defer w1.DeleteEventSource()

	w1.Then().
		ExpectEventSourcePodLogContains(LogEventSourceStarted)

	defer w1.Then().EventSourcePodPortForward(12003, 12000).
		EventSourcePodPortForward(13003, 13000).
		EventSourcePodPortForward(14003, 14000).TerminateAllPodPortForwards()

	// Start one Sensor
	w2 := s.Given().Sensor("@testdata/sensor-multi-sensor.yaml").
		When().
		CreateSensor().
		WaitForSensorReady()
	defer w2.DeleteSensor()

	w2.Then().
		ExpectSensorPodLogContains(LogSensorStarted, util.PodLogCheckOptionWithCount(1))

	time.Sleep(3 * time.Second)

	// Trigger first dependency
	// test-dep-1
	s.e("http://localhost:12003").POST("/example1").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	w1.Then().ExpectEventSourcePodLogContains(LogPublishEventSuccessful, util.PodLogCheckOptionWithCount(1))

	// Start second Sensor
	w3 := s.Given().Sensor("@testdata/sensor-multi-sensor-2.yaml").
		When().
		CreateSensor().
		WaitForSensorReady()
	defer w3.DeleteSensor()

	w3.Then().
		ExpectSensorPodLogContains(LogSensorStarted, util.PodLogCheckOptionWithCount(1))

	// Trigger second dependency
	// test-dep-2
	s.e("http://localhost:13003").POST("/example2").WithBytes([]byte("{}")).
		Expect().
		Status(200)
	w1.Then().ExpectEventSourcePodLogContains(LogPublishEventSuccessful, util.PodLogCheckOptionWithCount(2))

	// Verify trigger occurs for first Sensor and not second
	w2.Then().ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"))
	w3.Then().ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), util.PodLogCheckOptionWithCount(0))

}

func (s *FunctionalSuite) TestAtLeastOnce() {
	// Send an event to a sensor with a failing trigger and make sure it doesn't ACK it.
	// Delete the sensor and launch sensor with same name and non-failing trigger so it ACKS it.

	// Start EventSource

	if fixtures.GetBusDriverSpec() == fixtures.E2EEventBusSTAN {
		s.T().SkipNow() // Skipping because AtLeastOnce does not apply for NATS.
	}
	w1 := s.Given().EventSource("@testdata/es-atleastonce.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady()

	defer w1.DeleteEventSource()

	w1.Then().
		ExpectEventSourcePodLogContains(LogEventSourceStarted)

	defer w1.Then().EventSourcePodPortForward(12006, 12000).
		TerminateAllPodPortForwards()

	w2 := s.Given().Sensor("@testdata/sensor-atleastonce-failing.yaml").
		When().
		CreateSensor().
		WaitForSensorReady()
	w2.Then().
		ExpectSensorPodLogContains(LogSensorStarted, util.PodLogCheckOptionWithCount(1))
	time.Sleep(3 * time.Second)
	s.e("http://localhost:12006").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	w1.Then().ExpectEventSourcePodLogContains(LogPublishEventSuccessful, util.PodLogCheckOptionWithCount(1))
	w2.Then().ExpectSensorPodLogContains("InProgess")
	w2.DeleteSensor()
	time.Sleep(10 * time.Second)

	w3 := s.Given().Sensor("@testdata/sensor-atleastonce-triggerable.yaml").
		When().
		CreateSensor().
		WaitForSensorReady()
	defer w3.DeleteSensor()

	w3.Then().
		ExpectSensorPodLogContains(LogSensorStarted, util.PodLogCheckOptionWithCount(1))

	w3.Then().
		ExpectSensorPodLogContains(LogTriggerActionSuccessful("trigger-atleastonce"))
}

func (s *FunctionalSuite) TestAtMostOnce() {
	// Send an event to a sensor with a failing trigger but it will ACK it.
	// Delete the sensor and launch sensor with same name and non-failing trigger
	// to see that the event doesn't come through.


	// Start EventSource
	w1 := s.Given().EventSource("@testdata/es-atleastonce.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady()
	defer w1.DeleteEventSource()
	w1.Then().
		ExpectEventSourcePodLogContains(LogEventSourceStarted)

	defer w1.Then().EventSourcePodPortForward(12007, 12000).
		TerminateAllPodPortForwards()


	w2 := s.Given().Sensor("@testdata/sensor-atmostonce-failing.yaml").
		When().
		CreateSensor().
		WaitForSensorReady()
	w2.Then().
		ExpectSensorPodLogContains(LogSensorStarted, util.PodLogCheckOptionWithCount(1))
	time.Sleep(3 * time.Second)
	s.e("http://localhost:12007").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	w1.Then().ExpectEventSourcePodLogContains(LogPublishEventSuccessful, util.PodLogCheckOptionWithCount(1))
	w2.Then().ExpectSensorPodLogContains("acked message of Stream seq")
	time.Sleep(3 * time.Second)

	w2.DeleteSensor()

	w3 := s.Given().Sensor("@testdata/sensor-atmostonce-triggerable.yaml").
		When().
		CreateSensor().
		WaitForSensorReady()
	defer w3.DeleteSensor()

	w3.Then().
		ExpectSensorPodLogContains(LogSensorStarted, util.PodLogCheckOptionWithCount(1))

	w3.Then().
	ExpectSensorPodLogContains(LogTriggerActionSuccessful("trigger-atmostonce"), util.PodLogCheckOptionWithCount(0))
}

func (s *FunctionalSuite) TestMultipleSensorAtLeastOnceTrigger() {
	// Start two sensors which each use "A && B", but staggered in time such that one receives the partial condition
	// Then send the other part of the condition and verify that only one triggers

	// Start EventSource

	w1 := s.Given().EventSource("@testdata/es-multi-sensor.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady()
	defer w1.DeleteEventSource()

	w1.Then().
		ExpectEventSourcePodLogContains(LogEventSourceStarted)

	defer w1.Then().EventSourcePodPortForward(12004, 12000).
		EventSourcePodPortForward(13004, 13000).
		EventSourcePodPortForward(14004, 14000).TerminateAllPodPortForwards()

	// Start one Sensor
	w2 := s.Given().Sensor("@testdata/sensor-multi-sensor-atleastonce.yaml").
		When().
		CreateSensor().
		WaitForSensorReady()
	defer w2.DeleteSensor()

	w2.Then().
		ExpectSensorPodLogContains(LogSensorStarted, util.PodLogCheckOptionWithCount(1))

	time.Sleep(3 * time.Second)

	// Trigger first dependency
	// test-dep-1
	s.e("http://localhost:12004").POST("/example1").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	w1.Then().ExpectEventSourcePodLogContains(LogPublishEventSuccessful, util.PodLogCheckOptionWithCount(1))

	// Start second Sensor
	w3 := s.Given().Sensor("@testdata/sensor-multi-sensor-2-atleastonce.yaml").
		When().
		CreateSensor().
		WaitForSensorReady()
	defer w3.DeleteSensor()

	w3.Then().
		ExpectSensorPodLogContains(LogSensorStarted, util.PodLogCheckOptionWithCount(1))

	// Trigger second dependency
	// test-dep-2
	s.e("http://localhost:13004").POST("/example2").WithBytes([]byte("{}")).
		Expect().
		Status(200)
	w1.Then().ExpectEventSourcePodLogContains(LogPublishEventSuccessful, util.PodLogCheckOptionWithCount(2))

	// Verify trigger occurs for first Sensor and not second
	w2.Then().ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1-atleastonce"))
	w3.Then().ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1-atleastonce"), util.PodLogCheckOptionWithCount(0))

}

func (s *FunctionalSuite) TestTriggerSpecChange() {
	// Start a sensor which uses "A && B"; send A; replace the Sensor with a new spec which uses A; send C and verify that there's no trigger

	// Start EventSource
	t1 := s.Given().EventSource("@testdata/es-trigger-spec-change.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady().
		Then().
		ExpectEventSourcePodLogContains(LogEventSourceStarted).
		EventSourcePodPortForward(12005, 12000).
		EventSourcePodPortForward(13005, 13000).
		EventSourcePodPortForward(14005, 14000)

	defer t1.When().DeleteEventSource()
	defer t1.TerminateAllPodPortForwards()

	// Start one Sensor

	t2 := s.Given().Sensor("@testdata/sensor-trigger-spec-change.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted, util.PodLogCheckOptionWithCount(1))

	time.Sleep(3 * time.Second)

	// Trigger first dependency
	// test-dep-1
	s.e("http://localhost:12005").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful, util.PodLogCheckOptionWithCount(1))

	t2.When().DeleteSensor().Then().ExpectNoSensorPodFound()

	// Change Sensor's spec
	t2 = s.Given().Sensor("@testdata/sensor-trigger-spec-change-2.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted, util.PodLogCheckOptionWithCount(1))

	defer t2.When().DeleteSensor()

	time.Sleep(3 * time.Second)

	// test-dep-3
	s.e("http://localhost:14005").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful, util.PodLogCheckOptionWithCount(2))
	// Verify no Trigger this time since test-dep-1 should have been cleared
	t2.ExpectSensorPodLogContains(LogTriggerActionSuccessful("log-trigger-1"), util.PodLogCheckOptionWithCount(0))
}

func TestFunctionalSuite(t *testing.T) {
	suite.Run(t, new(FunctionalSuite))
}
