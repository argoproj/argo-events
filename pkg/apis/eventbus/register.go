package eventbus

const (
	// Group is the API Group
	Group string = "argoproj.io"

	// EventSource constants
	Kind     string = "EventBus"
	Singular string = "eventbus"
	Plural   string = "eventbuses"
	FullName string = Plural + "." + Group
)
