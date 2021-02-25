// +build functional

package e2e

import (
	"testing"

	"github.com/argoproj/argo-events/test/e2e/fixtures"
	"github.com/stretchr/testify/suite"
)

type FunctionalSuite struct {
	fixtures.E2ESuite
}

func (s *FunctionalSuite) TestCreateCalendarEventSource() {
	s.Given().EventSource("@testdata/calendar-eventsource.yaml").
		When().
		CreateEventSource().
		WaitForEventSourceReady().
		WaitForEventSourceDeploymentReady().
		Then().
		ExpectEventSourcePodLogContains("succeeded to publish an event")

	s.Given().Sensor("@testdata/log-sensor.yaml").
		When().
		CreateSensor().
		WaitForSensorReady().
		WaitForSensorDeploymentReady().
		Then().
		ExpectSensorPodLogContains("successfully processed the trigger")
}

func TestFunctionalSuite(t *testing.T) {
	suite.Run(t, new(FunctionalSuite))
}
