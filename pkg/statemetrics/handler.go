package statemetrics

import (
	"context"
	"encoding/hex"

	lockboxv1 "github.com/cloudflare/lockbox/pkg/apis/lockbox.k8s.cloudflare.com/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	_ "sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// StateMetricProxy updates state metrics by implementing the [handler.EventHandler] interface.
type StateMetricProxy struct {
	info            *KubernetesVec
	created         *KubernetesVec
	resourceVersion *KubernetesVec
	lbType          *KubernetesVec
	peerKey         *KubernetesVec
	labels          *LabelsVec
}

// NewStateMetricProxy returns a StateMetricsProxy. All metrics must be non-nil.
func NewStateMetricProxy(info, created, resourceVersion, lbType, peerKey *KubernetesVec, labels *LabelsVec) *StateMetricProxy {
	return &StateMetricProxy{
		info:            info,
		created:         created,
		resourceVersion: resourceVersion,
		lbType:          lbType,
		peerKey:         peerKey,
		labels:          labels,
	}
}

// Create implements EventHandler.
func (s *StateMetricProxy) Create(ctx context.Context, evt event.CreateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	s.updateWith(evt.Object)
}

// Update implements EventHandler.
func (s *StateMetricProxy) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	s.updateWith(evt.ObjectNew)
}

// Delete implements EventHandler.
func (s *StateMetricProxy) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	uid := evt.Object.GetUID()

	s.info.Delete(uid)
	s.created.Delete(uid)
	s.resourceVersion.Delete(uid)
	s.lbType.Delete(uid)
	s.peerKey.Delete(uid)
	s.labels.Delete(uid)
}

// Generic implements EventHandler.
func (s *StateMetricProxy) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
}

// updateWith updates the metrics for Create and Update handles.
func (s *StateMetricProxy) updateWith(obj client.Object) {
	namespace := obj.GetNamespace()
	lockbox := obj.GetName()
	uid := obj.GetUID()

	s.info.WithLabelValues(uid, namespace, lockbox).Set(1)
	creationTime := obj.GetCreationTimestamp()
	if !creationTime.IsZero() {
		s.created.WithLabelValues(uid, namespace, lockbox).Set(float64(creationTime.Unix()))
	}
	s.resourceVersion.WithLabelValues(uid, namespace, lockbox, obj.GetResourceVersion()).Set(1)

	if lb, ok := obj.(*lockboxv1.Lockbox); ok {
		s.lbType.WithLabelValues(uid, namespace, lockbox, string(lb.Spec.Template.Type)).Set(1)
		s.peerKey.WithLabelValues(uid, namespace, lockbox, hex.EncodeToString(lb.Spec.Peer)).Set(1)
	}

	promLabels := kubernetesLabelsToPrometheusLabels(obj.GetLabels())
	promLabels["namespace"] = namespace
	promLabels["lockbox"] = lockbox

	s.labels.With(uid, promLabels).Set(1)
}
