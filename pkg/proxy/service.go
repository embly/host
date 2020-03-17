package proxy

type Service interface {
	IsAlive() bool
	Name() string
}

type defaultService struct {
	name string
	// TODO: alive?
}
