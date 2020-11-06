// +build suite

package controller_test

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	lockboxv1 "github.com/cloudflare/lockbox/pkg/apis/lockbox.k8s.cloudflare.com/v1"
	. "github.com/cloudflare/lockbox/pkg/lockbox-controller"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var cfg *rest.Config

func TestMain(m *testing.M) {
	t := &envtest.Environment{
		CRDs: []*apiextensionsv1beta1.CustomResourceDefinition{
			&apiextensionsv1beta1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "sealedsecrets.bitnami.com",
				},
				Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
					Group:   "bitnami.com",
					Scope:   apiextensionsv1beta1.NamespaceScoped,
					Version: "v1beta1",
					Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
						Kind:     "SealedSecret",
						ListKind: "SealedSecretList",
						Plural:   "sealedsecrets",
						Singular: "sealedsecret",
					},
				},
			},
		},
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
	pubKey, priKey, err := loadKeypair(t, "6a42b9fc2b011fb88c01741483e3bffe455bdab1ae35d0bb53a3c00d406d8836", "252173f975f0a0ddb198a7e5958c074203a0e9f44275e0b840f95d456c4acc2e")
	if err != nil {
		t.Fatalf("unexpected error loading keypair: %v", err)
	}

	corev1.AddToScheme(scheme.Scheme)
	lockboxv1.Install(scheme.Scheme)

	isController := true
	blockOwnerDeletion := true

	tests := []struct {
		name           string
		setup          func(client.Client) error
		teardown       func(client.Client) error
		expectedSecret corev1.Secret
	}{
		{
			name: "create secret",
			setup: func(c client.Client) error {
				lb := &lockboxv1.Lockbox{
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
					Spec: lockboxv1.LockboxSpec{
						Sender:    []byte{0xb2, 0xa3, 0xf, 0x85, 0xa, 0x58, 0xcf, 0x94, 0x4c, 0x62, 0x37, 0xd4, 0xef, 0xf5, 0xed, 0x11, 0x52, 0xfa, 0x1b, 0xc3, 0xb0, 0x4d, 0x27, 0xd5, 0x58, 0x67, 0x61, 0x67, 0xe0, 0x10, 0xb1, 0x5c},
						Peer:      []byte{0x6a, 0x42, 0xb9, 0xfc, 0x2b, 0x1, 0x1f, 0xb8, 0x8c, 0x1, 0x74, 0x14, 0x83, 0xe3, 0xbf, 0xfe, 0x45, 0x5b, 0xda, 0xb1, 0xae, 0x35, 0xd0, 0xbb, 0x53, 0xa3, 0xc0, 0xd, 0x40, 0x6d, 0x88, 0x36},
						Namespace: []byte{0x4d, 0xa0, 0x73, 0x8b, 0x95, 0xc3, 0xd4, 0x64, 0xe9, 0xab, 0xd, 0xb7, 0x1e, 0x5, 0x10, 0xed, 0x4c, 0x2f, 0x8a, 0x66, 0x6d, 0xec, 0x7c, 0x5d, 0x9b, 0xa7, 0xb7, 0x88, 0x49, 0x8a, 0xb9, 0x7f, 0xf0, 0x30, 0xe0, 0xad, 0x49, 0x7c, 0x3f, 0xe3, 0x1c, 0x2e, 0xe9, 0xb1, 0x2a, 0x70, 0x28},
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
							"test":  {0x7b, 0xca, 0x32, 0x90, 0xf7, 0x97, 0x3b, 0x6, 0xfb, 0x7c, 0xdc, 0x3a, 0x25, 0x82, 0x29, 0xdf, 0x9d, 0x1e, 0x46, 0x8d, 0xd4, 0x99, 0x49, 0x2, 0x63, 0x56, 0x54, 0x64, 0xae, 0x9e, 0xf2, 0xc0, 0x35, 0xf5, 0xf1, 0xcb, 0x67, 0xb7, 0xe2, 0xb1, 0x14, 0x42, 0x71, 0xc},
							"test1": {0x2c, 0x68, 0xed, 0x53, 0x55, 0x55, 0xe2, 0x2d, 0x71, 0x96, 0x85, 0xfd, 0xdb, 0x93, 0x1e, 0x77, 0x91, 0x2d, 0x76, 0xba, 0xae, 0x46, 0x30, 0x9e, 0xb6, 0x65, 0xa2, 0x49, 0xfe, 0x78, 0xc0, 0xcb, 0x6d, 0xf, 0xa8, 0xeb, 0xa8, 0xfc, 0xc0, 0xa0, 0xdc, 0x4, 0x16, 0x7, 0xa0},
						},
					},
				}

				return c.Create(context.TODO(), lb)
			},
			teardown: func(c client.Client) error {
				if err := c.Delete(context.TODO(), &lockboxv1.Lockbox{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example",
						Namespace: "example",
					},
				}); err != nil {
					return err
				}

				return c.Delete(context.TODO(), &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example",
						Namespace: "example",
					},
				})
			},
			expectedSecret: corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example",
					Namespace: "example",
					Labels: map[string]string{
						"type": "secret",
					},
					Annotations: map[string]string{
						"wave": "ignore",
					},
					SelfLink: "/api/v1/namespaces/example/secrets/example",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "lockbox.k8s.cloudflare.com/v1",
							Kind:               "Lockbox",
							Name:               "example",
							Controller:         &isController,
							BlockOwnerDeletion: &blockOwnerDeletion,
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
		{
			name: "avoids updating secrets owned by other controllers",
			setup: func(c client.Client) error {
				ss := &unstructured.Unstructured{}
				ss.SetName("example")
				ss.SetNamespace("example")
				ss.SetAPIVersion("bitnami.com/v1beta1")
				ss.SetKind("SealedSecret")

				if err := c.Create(context.TODO(), ss); err != nil {
					return err
				}
				<-time.After(1 * time.Second)

				if err := c.Get(context.TODO(), client.ObjectKey{
					Name:      "example",
					Namespace: "example",
				}, ss); err != nil {
					return err
				}

				lb := &lockboxv1.Lockbox{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example",
						Namespace: "example",
					},
					Spec: lockboxv1.LockboxSpec{
						Sender:    []byte{0xa, 0xda, 0x33, 0xf3, 0x48, 0xad, 0xb6, 0x4c, 0xaa, 0x6, 0x50, 0xc1, 0xe1, 0xa6, 0xeb, 0x49, 0x13, 0xe0, 0x53, 0xdf, 0xde, 0x44, 0x72, 0xd6, 0xe2, 0x51, 0x94, 0xee, 0xcb, 0xba, 0xc1, 0x4},
						Peer:      []byte{0x6a, 0x42, 0xb9, 0xfc, 0x2b, 0x1, 0x1f, 0xb8, 0x8c, 0x1, 0x74, 0x14, 0x83, 0xe3, 0xbf, 0xfe, 0x45, 0x5b, 0xda, 0xb1, 0xae, 0x35, 0xd0, 0xbb, 0x53, 0xa3, 0xc0, 0xd, 0x40, 0x6d, 0x88, 0x36},
						Namespace: []byte{0xa7, 0x4c, 0x72, 0x7a, 0x71, 0x1d, 0x98, 0x32, 0xa, 0x3, 0xbe, 0xe5, 0x9d, 0xd4, 0x8c, 0x39, 0x3, 0x42, 0x9c, 0x5e, 0xeb, 0x6d, 0x95, 0x46, 0x5c, 0x10, 0x62, 0xa3, 0xa7, 0xfb, 0xee, 0x19, 0xcb, 0x98, 0xbf, 0xc1, 0x19, 0x66, 0x6a, 0x77, 0x76, 0x22, 0x17, 0x8f, 0xa5, 0x24, 0x8e},
						Data: map[string][]byte{
							"updated": {0x78, 0x70, 0x68, 0xae, 0x9f, 0xf5, 0xed, 0x60, 0x74, 0x14, 0x6a, 0xc5, 0xc3, 0xb, 0xe2, 0xaa, 0x20, 0x68, 0x7a, 0xfb, 0xa6, 0x6a, 0x38, 0xc2, 0x20, 0x73, 0xb5, 0x45, 0x9f, 0x9, 0xf0, 0x15, 0xd1, 0x5c, 0x16, 0x51, 0x50, 0xaa, 0xea, 0x68, 0x3a, 0x95, 0xe6},
						},
					},
				}
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example",
						Namespace: "example",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "bitnami.com/v1alpha1",
								Kind:               "SealedSecret",
								Name:               "example",
								UID:                ss.GetUID(),
								Controller:         &isController,
								BlockOwnerDeletion: &blockOwnerDeletion,
							},
						},
					},
					Data: map[string][]byte{
						"test":  []byte("test"),
						"test1": []byte("test1"),
					},
				}

				if err := c.Create(context.TODO(), secret); err != nil {
					return err
				}
				<-time.After(1 * time.Second)

				return c.Create(context.TODO(), lb)
			},
			teardown: func(c client.Client) error {
				if err := c.Delete(context.TODO(), &lockboxv1.Lockbox{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example",
						Namespace: "example",
					},
				}); err != nil {
					return err
				}

				if err := c.Delete(context.TODO(), &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example",
						Namespace: "example",
					},
				}); err != nil {
					return err
				}

				ss := &unstructured.Unstructured{}
				ss.SetName("example")
				ss.SetNamespace("example")
				ss.SetAPIVersion("bitnami.com/v1beta1")
				ss.SetKind("SealedSecret")
				return c.Delete(context.TODO(), ss)
			},
			expectedSecret: corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example",
					Namespace: "example",
					SelfLink:  "/api/v1/namespaces/example/secrets/example",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "bitnami.com/v1alpha1",
							Kind:               "SealedSecret",
							Name:               "example",
							Controller:         &isController,
							BlockOwnerDeletion: &blockOwnerDeletion,
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mgr, err := manager.New(cfg, manager.Options{
				Scheme: scheme.Scheme,
			})
			if err != nil {
				t.Fatalf("Unexpected error while creating manager: %v", err)
			}

			c := mgr.GetClient()

			sr := NewSecretReconciler(pubKey, priKey, WithClient(c))
			err = builder.
				ControllerManagedBy(mgr).
				For(&lockboxv1.Lockbox{}).
				Owns(&corev1.Secret{}).
				Complete(sr)
			if err != nil {
				t.Fatalf("Unexpected error while creating controller: %v", err)
			}

			go func() {
				mgr.Start(ctx.Done())
			}()

			if err := tt.setup(c); err != nil {
				t.Fatalf("Failed setting up API: %v\n", err)
			}

			defer func() {
				if err := tt.teardown(c); err != nil {
					t.Errorf("teardown error: %v", err)
				}
				<-time.After(1 * time.Second)
			}()

			<-time.After(1 * time.Second)
			secret := corev1.Secret{}
			c.Get(context.TODO(), client.ObjectKey{
				Name:      "example",
				Namespace: "example",
			}, &secret)

			opts := []cmp.Option{
				cmpopts.IgnoreFields(metav1.ObjectMeta{}, "UID", "ResourceVersion", "CreationTimestamp", "ManagedFields"),
				cmpopts.IgnoreFields(metav1.OwnerReference{}, "UID"),
			}

			if diff := cmp.Diff(secret, tt.expectedSecret, opts...); diff != "" {
				t.Errorf("unexpected secret state: (+want -got)\n%s", diff)
			}
		})
	}
}
