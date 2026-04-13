package server

type ConfigurationJob struct {
	Set   func() error
	Unset func() error
}
