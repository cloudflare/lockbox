package statemetrics

import (
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
)

var invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// sanitizeLabel replaces non-alphanumeric characters with underscores.
func sanitizeLabel(l string) string {
	return invalidLabelCharRE.ReplaceAllString(l, "_")
}

// kubernetesLabelsToPrometheusLabels generates Prometheus-safe labels from
// the resource's Kubernetes labels.
func kubernetesLabelsToPrometheusLabels(labels map[string]string) prometheus.Labels {
	promLabels := map[string]string{}
	for l, v := range labels {
		promLabels["label_"+sanitizeLabel(l)] = v
	}

	return promLabels
}
