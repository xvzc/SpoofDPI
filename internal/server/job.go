package server

type ConfigurationJob struct {
	Up   func() error
	Down func() error
}
