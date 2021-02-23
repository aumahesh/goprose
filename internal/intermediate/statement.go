package intermediate

type GuardedStatement struct {
}

func NewStatement() (*GuardedStatement, error) {
	s := &GuardedStatement{}
	return s, nil
}
