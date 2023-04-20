package v1_test

import (
	"crypto/rand"
	"testing"

	v1 "github.com/cloudflare/lockbox/pkg/apis/lockbox.k8s.cloudflare.com/v1"
	"github.com/kevinburke/nacl"
	"github.com/kevinburke/nacl/box"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUnlock(t *testing.T) {
	_, priKey, err := loadKeypair(t, "6a42b9fc2b011fb88c01741483e3bffe455bdab1ae35d0bb53a3c00d406d8836", "252173f975f0a0ddb198a7e5958c074203a0e9f44275e0b840f95d456c4acc2e")
	assert.NilError(t, err)

	lb := &v1.Lockbox{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "example",
			Labels: map[string]string{
				"type": "lockbox",
			},
			Annotations: map[string]string{
				"helm.sh/hook": "pre-install",
			},
		},
		Spec: v1.LockboxSpec{
			Sender:    []byte{0xb2, 0xa3, 0xf, 0x85, 0xa, 0x58, 0xcf, 0x94, 0x4c, 0x62, 0x37, 0xd4, 0xef, 0xf5, 0xed, 0x11, 0x52, 0xfa, 0x1b, 0xc3, 0xb0, 0x4d, 0x27, 0xd5, 0x58, 0x67, 0x61, 0x67, 0xe0, 0x10, 0xb1, 0x5c},
			Peer:      []byte{0x6a, 0x42, 0xb9, 0xfc, 0x2b, 0x1, 0x1f, 0xb8, 0x8c, 0x1, 0x74, 0x14, 0x83, 0xe3, 0xbf, 0xfe, 0x45, 0x5b, 0xda, 0xb1, 0xae, 0x35, 0xd0, 0xbb, 0x53, 0xa3, 0xc0, 0xd, 0x40, 0x6d, 0x88, 0x36},
			Namespace: []byte{0x4d, 0xa0, 0x73, 0x8b, 0x95, 0xc3, 0xd4, 0x64, 0xe9, 0xab, 0xd, 0xb7, 0x1e, 0x5, 0x10, 0xed, 0x4c, 0x2f, 0x8a, 0x66, 0x6d, 0xec, 0x7c, 0x5d, 0x9b, 0xa7, 0xb7, 0x88, 0x49, 0x8a, 0xb9, 0x7f, 0xf0, 0x30, 0xe0, 0xad, 0x49, 0x7c, 0x3f, 0xe3, 0x1c, 0x2e, 0xe9, 0xb1, 0x2a, 0x70, 0x28},
			Template: v1.LockboxSecretTemplate{
				LockboxSecretTemplateMetadata: v1.LockboxSecretTemplateMetadata{
					Labels: map[string]string{
						"type": "secret",
					},
					Annotations: map[string]string{
						"wave": "ignore",
					},
				},
			},
			Data: map[string][]byte{
				"test":  {0x7b, 0xca, 0x32, 0x90, 0xf7, 0x97, 0x3b, 0x6, 0xfb, 0x7c, 0xdc, 0x3a, 0x25, 0x82, 0x29, 0xdf, 0x9d, 0x1e, 0x46, 0x8d, 0xd4, 0x99, 0x49, 0x2, 0x63, 0x56, 0x54, 0x64, 0xae, 0x9e, 0xf2, 0xc0, 0x35, 0xf5, 0xf1, 0xcb, 0x67, 0xb7, 0xe2, 0xb1, 0x14, 0x42, 0x71, 0xc},
				"test1": {0x2c, 0x68, 0xed, 0x53, 0x55, 0x55, 0xe2, 0x2d, 0x71, 0x96, 0x85, 0xfd, 0xdb, 0x93, 0x1e, 0x77, 0x91, 0x2d, 0x76, 0xba, 0xae, 0x46, 0x30, 0x9e, 0xb6, 0x65, 0xa2, 0x49, 0xfe, 0x78, 0xc0, 0xcb, 0x6d, 0xf, 0xa8, 0xeb, 0xa8, 0xfc, 0xc0, 0xa0, 0xdc, 0x4, 0x16, 0x7, 0xa0},
			},
		},
	}

	secret := &corev1.Secret{}
	expected := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"type": "secret",
			},
			Annotations: map[string]string{
				"wave": "ignore",
			},
		},
		Data: map[string][]byte{
			"test":  {0x74, 0x65, 0x73, 0x74},
			"test1": {0x74, 0x65, 0x73, 0x74, 0x31},
		},
	}

	assert.NilError(t, lb.UnlockInto(secret, priKey))
	assert.DeepEqual(t, secret, expected)
}

func TestUnlockErr(t *testing.T) {
	_, priKey, err := loadKeypair(t, "6a42b9fc2b011fb88c01741483e3bffe455bdab1ae35d0bb53a3c00d406d8836", "252173f975f0a0ddb198a7e5958c074203a0e9f44275e0b840f95d456c4acc2e")
	assert.NilError(t, err)

	lb := &v1.Lockbox{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "example",
			Labels: map[string]string{
				"type": "lockbox",
			},
			Annotations: map[string]string{
				"helm.sh/hook": "pre-install",
			},
		},
		Spec: v1.LockboxSpec{
			Sender:    []byte{0x46, 0xa5, 0xfa, 0xa9, 0xe1, 0xe6, 0xd8, 0x48, 0x14, 0xfd, 0x52, 0x67, 0x98, 0x39, 0x12, 0xda, 0x78, 0x61, 0x95, 0x84, 0x8d, 0xbb, 0xd2, 0x1f, 0x36, 0xaa, 0xdc, 0x6a, 0x18, 0x5c, 0xe5, 0x77},
			Peer:      []byte{0x6a, 0x42, 0xb9, 0xfc, 0x2b, 0x01, 0x1f, 0xb8, 0x8c, 0x01, 0x74, 0x14, 0x83, 0xe3, 0xbf, 0xfe, 0x45, 0x5b, 0xda, 0xb1, 0xae, 0x35, 0xd0, 0xbb, 0x53, 0xa3, 0xc0, 0x0d, 0x40, 0x6d, 0x88, 0x36},
			Namespace: []byte{0x4d, 0xa0, 0x73, 0x8b, 0x95, 0xc3, 0xd4, 0x64, 0xe9, 0xab, 0xd, 0xb7, 0x1e, 0x5, 0x10, 0xed, 0x4c, 0x2f, 0x8a, 0x66, 0x6d, 0xec, 0x7c, 0x5d, 0x9b, 0xa7, 0xb7, 0x88, 0x49, 0x8a, 0xb9, 0x7f, 0xf0, 0x30, 0xe0, 0xad, 0x49, 0x7c, 0x3f, 0xe3, 0x1c, 0x2e, 0xe9, 0xb1, 0x2a, 0x70, 0x28},
			Template: v1.LockboxSecretTemplate{
				LockboxSecretTemplateMetadata: v1.LockboxSecretTemplateMetadata{
					Labels: map[string]string{
						"type": "secret",
					},
					Annotations: map[string]string{
						"wave": "ignore",
					},
				},
			},
			Data: map[string][]byte{
				"test":  {0x7b, 0xca, 0x32, 0x90, 0xf7, 0x97, 0x3b, 0x6, 0xfb, 0x7c, 0xdc, 0x3a, 0x25, 0x82, 0x29, 0xdf, 0x9d, 0x1e, 0x46, 0x8d, 0xd4, 0x99, 0x49, 0x2, 0x63, 0x56, 0x54, 0x64, 0xae, 0x9e, 0xf2, 0xc0, 0x35, 0xf5, 0xf1, 0xcb, 0x67, 0xb7, 0xe2, 0xb1, 0x14, 0x42, 0x71, 0xc},
				"test1": {0x2c, 0x68, 0xed, 0x53, 0x55, 0x55, 0xe2, 0x2d, 0x71, 0x96, 0x85, 0xfd, 0xdb, 0x93, 0x1e, 0x77, 0x91, 0x2d, 0x76, 0xba, 0xae, 0x46, 0x30, 0x9e, 0xb6, 0x65, 0xa2, 0x49, 0xfe, 0x78, 0xc0, 0xcb, 0x6d, 0xf, 0xa8, 0xeb, 0xa8, 0xfc, 0xc0, 0xa0, 0xdc, 0x4, 0x16, 0x7, 0xa0},
			},
		},
	}

	secret := &corev1.Secret{}
	expected := &corev1.Secret{}

	err = lb.UnlockInto(secret, priKey)
	assert.ErrorContains(t, err, "Could not decrypt invalid input")
	assert.DeepEqual(t, secret, expected)
}

func TestLockUnlock(t *testing.T) {
	senderPubKey, senderPriKey, _ := box.GenerateKey(rand.Reader)
	serverPubKey, serverPriKey, _ := box.GenerateKey(rand.Reader)

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"type": "secret",
			},
		},
		Data: map[string][]byte{
			"test": {0x74, 0x65, 0x73, 0x74},
		},
	}

	lb := v1.NewFromSecret(secret, "namespace", serverPubKey, senderPubKey, senderPriKey)
	unlockedSecret := &corev1.Secret{}
	expectedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"type": "secret",
			},
		},
		Data: map[string][]byte{
			"test": {0x74, 0x65, 0x73, 0x74},
		},
	}

	assert.NilError(t, lb.UnlockInto(unlockedSecret, serverPriKey))
	assert.DeepEqual(t, unlockedSecret, expectedSecret)
}

func loadKeypair(t *testing.T, pub, pri string) (pubKey, priKey nacl.Key, err error) {
	t.Helper()

	pubKey, err = nacl.Load(pub)
	if err != nil {
		return
	}

	priKey, err = nacl.Load(pri)
	return
}
