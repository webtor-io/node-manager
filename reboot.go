package main

import (
	"context"
	"net/http"
	"time"

	"github.com/pkg/errors"
	s "github.com/webtor-io/node-manager/services"

	"github.com/urfave/cli"

	log "github.com/sirupsen/logrus"
)

func makeRebootCMD() cli.Command {
	rebootCMD := cli.Command{
		Name:    "reboot",
		Aliases: []string{"r"},
		Usage:   "reboots node",
		Action:  reboot,
	}
	configureReboot(&rebootCMD)
	return rebootCMD
}

func configureReboot(c *cli.Command) {
	configureProviders(c)
}

func reboot(c *cli.Context) error {
	name := c.Args().Get(0)
	if name != "" {
		log.Infof("rebooting %v", name)
	} else {
		return errors.New("node name must be provided")
	}
	k8s := s.NewK8s()

	cl := &http.Client{}

	manager := s.NewManager()

	manager.LoadProviders(c, cl)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5)*time.Minute)
	defer cancel()

	n, err := k8s.GetNodeByName(ctx, name)

	if err != nil {
		return err
	}

	err = manager.Reboot(ctx, n)
	if err != nil {
		log.WithError(err).Warnf("failed to reboot %v", n.Name)
	} else {
		log.Infof("done rebooting %v", name)
	}

	return nil
}
