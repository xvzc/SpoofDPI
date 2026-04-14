package server

type ConfigurationJob struct {
	Apply func() error
	Reset func() error
}
