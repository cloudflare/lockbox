//go:build suite
// +build suite

package controller_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	lockboxv1 "github.com/cloudflare/lockbox/pkg/apis/lockbox.k8s.cloudflare.com/v1"
	. "github.com/cloudflare/lockbox/pkg/lockbox-controller"
	"github.com/go-logr/zerologr"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/rs/zerolog"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/poll"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var cfg *rest.Config

func TestMain(m *testing.M) {
	zl := zerolog.New(os.Stderr)
	logf.SetLogger(zerologr.New(&zl))
	t := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "deployment", "crds")},
	}
	lockboxv1.Install(scheme.Scheme)

	var err error
	if cfg, err = t.Start(); err != nil {
		log.Fatal(err)
	}

	code := m.Run()
	t.Stop()
	os.Exit(code)
}

func TestSuiteSecretReconciler(t *testing.T) {
	type testCase struct {
		name        string
		lockboxName string
		resources   []client.Object
		expected    *corev1.Secret
	}

	pubKey, priKey, err := loadKeypair(t, "6a42b9fc2b011fb88c01741483e3bffe455bdab1ae35d0bb53a3c00d406d8836", "252173f975f0a0ddb198a7e5958c074203a0e9f44275e0b840f95d456c4acc2e")
	assert.NilError(t, err)

	setup := func(t *testing.T, tc testCase) {
		mgr, err := manager.New(cfg, manager.Options{
			MetricsBindAddress: "0",
			Scheme:             scheme.Scheme,
		})
		assert.NilError(t, err)

		sr := NewSecretReconciler(pubKey, priKey, WithClient(mgr.GetClient()))
		err = builder.
			ControllerManagedBy(mgr).
			For(&lockboxv1.Lockbox{}).
			Owns(&corev1.Secret{}).
			Complete(sr)
		assert.NilError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			mgr.Start(ctx)
		}()
		t.Cleanup(cancel)
	}

	run := func(t *testing.T, tc testCase) {
		c, err := client.New(cfg, client.Options{})
		assert.NilError(t, err)

		for _, r := range tc.resources {
			c.Create(context.Background(), r)
		}

		secret := &corev1.Secret{}
		poll.WaitOn(t, func(t poll.LogT) poll.Result {
			err := c.Get(context.Background(), client.ObjectKey{
				Name:      "example",
				Namespace: "default",
			}, secret)

			if err == nil {
				return poll.Success()
			}

			if apierrors.IsNotFound(err) {
				return poll.Continue("secret was not found")
			}

			return poll.Error(err)
		})

		cm := &corev1.ConfigMap{}
		c.Get(context.Background(), client.ObjectKey{
			Name:      "example",
			Namespace: "default",
		}, cm)

		fmt.Printf("cm: %+v\n", *cm)

		assert.DeepEqual(t, secret, tc.expected,
			cmpopts.IgnoreFields(metav1.ObjectMeta{}, "UID", "ResourceVersion", "CreationTimestamp", "ManagedFields"),
			cmpopts.IgnoreFields(metav1.OwnerReference{}, "UID"),
		)

		for _, r := range tc.resources {
			c.Delete(context.Background(), r)
		}
		// delete the created resource too, as there's no garbage collector
		c.Delete(context.Background(), tc.expected)
	}

	testCases := []testCase{
		{
			name:        "create secret",
			lockboxName: "example",
			resources: []client.Object{
				&lockboxv1.Lockbox{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example",
						Namespace: "default",
						Labels: map[string]string{
							"type": "lockbox",
						},
						Annotations: map[string]string{
							"helm.sh/hook": "pre-install",
						},
					},
					Spec: lockboxv1.LockboxSpec{
						Sender:    []byte{0x74, 0xbd, 0xd8, 0x82, 0xf7, 0xd5, 0x87, 0xde, 0x08, 0x79, 0xf0, 0x9b, 0x35, 0x15, 0xf5, 0x2d, 0x1f, 0xb0, 0x26, 0xb3, 0x20, 0xe1, 0xe1, 0xd8, 0x5c, 0x5a, 0x0e, 0x1d, 0xfb, 0x80, 0x87, 0x23},
						Peer:      []byte{0x6a, 0x42, 0xb9, 0xfc, 0x2b, 0x01, 0x1f, 0xb8, 0x8c, 0x01, 0x74, 0x14, 0x83, 0xe3, 0xbf, 0xfe, 0x45, 0x5b, 0xda, 0xb1, 0xae, 0x35, 0xd0, 0xbb, 0x53, 0xa3, 0xc0, 0x0d, 0x40, 0x6d, 0x88, 0x36},
						Namespace: []byte{0x3a, 0x1a, 0x82, 0xd1, 0xad, 0x9f, 0x89, 0x6b, 0x59, 0x8e, 0xce, 0x45, 0xbc, 0x6f, 0x61, 0x34, 0x81, 0x7b, 0x7e, 0x2f, 0xa4, 0xd7, 0x15, 0xaf, 0x28, 0x15, 0xc0, 0x3e, 0x21, 0xfc, 0xcb, 0x3a, 0x38, 0x60, 0x96, 0xc7, 0xac, 0xe6, 0x56, 0xf2, 0xb7, 0x40, 0x4e, 0x9e, 0xb4, 0xbf, 0x96},
						Template: lockboxv1.LockboxSecretTemplate{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"type": "secret",
								},
								Annotations: map[string]string{
									"wave": "ignore",
								},
							},
						},
						Data: map[string][]byte{
							"test": {0x57, 0x17, 0x83, 0x22, 0x4c, 0x54, 0x1a, 0xb8, 0x83, 0x86, 0xc6, 0x15, 0xed, 0x23, 0x10, 0x58, 0x1d, 0xbc, 0x20, 0x47, 0xb4, 0x2a, 0x7f, 0xf6, 0xda, 0x4e, 0xa4, 0x88, 0x6b, 0x54, 0xed, 0xf6, 0xa3, 0x21, 0x73, 0xda, 0xca, 0x2b, 0xf7, 0x88, 0x13, 0xaa, 0xc2, 0xef},
						},
					},
				},
			},
			expected: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example",
					Namespace: "default",
					Labels: map[string]string{
						"type": "secret",
					},
					Annotations: map[string]string{
						"wave": "ignore",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "lockbox.k8s.cloudflare.com/v1",
							Kind:               "Lockbox",
							Name:               "example",
							Controller:         pointer.Bool(true),
							BlockOwnerDeletion: pointer.Bool(true),
						},
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"test": []byte("test"),
				},
			},
		},
		{
			name:        "avoids updating secrets owned by other controllers",
			lockboxName: "example",
			resources: []client.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "v1",
								Kind:               "ConfigMap",
								Name:               "example",
								UID:                "deadbeef",
								Controller:         pointer.Bool(true),
								BlockOwnerDeletion: pointer.Bool(true),
							},
						},
					},
					Data: map[string][]byte{
						"test":  []byte("test"),
						"test1": []byte("test1"),
					},
				},
				&lockboxv1.Lockbox{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example",
						Namespace: "default",
					},
					Spec: lockboxv1.LockboxSpec{
						Sender:    []byte{0x74, 0xbd, 0xd8, 0x82, 0xf7, 0xd5, 0x87, 0xde, 0x08, 0x79, 0xf0, 0x9b, 0x35, 0x15, 0xf5, 0x2d, 0x1f, 0xb0, 0x26, 0xb3, 0x20, 0xe1, 0xe1, 0xd8, 0x5c, 0x5a, 0x0e, 0x1d, 0xfb, 0x80, 0x87, 0x23},
						Peer:      []byte{0x6a, 0x42, 0xb9, 0xfc, 0x2b, 0x01, 0x1f, 0xb8, 0x8c, 0x01, 0x74, 0x14, 0x83, 0xe3, 0xbf, 0xfe, 0x45, 0x5b, 0xda, 0xb1, 0xae, 0x35, 0xd0, 0xbb, 0x53, 0xa3, 0xc0, 0x0d, 0x40, 0x6d, 0x88, 0x36},
						Namespace: []byte{0x3a, 0x1a, 0x82, 0xd1, 0xad, 0x9f, 0x89, 0x6b, 0x59, 0x8e, 0xce, 0x45, 0xbc, 0x6f, 0x61, 0x34, 0x81, 0x7b, 0x7e, 0x2f, 0xa4, 0xd7, 0x15, 0xaf, 0x28, 0x15, 0xc0, 0x3e, 0x21, 0xfc, 0xcb, 0x3a, 0x38, 0x60, 0x96, 0xc7, 0xac, 0xe6, 0x56, 0xf2, 0xb7, 0x40, 0x4e, 0x9e, 0xb4, 0xbf, 0x96},
						Data: map[string][]byte{
							"test": {0x57, 0x17, 0x83, 0x22, 0x4c, 0x54, 0x1a, 0xb8, 0x83, 0x86, 0xc6, 0x15, 0xed, 0x23, 0x10, 0x58, 0x1d, 0xbc, 0x20, 0x47, 0xb4, 0x2a, 0x7f, 0xf6, 0xda, 0x4e, 0xa4, 0x88, 0x6b, 0x54, 0xed, 0xf6, 0xa3, 0x21, 0x73, 0xda, 0xca, 0x2b, 0xf7, 0x88, 0x13, 0xaa, 0xc2, 0xef},
						},
					},
				},
			},
			expected: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example",
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "v1",
							Kind:               "ConfigMap",
							Name:               "example",
							Controller:         pointer.Bool(true),
							BlockOwnerDeletion: pointer.Bool(true),
						},
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"test":  []byte("test"),
					"test1": []byte("test1"),
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setup(t, tc)
			run(t, tc)
		})
	}

}
