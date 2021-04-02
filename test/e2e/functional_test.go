// +build functional

package e2e

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/argoproj/argo-events/test/e2e/fixtures"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/suite"
)

type FunctionalSuite struct {
	fixtures.E2ESuite
}

const (
	LogEventSourceStarted      = "Eventing server started."
	LogSensorStarted           = "Sensor started."
	LogPublishEventSuccessful  = "succeeded to publish an event"
	LogTriggerActionSuccessful = "successfully processed the trigger"
	LogTriggerActionFailed     = "failed to trigger action"
)

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
	s.Given().EventSource("@testdata/es-calendar.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady().
		Then().
		ExpectEventSourcePodLogContains(LogPublishEventSuccessful)

	s.Given().Sensor("@testdata/sensor-log.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogTriggerActionSuccessful)
}

func (s *FunctionalSuite) TestMetricsWithCalendar() {
	t1 := s.Given().EventSource("@testdata/es-calendar.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady().
		Then().
		ExpectEventSourcePodLogContains(LogEventSourceStarted).
		EventSourcePodPortForward(7777, 7777)

	defer t1.TerminateAllPodPortForwards()

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful)

	// EventSource POD metrics
	s.e("http://localhost:7777").GET("/metrics").
		Expect().
		Status(200).
		Body().
		Contains("argo_events_event_service_running_total").
		Contains("argo_events_events_sent_total").
		Contains("argo_events_event_processing_duration_milliseconds")

	t2 := s.Given().Sensor("@testdata/sensor-log.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted).
		SensorPodPortForward(7778, 7777)

	defer t2.TerminateAllPodPortForwards()

	t2.ExpectSensorPodLogContains(LogTriggerActionSuccessful)

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
		ExpectEventSourcePodLogContains(LogEventSourceStarted).
		EventSourcePodPortForward(12000, 12000).
		EventSourcePodPortForward(7777, 7777)

	defer t1.TerminateAllPodPortForwards()

	t2 := s.Given().Sensor("@testdata/sensor-test-metrics.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted).
		SensorPodPortForward(7778, 7777)

	defer t2.TerminateAllPodPortForwards()

	s.e("http://localhost:12000").POST("/example").WithBytes([]byte("{}")).
		Expect().
		Status(200)

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful)

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
	t2.ExpectSensorPodLogContains(LogTriggerActionSuccessful).
		ExpectSensorPodLogContains(LogTriggerActionFailed)

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
		ExpectEventSourcePodLogContains(LogEventSourceStarted)

	t2 := s.Given().Sensor("@testdata/sensor-resource.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		Then().
		ExpectSensorPodLogContains(LogSensorStarted)

	w1.Exec("kubectl", []string{"-n", fixtures.Namespace, "delete", "pod", "test-pod"}, fixtures.OutputRegexp(`pod "test-pod" deleted`))

	t1.ExpectEventSourcePodLogContains(LogPublishEventSuccessful)

	t2.ExpectSensorPodLogContains(LogTriggerActionSuccessful)
}

func TestFunctionalSuite(t *testing.T) {
	suite.Run(t, new(FunctionalSuite))
}
