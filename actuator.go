package actuator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cfunkhouser/actuator/internal"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
)

type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
)

type Alert struct {
	Status      AlertStatus                               `json:"status"`
	Labels      map[internal.LabelKey]internal.LabelValue `json:"labels"`
	Annotations map[string]string                         `json:"annotations"`
	Start       time.Time                                 `json:"startsAt"`
	End         time.Time                                 `json:"endsAt"`
	Generator   string                                    `json:"generatorURL"`
	Fingerprint string                                    `json:"fingerprint"`
}

type AlermanagerWebhookPayload struct {
	Version           string                                    `json:"version"`
	GroupKey          string                                    `json:"groupKey"`
	TruncatedAlerts   int                                       `json:"truncatedAlerts"`
	Status            AlertStatus                               `json:"status"`
	Receiver          string                                    `json:"receiver"`
	GroupLabels       map[internal.LabelKey]internal.LabelValue `json:"groupLabels"`
	CommonLabels      map[internal.LabelKey]internal.LabelValue `json:"commonLabels"`
	CommonAnnotations map[string]string                         `json:"commonAnnotations"`
	Alertmanager      string                                    `json:"externalURL"`
	Alerts            []*Alert                                  `json:"alerts"`
}

type Actor interface {
	ActOn(*Alert) error
}

type plan struct {
	actions *internal.Trie
}

type Condition func(*plan, []Actor) error

func WhenAlertHasLabels(labels []*internal.Label) Condition {
	return func(plan *plan, acts []Actor) error {
		var ls internal.LabelSet
		if err := ls.Add(labels...); err != nil {
			return err
		}
		plan.actions.Insert(ls, acts)
		return nil
	}
}

type ActionPlan func(*plan) error

func Do(what []Actor, when ...Condition) ActionPlan {
	return func(plan *plan) error {
		for _, c := range when {
			if err := c(plan, what); err != nil {
				return err
			}
		}
		return nil
	}
}

func (p *plan) HandlePayload(ctx context.Context, payload *AlermanagerWebhookPayload) error {
	var merr error
	var payloadLS internal.LabelSet
	payloadLS.AccumulateMap(payload.CommonLabels)
	payloadLS.AccumulateMap(payload.GroupLabels)
	for _, alert := range payload.Alerts {
		ls := payloadLS.Copy()
		ls.AccumulateMap(alert.Labels)
		alertKey := ls.String()
		logrus.WithField("alert_key", alertKey).Debug("attempting to handle alert")
		if v := p.actions.Get(ls); len(v) != 0 {
			for _, ia := range v {
				if act, ok := ia.(Actor); ok {
					if err := act.ActOn(alert); err != nil {
						merr = multierror.Append(merr, err)
						break
					}
				} else {
					logrus.WithField("alert_key", alertKey).Warnf("got a non-Actor out of the plan: %+v", ia)
				}
			}
		} else {
			logrus.WithField("alert_key", alertKey).Debug("nothing matched alert")
		}
	}
	return merr
}

func fail(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(msg))
}

type v4Handler struct {
	plan *plan
}

func (h *v4Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		logrus.WithFields(logrus.Fields{
			"client":           r.RemoteAddr,
			"bad_content_type": ct,
		}).Warn("got unexpected Content-Type from client")
		fail(w, fmt.Sprintf("Not sure what to do with Content-Type %q", ct), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	var payload AlermanagerWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logrus.WithError(err).WithField("client", r.RemoteAddr).Warn("failed receiving payload from client")
		fail(w, fmt.Sprintf("Something unexpected happened reading the payload: %v", err), http.StatusBadRequest)
		return
	}
	if v := payload.Version; v != "4" {
		logrus.WithFields(logrus.Fields{
			"client":      r.RemoteAddr,
			"bad_version": v,
		}).Warn("got unexpected payload version from client")
		fail(w, fmt.Sprintf("Unexpected payload version %q from Alertmanager", v), http.StatusBadRequest)
		return
	}
	if err := h.plan.HandlePayload(r.Context(), &payload); err != nil {
		logrus.WithError(err).Warn("errors while processing")
	}
}

func New(plans ...ActionPlan) (http.Handler, error) {
	plan := &plan{
		actions: internal.NewTrie(),
	}
	for _, p := range plans {
		if err := p(plan); err != nil {
			return nil, err
		}
	}
	return &v4Handler{plan: plan}, nil
}
