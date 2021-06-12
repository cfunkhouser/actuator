package actuator

import (
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type handlerConfig struct {
	Path   string `json:"path" yaml:"path"`
	Token  string `json:"token" yaml:"token"`
	Action string `json:"action" yaml:"action"`
}

type actionConfig struct {
	Name    string `json:"name" yaml:"name"`
	Command string `json:"command" yaml:"command"`
}

type actuatorConfig struct {
	Handlers []handlerConfig `json:"handlers" yaml:"handlers"`
	Actions  []actionConfig  `json:"actions" yaml:"actions"`
}

func FromConfig(r io.Reader) (http.Handler, error) {
	mux := http.NewServeMux()
	mux.Handle("/", DefaultHandler())

	var c actuatorConfig
	if err := yaml.NewDecoder(r).Decode(&c); err != nil {
		return mux, err
	}

	actions := make(map[string]Action)
	for _, ac := range c.Actions {
		if ac.Command == "" {
			logrus.WithField("action", ac.Name).Info("action has no command, and so will just be logged")
			actions[ac.Name] = &LogAction{}
			continue
		}
		cmd, err := Command(ac.Command)
		if err != nil {
			return mux, err
		}
		actions[ac.Name] = cmd
	}

	for _, hc := range c.Handlers {
		if a, has := actions[hc.Action]; has {
			var opts []Option
			if hc.Token != "" {
				opts = append(opts, WithToken(hc.Token))
			}
			h := Handle(hc.Path, a, opts...)
			mux.Handle(hc.Path, h)
		}
	}

	return mux, nil
}
