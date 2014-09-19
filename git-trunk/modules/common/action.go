package common

type ActionFunc func() error

func (action ActionFunc) Rollback() error {
	return action()
}
