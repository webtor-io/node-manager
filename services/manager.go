package services

import (
	"context"
	"net/http"
	"sync"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	p "github.com/webtor-io/node-manager/services/providers"
	t "github.com/webtor-io/node-manager/services/types"
)

var (
	ProviderNotFoundErr = errors.New("provider not found")
)

type Provider interface {
	Reboot(ctx context.Context, n *t.Node) error
}

type Manager struct {
	providers map[string]Provider
	once      sync.Once
}

func NewManager() *Manager {
	return &Manager{
		providers: map[string]Provider{},
	}
}

func (s *Manager) AddProvider(n string, p Provider) {
	s.providers[n] = p
}

func (s *Manager) getProvider(n *t.Node) (Provider, error) {
	p, ok := s.providers[n.Provider]
	if !ok {
		return nil, ProviderNotFoundErr
	}
	return p, nil
}

func (s *Manager) Reboot(ctx context.Context, n *t.Node) error {
	p, err := s.getProvider(n)
	if err != nil {
		return err
	}
	return p.Reboot(ctx, n)
}

func (s *Manager) initProviders(c *cli.Context, cl *http.Client) {
	contabo := p.NewContabo(c, cl)
	if contabo != nil {
		s.AddProvider("contabo", contabo)
	}

	hetznerRobot := p.NewHetznerRobot(c, cl)
	if hetznerRobot != nil {
		s.AddProvider("hetzner_robot", contabo)
	}
}

func (s *Manager) LoadProviders(c *cli.Context, cl *http.Client) {
	s.once.Do(func() {
		s.initProviders(c, cl)
	})
}
