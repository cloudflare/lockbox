package statemetrics

import (
	"encoding/hex"

	lockboxv1 "github.com/cloudflare/lockbox/pkg/apis/lockbox.k8s.cloudflare.com/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

// StateMetricProxy updates state metrics by intercepting events as they are passed to event handlers.
type StateMetricProxy struct {
	enqueuer handler.EventHandler

	info            *KubernetesVec
	created         *KubernetesVec
	resourceVersion *KubernetesVec
	lbType          *KubernetesVec
	peerKey         *KubernetesVec
	labels          *LabelsVec
}

// NewStateMetricProxy returns a StateMetricsProxy. All metrics must be non-nil.
func NewStateMetricProxy(enqueuer handler.EventHandler, info, created, resourceVersion, lbType, peerKey *KubernetesVec, labels *LabelsVec) *StateMetricProxy {
	return &StateMetricProxy{
		enqueuer:        enqueuer,
		info:            info,
		created:         created,
		resourceVersion: resourceVersion,
		lbType:          lbType,
		peerKey:         peerKey,
		labels:          labels,
	}
}

// Create implements EventHandler.
func (s *StateMetricProxy) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	s.updateWith(evt.Meta, evt.Object)

	if s.enqueuer != nil {
		s.enqueuer.Create(evt, q)
	}
}

// Update implements EventHandler.
func (s *StateMetricProxy) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	s.updateWith(evt.MetaNew, evt.ObjectNew)

	if s.enqueuer != nil {
		s.enqueuer.Update(evt, q)
	}
}

// Delete implements EventHandler.
func (s *StateMetricProxy) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	uid := evt.Meta.GetUID()

	s.info.Delete(uid)
	s.created.Delete(uid)
	s.resourceVersion.Delete(uid)
	s.lbType.Delete(uid)
	s.peerKey.Delete(uid)
	s.labels.Delete(uid)

	if s.enqueuer != nil {
		s.enqueuer.Delete(evt, q)
	}
}

// Generic implements EventHandler.
func (s *StateMetricProxy) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if s.enqueuer != nil {
		s.enqueuer.Generic(evt, q)
	}
}

// updateWith updates the metrics for Create and Update handles.
func (s *StateMetricProxy) updateWith(meta metav1.Object, obj runtime.Object) {
	namespace := meta.GetNamespace()
	lockbox := meta.GetName()
	uid := meta.GetUID()

	s.info.WithLabelValues(uid, namespace, lockbox).Set(1)
	creationTime := meta.GetCreationTimestamp()
	if !creationTime.IsZero() {
		s.created.WithLabelValues(uid, namespace, lockbox).Set(float64(creationTime.Unix()))
	}
	s.resourceVersion.WithLabelValues(uid, namespace, lockbox, meta.GetResourceVersion()).Set(1)

	if lb, ok := obj.(*lockboxv1.Lockbox); ok {
		s.lbType.WithLabelValues(uid, namespace, lockbox, string(lb.Spec.Template.Type)).Set(1)
		s.peerKey.WithLabelValues(uid, namespace, lockbox, hex.EncodeToString(lb.Spec.Peer)).Set(1)
	}

	promLabels := kubernetesLabelsToPrometheusLabels(meta.GetLabels())
	promLabels["namespace"] = namespace
	promLabels["lockbox"] = lockbox

	s.labels.With(uid, promLabels).Set(1)
}
