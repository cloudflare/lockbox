package controller_test

import (
	"context"
	"testing"

	lockboxv1 "github.com/cloudflare/lockbox/pkg/apis/lockbox.k8s.cloudflare.com/v1"
	controller "github.com/cloudflare/lockbox/pkg/lockbox-controller"
	"github.com/kevinburke/nacl"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestSecretReconciler(t *testing.T) {
	type testCase struct {
		name        string
		lockboxName string
		resources   []client.Object
		expected    *corev1.Secret
		expectedErr string
	}

	run := func(t *testing.T, tc testCase) {
		scheme := runtime.NewScheme()
		assert.NilError(t, corev1.AddToScheme(scheme))
		assert.NilError(t, lockboxv1.Install(scheme))

		client := clientfake.NewClientBuilder().
			WithObjects(tc.resources...).
			WithScheme(scheme).
			Build()

		pubKey, priKey, err := loadKeypair(t, "6a42b9fc2b011fb88c01741483e3bffe455bdab1ae35d0bb53a3c00d406d8836", "252173f975f0a0ddb198a7e5958c074203a0e9f44275e0b840f95d456c4acc2e")
		assert.NilError(t, err)

		lsn := types.NamespacedName{Name: tc.lockboxName, Namespace: "example"}
		sr := controller.NewSecretReconciler(pubKey, priKey, controller.WithClient(client))

		_, err = sr.Reconcile(context.Background(), reconcile.Request{NamespacedName: lsn})
		if tc.expectedErr != "" {
			assert.ErrorContains(t, err, tc.expectedErr)
		} else {
			assert.NilError(t, err)
		}

		actual := &corev1.Secret{}
		err = client.Get(context.Background(), lsn, actual)

		if tc.expected == nil {
			assert.Assert(t, apierrors.IsNotFound(err))
			return
		}

		assert.NilError(t, err)
		assert.DeepEqual(t, actual, tc.expected)
	}

	testCases := []testCase{
		{
			name:        "new lockbox",
			lockboxName: "example",
			resources: []client.Object{
				&lockboxv1.Lockbox{
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
							Type: corev1.SecretTypeOpaque,
						},
						Data: map[string][]byte{
							"test":  {0x7b, 0xca, 0x32, 0x90, 0xf7, 0x97, 0x3b, 0x6, 0xfb, 0x7c, 0xdc, 0x3a, 0x25, 0x82, 0x29, 0xdf, 0x9d, 0x1e, 0x46, 0x8d, 0xd4, 0x99, 0x49, 0x2, 0x63, 0x56, 0x54, 0x64, 0xae, 0x9e, 0xf2, 0xc0, 0x35, 0xf5, 0xf1, 0xcb, 0x67, 0xb7, 0xe2, 0xb1, 0x14, 0x42, 0x71, 0xc},
							"test1": {0x2c, 0x68, 0xed, 0x53, 0x55, 0x55, 0xe2, 0x2d, 0x71, 0x96, 0x85, 0xfd, 0xdb, 0x93, 0x1e, 0x77, 0x91, 0x2d, 0x76, 0xba, 0xae, 0x46, 0x30, 0x9e, 0xb6, 0x65, 0xa2, 0x49, 0xfe, 0x78, 0xc0, 0xcb, 0x6d, 0xf, 0xa8, 0xeb, 0xa8, 0xfc, 0xc0, 0xa0, 0xdc, 0x4, 0x16, 0x7, 0xa0},
						},
					},
				},
			},
			expected: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
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
					ResourceVersion: "1",
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
					"test":  []byte("test"),
					"test1": []byte("test1"),
				},
			},
		},
		{
			name:        "update lockbox secret",
			lockboxName: "example",
			resources: []client.Object{
				&lockboxv1.Lockbox{
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
						Template: lockboxv1.LockboxSecretTemplate{
							Type: corev1.SecretTypeOpaque,
						},
					},
				},
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
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
						ResourceVersion: "1",
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
						"test":  []byte("test"),
						"test1": []byte("test1"),
					},
				},
			},
			expected: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "example",
					Namespace:       "example",
					ResourceVersion: "2",
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
					"updated": []byte("yep"),
				},
			},
		},
		{
			name:        "secret conflict",
			lockboxName: "example",
			resources: []client.Object{
				&lockboxv1.Lockbox{
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
				},
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            "example",
						Namespace:       "example",
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "bitnami.com/v1alpha1",
								Kind:               "SealedSecret",
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
			expected: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "example",
					Namespace:       "example",
					ResourceVersion: "1",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "bitnami.com/v1alpha1",
							Kind:               "SealedSecret",
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
			expectedErr: "already owned by another SealedSecret controller example",
		},
		{
			name:        "docker-registry secret",
			lockboxName: "example",
			resources: []client.Object{
				&lockboxv1.Lockbox{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example",
						Namespace: "example",
					},
					Spec: lockboxv1.LockboxSpec{
						Sender:    []byte{0x64, 0x22, 0x82, 0xe9, 0x35, 0x2a, 0x36, 0x7a, 0x40, 0x75, 0xd5, 0x14, 0xa6, 0x24, 0xef, 0xe3, 0x59, 0xda, 0xf5, 0xe9, 0xbe, 0xc8, 0x2d, 0x88, 0xb1, 0x17, 0xe8, 0xf2, 0x99, 0xb0, 0x9f, 0x71},
						Peer:      []byte{0x6a, 0x42, 0xb9, 0xfc, 0x2b, 0x1, 0x1f, 0xb8, 0x8c, 0x1, 0x74, 0x14, 0x83, 0xe3, 0xbf, 0xfe, 0x45, 0x5b, 0xda, 0xb1, 0xae, 0x35, 0xd0, 0xbb, 0x53, 0xa3, 0xc0, 0xd, 0x40, 0x6d, 0x88, 0x36},
						Namespace: []byte{0xd3, 0x1c, 0xc6, 0x29, 0x65, 0xac, 0xd6, 0x5, 0x3a, 0x60, 0xe1, 0x7c, 0xf8, 0xb9, 0x7, 0xdd, 0xdb, 0xf0, 0x82, 0xab, 0x90, 0x38, 0x7, 0x56, 0x72, 0x68, 0xef, 0x56, 0x3b, 0xae, 0x13, 0x16, 0x7e, 0x3e, 0xf6, 0xaf, 0xb4, 0x7b, 0x10, 0xed, 0x77, 0x29, 0xae, 0xcb, 0x96, 0x7f, 0xc9},
						Data: map[string][]byte{
							".dockerconfigjson": {0x98, 0x39, 0x7b, 0x93, 0x7a, 0xb6, 0x4, 0xc, 0xb8, 0x52, 0xf0, 0x97, 0x2e, 0x74, 0xed, 0xd6, 0x41, 0x7a, 0x7d, 0x20, 0xda, 0x35, 0x2d, 0xdf, 0x2b, 0x94, 0x9f, 0x78, 0x78, 0xd4, 0x29, 0x30, 0x6d, 0xbf, 0x9c, 0x59, 0x9f, 0xb4, 0x47, 0x5e, 0x10, 0x4a, 0xd2, 0xf, 0xd8, 0x77, 0x7d, 0x8, 0x11, 0x36, 0x41, 0xa9, 0xb2, 0x77, 0xac, 0xd9, 0xa3, 0x8, 0x81, 0x0, 0x6, 0x34, 0xde, 0x3e, 0xfc, 0x38, 0x4c, 0xa4, 0x27, 0xff, 0x1f, 0x67, 0x8, 0xef, 0x6, 0xff, 0x31, 0x80, 0xd, 0x4e, 0xcf, 0x6c, 0xec, 0x79, 0x78, 0x7d, 0x9f, 0x5b, 0x34, 0xe4, 0x5a, 0x44, 0x49, 0x57, 0xfd, 0xeb, 0x43, 0xd4, 0x4e, 0xe5, 0x15, 0xbc, 0xa8, 0x5a, 0x86, 0xd, 0xb9, 0xaa, 0x45, 0x6c, 0x4b, 0x17, 0x66, 0x13, 0xb2, 0x8c, 0x46, 0x7e, 0xdc, 0xe, 0x21, 0x54, 0x39, 0x27, 0xa3, 0x93, 0x52, 0x46, 0xa1, 0x71, 0x21, 0x8e, 0x27, 0x62, 0x6b, 0x86, 0xa6, 0xe4, 0x98, 0xc4, 0xff, 0x8, 0xed, 0xba, 0x4d, 0xa1, 0xfa, 0x53, 0x25, 0xb7, 0x29, 0x20, 0x1a, 0xab, 0x4a, 0xf5, 0x99, 0x99, 0x6a, 0x9d, 0xb8, 0x96, 0x28, 0x9b, 0x6a, 0xda, 0xb8, 0xee, 0x9c, 0x5f, 0xc1, 0x91, 0x0, 0x38, 0x84, 0x90, 0xdf, 0xbd, 0x9a, 0x1b, 0x9e, 0xd6, 0xe4, 0x3d},
						},
						Template: lockboxv1.LockboxSecretTemplate{
							Type: corev1.SecretTypeDockerConfigJson,
						},
					},
				},
			},
			expected: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "example",
					Namespace:       "example",
					ResourceVersion: "1",
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
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					".dockerconfigjson": {0x65, 0x79, 0x4a, 0x68, 0x64, 0x58, 0x52, 0x6f, 0x63, 0x79, 0x49, 0x36, 0x65, 0x79, 0x4a, 0x6b, 0x62, 0x32, 0x4e, 0x72, 0x5a, 0x58, 0x49, 0x75, 0x5a, 0x58, 0x68, 0x68, 0x62, 0x58, 0x42, 0x73, 0x5a, 0x53, 0x35, 0x6a, 0x62, 0x32, 0x30, 0x69, 0x4f, 0x6e, 0x73, 0x69, 0x56, 0x58, 0x4e, 0x6c, 0x63, 0x6d, 0x35, 0x68, 0x62, 0x57, 0x55, 0x69, 0x4f, 0x69, 0x4a, 0x71, 0x62, 0x32, 0x56, 0x6b, 0x5a, 0x58, 0x5a, 0x6c, 0x62, 0x47, 0x39, 0x77, 0x5a, 0x58, 0x49, 0x69, 0x4c, 0x43, 0x4a, 0x51, 0x59, 0x58, 0x4e, 0x7a, 0x64, 0x32, 0x39, 0x79, 0x5a, 0x43, 0x49, 0x36, 0x49, 0x6e, 0x42, 0x68, 0x63, 0x33, 0x4e, 0x33, 0x62, 0x33, 0x4a, 0x6b, 0x49, 0x69, 0x77, 0x69, 0x52, 0x57, 0x31, 0x68, 0x61, 0x57, 0x77, 0x69, 0x4f, 0x69, 0x4a, 0x71, 0x62, 0x32, 0x56, 0x41, 0x5a, 0x58, 0x68, 0x68, 0x62, 0x58, 0x42, 0x73, 0x5a, 0x53, 0x35, 0x6a, 0x62, 0x32, 0x30, 0x69, 0x66, 0x58, 0x31, 0x39},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
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
