package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	lockboxv1 "github.com/cloudflare/lockbox/pkg/apis/lockbox.k8s.cloudflare.com/v1"
	"github.com/cloudflare/lockbox/pkg/flagvar"
	lockboxcontroller "github.com/cloudflare/lockbox/pkg/lockbox-controller"
	server "github.com/cloudflare/lockbox/pkg/lockbox-server"
	"github.com/cloudflare/lockbox/pkg/statemetrics"
	"github.com/go-logr/zerologr"
	"github.com/kevinburke/nacl"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	pubKey, priKey nacl.Key
	version        = "dev"
	syncPeriod     = 1 * time.Hour
	keypairPath    = flagvar.File{Value: "/etc/lockbox/keypair.yaml"}
	metricsAddr    = flagvar.TCPAddr{Text: ":8080"}
	httpAddr       = flagvar.TCPAddr{Text: ":8081"}
)

func main() {
	flag.Var(&keypairPath, "keypair", fmt.Sprintf("public/private 32 byte keypairs (%s)", keypairPath.Help()))
	flag.Var(&metricsAddr, "metrics-addr", fmt.Sprintf("bind for HTTP metrics (%s)", metricsAddr.Help()))
	flag.Var(&httpAddr, "http-addr", fmt.Sprintf("bind for HTTP server (%s)", httpAddr.Help()))
	flag.DurationVar(&syncPeriod, "sync-period", syncPeriod, "controller sync period")
	flag.String("v", "", "log level for V logs")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerologr.NameFieldName = "logger"
	zerologr.NameSeparator = "/"

	zl := zerolog.New(os.Stderr).With().Caller().Timestamp().Logger()
	logf.SetLogger(zerologr.New(&zl))
	logger := zl.With().Str("name", "main").Logger()

	keypair, err := os.Open(keypairPath.Value)
	if err != nil {
		logger.Fatal().Err(err).Str("path", keypairPath.Value).Msg("unable to open keypair")
		os.Exit(1)
	}
	pubKey, priKey, err = KeyPairFromYAMLOrJSON(keypair)
	if err != nil {
		logger.Fatal().Err(err).Str("path", keypairPath.Value).Msg("unable to parse keypair")
		os.Exit(1)
	}
	_ = keypair.Close()

	err = lockboxv1.AddToScheme(scheme.Scheme)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to add lockbox schemes")
		os.Exit(1)
	}

	cfg, err := config.GetConfig()
	cfg.UserAgent = fmt.Sprintf("%s/%s (%s/%s)", os.Args[0], version, runtime.GOOS, runtime.GOARCH)

	if err != nil {
		logger.Fatal().Err(err).Msg("unable to get kubeconfig")
		os.Exit(1)
	}

	mgr, err := manager.New(cfg, manager.Options{
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr.Text,
		},
		Cache: cache.Options{
			SyncPeriod: &syncPeriod,
		},
		Scheme: scheme.Scheme,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to create controller manager")
		os.Exit(1)
	}

	recorder := mgr.GetEventRecorder("lockbox")
	client := mgr.GetClient()

	sr := lockboxcontroller.NewSecretReconciler(pubKey, priKey, lockboxcontroller.WithRecorder(recorder), lockboxcontroller.WithClient(client))

	info := statemetrics.NewKubernetesVec(statemetrics.KubernetesOpts{
		Name: "kube_lockbox_info",
		Help: "Information about Lockbox",
	}, []string{"namespace", "lockbox"})
	created := statemetrics.NewKubernetesVec(statemetrics.KubernetesOpts{
		Name: "kube_lockbox_created",
		Help: "Unix creation timestamp",
	}, []string{"namespace", "lockbox"})
	resourceVersion := statemetrics.NewKubernetesVec(statemetrics.KubernetesOpts{
		Name: "kube_lockbox_resource_version",
		Help: "Resource version representing a specific version of a Lockbox",
	}, []string{"namespace", "lockbox", "resource_version"})
	lbType := statemetrics.NewKubernetesVec(statemetrics.KubernetesOpts{
		Name: "kube_lockbox_type",
		Help: "Lockbox secret type",
	}, []string{"namespace", "lockbox", "type"})
	peerKey := statemetrics.NewKubernetesVec(statemetrics.KubernetesOpts{
		Name: "kube_lockbox_peer",
		Help: "Lockbox peer key",
	}, []string{"namespace", "lockbox", "peer"})
	labels := statemetrics.NewLabelsVec(statemetrics.KubernetesOpts{
		Name: "kube_lockbox_labels",
		Help: "Kubernetes labels converted to Prometheus labels",
	})
	metrics.Registry.MustRegister(info, created, resourceVersion, lbType, labels, peerKey)

	mh := statemetrics.NewStateMetricProxy(
		info, created, resourceVersion,
		lbType, peerKey, labels,
	)

	if err := builder.ControllerManagedBy(mgr).
		For(&lockboxv1.Lockbox{}).
		Owns(&corev1.Secret{}).
		Watches(&lockboxv1.Lockbox{}, mh).
		Complete(reconcile.AsReconciler(mgr.GetClient(), sr)); err != nil {
		logger.Fatal().Err(err).Send()
	}

	mux := http.NewServeMux()
	mux.Handle("GET /v1/public", server.PublicKey(pubKey))
	if err := mgr.Add(&manager.Server{
		Name: "keyserver",
		Server: &http.Server{
			Handler:           mux,
			Addr:              httpAddr.Text,
			MaxHeaderBytes:    1 << 20,
			IdleTimeout:       90 * time.Second,
			ReadHeaderTimeout: 32 * time.Second,
		},
	}); err != nil {
		logger.Fatal().Err(err).Send()
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		logger.Fatal().Err(err).Send()
	}
}
