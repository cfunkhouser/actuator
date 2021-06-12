package actuator

import (
	"time"
)

type AlertStatus string
type LabelKey string
type LabelValue string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
)

type Alert struct {
	Status      AlertStatus             `json:"status"`
	Labels      map[LabelKey]LabelValue `json:"labels"`
	Annotations map[string]string       `json:"annotations"`
	Start       time.Time               `json:"startsAt"`
	End         time.Time               `json:"endsAt"`
	Generator   string                  `json:"generatorURL"`
	Fingerprint string                  `json:"fingerprint"`
}

type AlermanagerWebhookPayload struct {
	Version           string                  `json:"version"`
	GroupKey          string                  `json:"groupKey"`
	TruncatedAlerts   int                     `json:"truncatedAlerts"`
	Status            AlertStatus             `json:"status"`
	Receiver          string                  `json:"receiver"`
	GroupLabels       map[LabelKey]LabelValue `json:"groupLabels"`
	CommonLabels      map[LabelKey]LabelValue `json:"commonLabels"`
	CommonAnnotations map[string]string       `json:"commonAnnotations"`
	Alertmanager      string                  `json:"externalURL"`
	Alerts            []*Alert                `json:"alerts"`
}
