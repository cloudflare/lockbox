package v1

import (
	"github.com/kevinburke/nacl"
	"github.com/kevinburke/nacl/box"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const keySize = nacl.KeySize

// NewFromSecret creates a Lockbox wrapping the provided Secret. The value of each secret
// are individually encrypted using the provided key pair.
func NewFromSecret(secret corev1.Secret, namespace string, peer, pub, pri nacl.Key) *Lockbox {
	encNS := box.EasySeal([]byte(namespace), peer, pri)

	b := &Lockbox{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: namespace,
		},
		Spec: LockboxSpec{
			Sender:    pub[:],
			Peer:      peer[:],
			Namespace: encNS,
			Data:      map[string][]byte{},
			Template: LockboxSecretTemplate{
				LockboxSecretTemplateMetadata: LockboxSecretTemplateMetadata{
					Labels:      secret.ObjectMeta.Labels,
					Annotations: secret.ObjectMeta.Annotations,
				},
				Type: secret.Type,
			},
		},
	}

	for key, value := range secret.Data {
		enc := box.EasySeal(value, peer, pri)
		b.Spec.Data[key] = enc
	}

	for key, value := range secret.StringData {
		enc := box.EasySeal([]byte(value), peer, pri)
		b.Spec.Data[key] = enc
	}

	return b
}

// UnlockInto decrypts each secret value into the provided secret.
func (in *Lockbox) UnlockInto(secret *corev1.Secret, pri nacl.Key) error {
	sender := new([keySize]byte)
	copy(sender[:], in.Spec.Sender)

	data := make(map[string][]byte, len(in.Spec.Data))
	for key, val := range in.Spec.Data {
		d, err := box.EasyOpen(val, sender, pri)
		if err != nil {
			return decryptSecretKeyError{error: err, key: key}
		}

		data[key] = d
	}

	secret.Data = data
	secret.Type = in.Spec.Template.Type
	secret.Labels = in.Spec.Template.Labels
	secret.Annotations = in.Spec.Template.Annotations

	return nil
}

// decryptSecretKeyError wraps error while decrypting data from a secret.
// This allows preserving the key for farther error messages.
type decryptSecretKeyError struct {
	error
	key string
}

// SecretKey returns the secret data key that triggered this error.
func (e decryptSecretKeyError) SecretKey() string {
	return e.key
}

// Unwrap implements Wrapper, returning the underlying error message.
func (e decryptSecretKeyError) Unwrap() error {
	return e.error
}

func (in *Lockbox) GetConditions() []Condition {
	return in.Status.Conditions
}

func (in *Lockbox) SetConditions(conditions []Condition) {
	in.Status.Conditions = conditions
}
