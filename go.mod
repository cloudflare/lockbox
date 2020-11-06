module github.com/cloudflare/lockbox

go 1.15

require (
	github.com/butonic/zerologr v0.0.0-20191210074216-d798ee237d84
	github.com/google/go-cmp v0.5.2
	github.com/kevinburke/nacl v0.0.0-20190829012316-f3ed23dbd7f8
	github.com/oklog/run v1.1.0
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/common v0.4.1
	github.com/rs/zerolog v1.20.0
	k8s.io/api v0.18.6
	k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/yaml v1.2.0
)
