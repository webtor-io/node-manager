package main

import (
	"context"
	"net/http"
	"time"

	"github.com/urfave/cli"
	s "github.com/webtor-io/node-manager/services"

	log "github.com/sirupsen/logrus"
)

func makeHealCMD() cli.Command {
	healCmd := cli.Command{
		Name:    "heal",
		Aliases: []string{"h"},
		Usage:   "reboots all NotReady nodes",
		Action:  heal,
	}
	configureHeal(&healCmd)
	return healCmd
}

func configureHeal(c *cli.Command) {
	configureProviders(c)
}

func heal(c *cli.Context) error {
	k8s := s.NewK8s()

	cl := &http.Client{}

	manager := s.NewManager()

	manager.LoadProviders(c, cl)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5)*time.Minute)
	defer cancel()

	log.Info("checking NotReady nodes")

	nodes, err := k8s.GetNotReadyNodes(ctx)
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		log.Info("nothing to heal")
		return nil
	}

	for _, n := range nodes {
		log.Infof("node %v not ready", n.Name)
		log.Infof("rebooting %v", n.Name)
		err := manager.Reboot(ctx, &n)
		if err != nil {
			log.WithError(err).Warnf("failed to heal %v", n.Name)
		} else {
			log.Infof("done rebooting %v", n.Name)
		}
	}

	return nil
}
