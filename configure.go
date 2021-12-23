package main

import (
	"github.com/urfave/cli"
	p "github.com/webtor-io/node-manager/services/providers"
)

func configure(app *cli.App) {
	healCmd := makeHealCMD()
	rebootCmd := makeRebootCMD()
	app.Commands = []cli.Command{
		healCmd,
		rebootCmd,
	}
}

func configureProviders(c *cli.Command) {
	c.Flags = p.RegisterContaboFlags(c.Flags)
}
