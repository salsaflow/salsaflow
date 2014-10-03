package common

type Action interface {
	Rollback() error
}

type ActionFunc func() error

func (action ActionFunc) Rollback() error {
	return action()
}
