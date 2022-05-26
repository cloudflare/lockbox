package statemetrics

import (
	"strings"
	"testing"
	"time"

	lockboxv1 "github.com/cloudflare/lockbox/pkg/apis/lockbox.k8s.cloudflare.com/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestStateMetricsProxy_Create(t *testing.T) {
	lb := &lockboxv1.Lockbox{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "fizz",
			Name:              "buzz",
			UID:               "foobar",
			ResourceVersion:   "9001",
			CreationTimestamp: metav1.Date(2001, time.September, 9, 1, 46, 40, 0, time.UTC),
			Labels: map[string]string{
				"testing": "true",
			},
		},
		Spec: lockboxv1.LockboxSpec{
			Peer: []byte{0xDE, 0xAD, 0xBE, 0xEF},
			Template: lockboxv1.LockboxSecretTemplate{
				Type: "golang.org/testing",
			},
		},
	}
	info, created, resourceVersion, lbType, peerKey, labels := createMetricVectors(t)

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(info, created, resourceVersion, lbType, peerKey, labels)

	evt := event.CreateEvent{Object: lb}

	handler := NewStateMetricProxy(nil, info, created, resourceVersion, lbType, peerKey, labels)
	handler.Create(evt, nil)

	expected := strings.NewReader(`
# HELP kube_lockbox_info Information about Lockbox
# TYPE kube_lockbox_info gauge
kube_lockbox_info{lockbox="buzz",namespace="fizz"} 1
# HELP kube_lockbox_created Unix creation timestamp
# TYPE kube_lockbox_created gauge
kube_lockbox_created{lockbox="buzz",namespace="fizz"} 1e9
# HELP kube_lockbox_resource_version Resource version representing a specific version of a Lockbox
# TYPE kube_lockbox_resource_version gauge
kube_lockbox_resource_version{lockbox="buzz",namespace="fizz",resource_version="9001"} 1
# HELP kube_lockbox_type Lockbox secret type
# TYPE kube_lockbox_type gauge
kube_lockbox_type{lockbox="buzz",namespace="fizz",type="golang.org/testing"} 1
# HELP kube_lockbox_peer Lockbox peer key
# TYPE kube_lockbox_peer gauge
kube_lockbox_peer{lockbox="buzz",namespace="fizz",peer="deadbeef"} 1
# HELP kube_lockbox_labels Kubernetes labels converted to Prometheus labels
# TYPE kube_lockbox_labels gauge
kube_lockbox_labels{label_testing="true",lockbox="buzz",namespace="fizz"} 1
`)

	if err := testutil.GatherAndCompare(reg, expected); err != nil {
		t.Error(err)
	}
}

func TestStateMetricsProxy_Update(t *testing.T) {
	old := &lockboxv1.Lockbox{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "fizz",
			Name:              "buzz",
			UID:               "foobar",
			ResourceVersion:   "8999",
			CreationTimestamp: metav1.Now(),
			Labels: map[string]string{
				"testing": "false",
			},
		},
		Spec: lockboxv1.LockboxSpec{
			Peer: []byte{0x00},
			Template: lockboxv1.LockboxSecretTemplate{
				Type: "example.org/old",
			},
		},
	}
	lb := &lockboxv1.Lockbox{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "fizz",
			Name:              "buzz",
			UID:               "foobar",
			ResourceVersion:   "9001",
			CreationTimestamp: metav1.Date(2001, time.September, 9, 1, 46, 40, 0, time.UTC),
			Labels: map[string]string{
				"testing": "true",
			},
		},
		Spec: lockboxv1.LockboxSpec{
			Peer: []byte{0xDE, 0xAD, 0xBE, 0xEF},
			Template: lockboxv1.LockboxSecretTemplate{
				Type: "golang.org/testing",
			},
		},
	}
	info, created, resourceVersion, lbType, peerKey, labels := createMetricVectors(t)

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(info, created, resourceVersion, lbType, peerKey, labels)

	create := event.CreateEvent{Object: old}
	upd := event.UpdateEvent{
		ObjectOld: old,
		ObjectNew: lb,
	}

	handler := NewStateMetricProxy(nil, info, created, resourceVersion, lbType, peerKey, labels)
	handler.Create(create, nil)
	handler.Update(upd, nil)

	expected := strings.NewReader(`
# HELP kube_lockbox_info Information about Lockbox
# TYPE kube_lockbox_info gauge
kube_lockbox_info{lockbox="buzz",namespace="fizz"} 1
# HELP kube_lockbox_created Unix creation timestamp
# TYPE kube_lockbox_created gauge
kube_lockbox_created{lockbox="buzz",namespace="fizz"} 1e9
# HELP kube_lockbox_resource_version Resource version representing a specific version of a Lockbox
# TYPE kube_lockbox_resource_version gauge
kube_lockbox_resource_version{lockbox="buzz",namespace="fizz",resource_version="9001"} 1
# HELP kube_lockbox_type Lockbox secret type
# TYPE kube_lockbox_type gauge
kube_lockbox_type{lockbox="buzz",namespace="fizz",type="golang.org/testing"} 1
# HELP kube_lockbox_peer Lockbox peer key
# TYPE kube_lockbox_peer gauge
kube_lockbox_peer{lockbox="buzz",namespace="fizz",peer="deadbeef"} 1
# HELP kube_lockbox_labels Kubernetes labels converted to Prometheus labels
# TYPE kube_lockbox_labels gauge
kube_lockbox_labels{label_testing="true",lockbox="buzz",namespace="fizz"} 1
`)

	if err := testutil.GatherAndCompare(reg, expected); err != nil {
		t.Error(err)
	}
}

func TestStateMetricsProxy_Delete(t *testing.T) {
	lb := &lockboxv1.Lockbox{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "fizz",
			Name:              "buzz",
			UID:               "foobar",
			ResourceVersion:   "9001",
			CreationTimestamp: metav1.Date(2001, time.September, 9, 1, 46, 40, 0, time.UTC),
			Labels: map[string]string{
				"testing": "true",
			},
		},
		Spec: lockboxv1.LockboxSpec{
			Peer: []byte{0xDE, 0xAD, 0xBE, 0xEF},
			Template: lockboxv1.LockboxSecretTemplate{
				Type: "golang.org/testing",
			},
		},
	}
	info, created, resourceVersion, lbType, peerKey, labels := createMetricVectors(t)

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(info, created, resourceVersion, lbType, peerKey, labels)

	create := event.CreateEvent{Object: lb}
	deleted := event.DeleteEvent{
		Object:             lb,
		DeleteStateUnknown: false,
	}

	handler := NewStateMetricProxy(nil, info, created, resourceVersion, lbType, peerKey, labels)
	handler.Create(create, nil)
	handler.Delete(deleted, nil)

	expected := &strings.Reader{}

	if err := testutil.GatherAndCompare(reg, expected); err != nil {
		t.Error(err)
	}
}

func createMetricVectors(t *testing.T) (info, created, resourceVersion, lbType, peerKey *KubernetesVec, labels *LabelsVec) {
	info = NewKubernetesVec(KubernetesOpts{
		Name: "kube_lockbox_info",
		Help: "Information about Lockbox",
	}, []string{"namespace", "lockbox"})
	created = NewKubernetesVec(KubernetesOpts{
		Name: "kube_lockbox_created",
		Help: "Unix creation timestamp",
	}, []string{"namespace", "lockbox"})
	resourceVersion = NewKubernetesVec(KubernetesOpts{
		Name: "kube_lockbox_resource_version",
		Help: "Resource version representing a specific version of a Lockbox",
	}, []string{"namespace", "lockbox", "resource_version"})
	lbType = NewKubernetesVec(KubernetesOpts{
		Name: "kube_lockbox_type",
		Help: "Lockbox secret type",
	}, []string{"namespace", "lockbox", "type"})
	peerKey = NewKubernetesVec(KubernetesOpts{
		Name: "kube_lockbox_peer",
		Help: "Lockbox peer key",
	}, []string{"namespace", "lockbox", "peer"})
	labels = NewLabelsVec(KubernetesOpts{
		Name: "kube_lockbox_labels",
		Help: "Kubernetes labels converted to Prometheus labels",
	})

	return
}
