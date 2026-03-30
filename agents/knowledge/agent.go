package knowledge

type Agent struct {
}

func NewAgent() *Agent {
	return &Agent{}
}

func (a *Agent) SaveMessage(groupID int64, userID int64, text string) error {
	return nil
}

func (a *Agent) Query(keyword string) ([]string, error) {
	return nil, nil
}
