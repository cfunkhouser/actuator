// actuate-exec executes a command on the local system when an alert matching
// configured conditions is fired.
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/cfunkhouser/actuator"
	"github.com/cfunkhouser/actuator/internal"
	"github.com/cfunkhouser/httpdumper"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type echoActor struct{}

func (*echoActor) ActOn(alert *actuator.Alert) error {
	logrus.WithField("alertname", alert.Labels["alertname"]).Debug("hello I am an actor and I am acting!")
	return nil
}

func serveExporter(c *cli.Context) error {
	h, err := actuator.New(
		actuator.Do(
			[]actuator.Actor{&echoActor{}},
			actuator.WhenAlertHasLabels([]*internal.Label{
				{Key: "site", Value: "context-switch"},
				{Key: "severity", Value: "critical"},
			})))
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("/alerts", h)
	return http.ListenAndServe(c.String("server.address"), httpdumper.LogWrap(mux))
}

const defaultActuatorAddress = "0.0.0.0:9942"

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	app := &cli.App{
		Name:  "actuate-exec",
		Usage: "Execute commands in response to fired alerts",
		Flags: []cli.Flag{&cli.StringFlag{
			Name:    "server.address",
			Aliases: []string{"a"},
			Value:   defaultActuatorAddress,
			Usage:   "ip:port from which to serve Prometheus metrics",
		}},
		Action: serveExporter,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
