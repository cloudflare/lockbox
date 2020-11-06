package statemetrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/types"
)

// Kubernetes is a metric tracking a numerical value that can arbitrary go up and down.
type Kubernetes interface {
	prometheus.Metric
	prometheus.Collector

	// Set sets the tracked number to an arbitrary value.
	Set(float64)
}

// kubernetes implements Kubernetes.
type kubernetes struct {
	prometheus.Metric
	values []string
	desc   *prometheus.Desc
}

// Set creates a new constant metric for this numerical value
func (k *kubernetes) Set(val float64) {
	k.Metric = prometheus.MustNewConstMetric(k.desc, prometheus.GaugeValue, val, k.values...)
}

// Describe implements Collector
func (k *kubernetes) Describe(ch chan<- *prometheus.Desc) {
	ch <- k.desc
}

// Collect implements Collector
func (k *kubernetes) Collect(ch chan<- prometheus.Metric) {
	ch <- k.Metric
}

// KubernetesVec is a Collector that bundles a set of Kubernetes metrics that all share the same Desc,
// but have different values for their variable labels. This is used if you want to count the same
// thing partitioned by various dimensions (e.g., namespaces, types). Create instances with
// NewKubernetesVec.
type KubernetesVec struct {
	desc    *prometheus.Desc
	metrics map[types.UID]Kubernetes
	mu      sync.Mutex
}

// KubernetesOpts is an alias for Opts.
type KubernetesOpts prometheus.Opts

// NewKubernetesVec creates a new KubernetesVec based on the provided KubernetesOpts and partitioned
// by the given label names.
func NewKubernetesVec(opts KubernetesOpts, labelNames []string) *KubernetesVec {
	desc := prometheus.NewDesc(
		prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name),
		opts.Help,
		labelNames,
		opts.ConstLabels,
	)

	return &KubernetesVec{
		desc:    desc,
		metrics: make(map[types.UID]Kubernetes),
	}
}

// WithLabelValues returns the Kubenetes metric for the given slice of label values (in the same order
// as the variable labels).
//
// Consecutive calls for the same uid replace earlier metrics.
func (v *KubernetesVec) WithLabelValues(uid types.UID, lvs ...string) Kubernetes {
	k := &kubernetes{
		values: lvs,
		desc:   v.desc,
	}
	k.Set(0)

	v.mu.Lock()
	v.metrics[uid] = k
	v.mu.Unlock()

	return k
}

// Delete deletes the metric stored for this uid.
func (v *KubernetesVec) Delete(uid types.UID) {
	v.mu.Lock()
	defer v.mu.Unlock()

	delete(v.metrics, uid)
}

// Describe implements Collector.
func (v *KubernetesVec) Describe(ch chan<- *prometheus.Desc) {
	ch <- v.desc
}

// Collect implements Collector.
func (v *KubernetesVec) Collect(ch chan<- prometheus.Metric) {
	v.mu.Lock()
	defer v.mu.Unlock()

	for _, metric := range v.metrics {
		ch <- metric
	}
}

// LabelsVec is a Collector that bundles a set of unchecked Kubernetes metrics that all share the
// same Desc, but have different labels. This is used if you want to count the same thing
// partitioned by unbounded dimensions (e.g., Kubernetes labels). Create instances with
// NewLabelsVec.
type LabelsVec struct {
	opts    KubernetesOpts
	metrics map[types.UID]Kubernetes
	mu      sync.Mutex
}

// NewLabelsVec creates a new LabelsVec based on the provided KubernetesOpts.
func NewLabelsVec(opts KubernetesOpts) *LabelsVec {
	return &LabelsVec{
		opts:    opts,
		metrics: make(map[types.UID]Kubernetes),
	}
}

// With returns the Kubernetes metric for the given Labels map.
//
// Consecutive calls with the same uid replace earlier metrics.
func (v *LabelsVec) With(uid types.UID, l prometheus.Labels) Kubernetes {
	labels := make([]string, 0, len(l))
	values := make([]string, 0, len(l))

	for label, value := range l {
		labels = append(labels, label)
		values = append(values, value)
	}

	desc := prometheus.NewDesc(
		prometheus.BuildFQName(v.opts.Namespace, v.opts.Subsystem, v.opts.Name),
		v.opts.Help,
		labels,
		v.opts.ConstLabels,
	)

	k := &kubernetes{
		values: values,
		desc:   desc,
	}
	k.Set(0)

	v.mu.Lock()
	v.metrics[uid] = k
	v.mu.Unlock()

	return k
}

// Delete deletes the metric stored for this uid.
func (v *LabelsVec) Delete(uid types.UID) {
	v.mu.Lock()
	defer v.mu.Unlock()

	delete(v.metrics, uid)
}

// Describe implements Collector. No Desc are sent on the channel to make this
// an unchecked collector.
func (v *LabelsVec) Describe(chan<- *prometheus.Desc) {}

// Collect implements Collector.
func (v *LabelsVec) Collect(ch chan<- prometheus.Metric) {
	v.mu.Lock()
	defer v.mu.Unlock()

	for _, metric := range v.metrics {
		ch <- metric
	}
}
