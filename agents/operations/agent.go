package operations

type Agent struct {
}

func NewAgent() *Agent {
	return &Agent{}
}

func (a *Agent) RecordError(err error) {
}

func (a *Agent) GetReport() string {
	return ""
}
