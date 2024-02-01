package main

import (
	"context"
	"flag"
	"fmt"
	"net"
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
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
	keypair.Close()

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

	recorder := mgr.GetEventRecorderFor("lockbox")
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
		&handler.EnqueueRequestForObject{},
		info, created, resourceVersion,
		lbType, peerKey, labels,
	)

	c, err := controller.New("lockbox-controller", mgr, controller.Options{
		Reconciler: reconcile.AsReconciler(mgr.GetClient(), sr),
	})

	if err != nil {
		logger.Fatal().Err(err).Msg("unable to create controller")
		os.Exit(1)
	}

	if err := c.Watch(source.Kind(mgr.GetCache(), &lockboxv1.Lockbox{}), mh); err != nil {
		logger.Fatal().Err(err).Msg("unable to watch Lockbox resources")
		os.Exit(1)
	}

	if err := c.Watch(source.Kind(mgr.GetCache(), &corev1.Secret{}), handler.EnqueueRequestForOwner(scheme.Scheme, mgr.GetRESTMapper(), &lockboxv1.Lockbox{}, handler.OnlyControllerOwner())); err != nil {
		logger.Fatal().Err(err).Msg("unable to watch Secret resources")
		os.Exit(1)
	}

	// TODO(terin): make server implement Runnable
	if err := mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		mux := http.NewServeMux()
		mux.Handle("/v1/public", server.PublicKey(pubKey))

		ln, err := net.Listen("tcp", httpAddr.Text)
		if err != nil {
			return err
		}

		// sig.kubernetes.io/controller-runtime/pkg/internal/httpserver
		s := http.Server{
			Handler:           mux,
			MaxHeaderBytes:    1 << 20,
			IdleTimeout:       90 * time.Second,
			ReadHeaderTimeout: 32 * time.Second,
		}

		idleConnsClosed := make(chan struct{})
		go func() {
			<-ctx.Done()

			if err := s.Shutdown(context.Background()); err != nil {
				logger.Err(err).Send()
			}
			close(idleConnsClosed)
		}()

		if err := s.Serve(ln); err != nil && err != http.ErrServerClosed {
			return err
		}

		<-idleConnsClosed
		return nil
	})); err != nil {
		logger.Fatal().Err(err).Msg("unable to add server runnable")
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		logger.Fatal().Err(err).Send()
	}
}
