package providers

import (
	"context"
	"net/http"

	"github.com/urfave/cli"
	t "github.com/webtor-io/node-manager/services/types"
)

type HetznerRobot struct {
	cl *http.Client
}

func NewHetznerRobot(c *cli.Context, cl *http.Client) *HetznerRobot {
	return &HetznerRobot{
		cl: cl,
	}
}

func (s *HetznerRobot) Reboot(ctx context.Context, n *t.Node) error {
	return nil
}
