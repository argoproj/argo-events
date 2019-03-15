package common

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	"testing"
)

type fakeEventSource struct {
	Msg string `json:"msg"`
}

func TestValidateGatewayEventSource(t *testing.T) {
	convey.Convey("Given an event source, validate it", t, func() {
		esStr := `
msg: "hello"
`
		es, err := ValidateGatewayEventSource(esStr, func(s string) (i interface{}, e error) {
			t := &fakeEventSource{}
			err := yaml.Unmarshal([]byte(esStr), t)
			return t, err
		}, func(i interface{}) error {
			t, ok := i.(*fakeEventSource)
			if !ok {
				return fmt.Errorf("failed to cast to fake event source")
			}
			if t == nil {
				return fmt.Errorf("event source cant be nil")
			}
			if t.Msg == "" {
				return fmt.Errorf("msg can't be empty")
			}
			return nil
		})

		convey.So(err, convey.ShouldBeNil)
		convey.So(es, convey.ShouldNotBeNil)
		convey.So(es.IsValid, convey.ShouldEqual, true)
	})
}
