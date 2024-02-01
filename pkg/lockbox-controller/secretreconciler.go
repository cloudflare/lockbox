package controller

import (
	"context"
	"encoding/base64"
	"fmt"

	lockboxv1 "github.com/cloudflare/lockbox/pkg/apis/lockbox.k8s.cloudflare.com/v1"
	"github.com/cloudflare/lockbox/pkg/util/conditions"
	"github.com/kevinburke/nacl"
	"github.com/kevinburke/nacl/box"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//go:generate controller-gen rbac:roleName=lockbox-controller paths=./. output:rbac:artifacts:config=../../deployment/rbac

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;patch;update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="lockbox.k8s.cloudflare.com",resources=lockboxes,verbs=get;list;watch
// +kubebuilder:rbac:groups="lockbox.k8s.cloudflare.com",resources=lockboxes/status,verbs=get;update;patch

const keySize = nacl.KeySize

// SecretReconcilerOption allows for functional options to modify the SecretReconciler
type SecretReconcilerOption func(s *SecretReconciler)

// SecretReconciler implements the reconciliation logic for Lockbox secrets.
type SecretReconciler struct {
	pubKey, priKey nacl.Key

	client   client.Client
	recorder record.EventRecorder
}

// NewSecretReconciler creates a reconciler controller for the provided keypair and options.
//
// If not mutated by any options, the reconciler uses a noop API client and events recorder.
func NewSecretReconciler(pubKey, priKey nacl.Key, options ...SecretReconcilerOption) *SecretReconciler {
	sr := &SecretReconciler{
		pubKey:   pubKey,
		priKey:   priKey,
		client:   clientfake.NewClientBuilder().Build(),
		recorder: &record.FakeRecorder{},
	}

	for _, opt := range options {
		opt(sr)
	}

	return sr
}

// Reconcile implements reconcile.Reconciler by ensuring Lockbox controlled Secrets are as described.
func (s *SecretReconciler) Reconcile(ctx context.Context, lb *lockboxv1.Lockbox) (reconcile.Result, error) {
	if len(lb.Spec.Sender) != keySize {
		msg := fmt.Sprintf("invalid sender key length, got %d wanted %d", len(lb.Spec.Sender), keySize)

		s.recorder.Eventf(lb, "Warning", "InvalidKeyLength", msg)
		conditions.Set(lb, conditions.FalseCondition(lockboxv1.ReadyCondition, "InvalidKeyLength", lockboxv1.ConditionSeverityError, msg))
		_ = s.client.Status().Update(ctx, lb)
		return reconcile.Result{}, fmt.Errorf("incorrect sender key length: %d, should be %d", len(lb.Spec.Sender), keySize)
	}
	if len(lb.Spec.Peer) != keySize {
		msg := fmt.Sprintf("invalid peer key length, got %d wanted %d", len(lb.Spec.Peer), keySize)

		s.recorder.Eventf(lb, "Warning", "InvalidKeyLength", msg)
		conditions.Set(lb, conditions.FalseCondition(lockboxv1.ReadyCondition, "InvalidKeyLength", lockboxv1.ConditionSeverityError, msg))
		_ = s.client.Status().Update(ctx, lb)
		return reconcile.Result{}, fmt.Errorf("incorrect peer key length: %d, should be %d", len(lb.Spec.Peer), keySize)
	}

	peerKey := new([keySize]byte)
	copy(peerKey[:], lb.Spec.Peer)

	if !nacl.Verify32(peerKey, s.pubKey) {
		msg := fmt.Sprintf("lockbox has unknown peer key %q", base64.StdEncoding.EncodeToString(lb.Spec.Peer))

		s.recorder.Eventf(lb, "Warning", "UnknownPeerKey", msg)
		conditions.Set(lb, conditions.FalseCondition(lockboxv1.ReadyCondition, "UnknownPeerKey", lockboxv1.ConditionSeverityError, msg))
		_ = s.client.Status().Update(ctx, lb)
		return reconcile.Result{}, fmt.Errorf("unknown peer key")
	}

	sender := new([keySize]byte)
	copy(sender[:], lb.Spec.Sender)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lb.Name,
			Namespace: lb.Namespace,
		},
	}

	namespace, err := box.EasyOpen(lb.Spec.Namespace, sender, s.priKey)
	if err != nil {
		msg := fmt.Sprintf("unable to open lockbox with peer key %q", base64.StdEncoding.EncodeToString(lb.Spec.Peer))

		s.recorder.Eventf(lb, "Warning", "InvalidLockbox", msg)
		conditions.Set(lb, conditions.FalseCondition(lockboxv1.ReadyCondition, "InvalidLockbox", lockboxv1.ConditionSeverityError, msg))
		_ = s.client.Status().Update(ctx, lb)
		return reconcile.Result{}, err
	}

	if string(namespace) != lb.Namespace {
		msg := fmt.Sprintf("locked for namespace %q, found in namespace %s", namespace, lb.Namespace)

		s.recorder.Eventf(lb, "Warning", "InvalidNamespace", msg)
		conditions.Set(lb, conditions.FalseCondition(lockboxv1.ReadyCondition, "InvalidNamespace", lockboxv1.ConditionSeverityWarning, msg))
		_ = s.client.Status().Update(ctx, lb)
		return reconcile.Result{}, fmt.Errorf("incorrect namespace: %s, should be %s", namespace, lb.Namespace)
	}

	_, err = controllerutil.CreateOrPatch(
		ctx,
		s.client,
		secret,
		s.reconcileExisting(lb, sender, secret))

	if err != nil {
		conditions.Set(lb, conditions.FalseCondition(lockboxv1.ReadyCondition, "InvalidLockbox", lockboxv1.ConditionSeverityWarning, err.Error()))
		_ = s.client.Status().Update(ctx, lb)
		return reconcile.Result{}, err
	}

	conditions.Set(lb, conditions.TrueCondition(lockboxv1.ReadyCondition))
	_ = s.client.Status().Update(ctx, lb)
	return reconcile.Result{}, nil
}

// reconcileExisting returns a function suitable for controllerutil.CreateOrUpdate that mutates a Secret object
// to reflect the desired state.
func (s *SecretReconciler) reconcileExisting(lb *lockboxv1.Lockbox, sender nacl.Key, secret *corev1.Secret) func() error {
	return func() error {
		if err := controllerutil.SetControllerReference(lb, secret, s.client.Scheme()); err != nil {
			switch err := err.(type) {
			case decryptSecretKeyErrorer:
				s.recorder.Eventf(lb, "Warning", "InvalidLockbox", "lockbox contained key %q that could not be unlocked", err.SecretKey())
			default:
				s.recorder.Eventf(lb, "Warning", "InvalidLockbox", "lockbox could not be unlocked")
			}

			return err
		}

		return lb.UnlockInto(secret, s.priKey)
	}
}

// WithRecorder sets the EventRecorder used by the SecretReconciler.
func WithRecorder(r record.EventRecorder) SecretReconcilerOption {
	return func(s *SecretReconciler) {
		s.recorder = r
	}
}

// WithClient sets the API Client used by the SecretReconciler
func WithClient(c client.Client) SecretReconcilerOption {
	return func(s *SecretReconciler) {
		s.client = c
	}
}

// decryptSecretKeyErrorer matches the unexported error type, to
// fetch the secret data key that triggered the error.
type decryptSecretKeyErrorer interface {
	SecretKey() string
}
