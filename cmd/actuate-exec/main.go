// actuate-exec executes a command on the local system when an alert matching
// configured conditions is fired.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/cfunkhouser/actuator"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func openConfigFile(c *cli.Context) (io.ReadCloser, error) {
	cfp := c.String("config.file")
	if cfp == "" {
		return nil, cli.Exit("config.file is empty", 1)
	}
	f, err := os.Open(cfp)
	if err != nil {
		return nil, cli.Exit(err, 1)
	}
	return f, nil
}

func serveExporter(c *cli.Context) error {
	cf, err := openConfigFile(c)
	if err != nil {
		return err
	}
	handler, err := actuator.FromConfig(cf)
	if err != nil {
		return cli.Exit(err, 1)
	}
	return http.ListenAndServe(c.String("server.address"), handler)
}

const (
	defaultActuatorAddress = "0.0.0.0:9942"
	defaultConfigFile      = "/etc/actuator/actuator.yml"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	app := &cli.App{
		Name:  "actuate-exec",
		Usage: "Execute commands in response to fired alerts",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "server.address",
				Aliases: []string{"a"},
				Value:   defaultActuatorAddress,
				Usage:   "ip:port from which to serve Prometheus metrics",
			},
			&cli.StringFlag{
				Name:    "config.file",
				Aliases: []string{"c"},
				Value:   defaultConfigFile,
				Usage:   "actuator config file location",
			},
		},
		Action: serveExporter,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
