package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/butonic/zerologr"
	lockboxv1 "github.com/cloudflare/lockbox/pkg/apis/lockbox.k8s.cloudflare.com/v1"
	"github.com/cloudflare/lockbox/pkg/flagvar"
	lockboxcontroller "github.com/cloudflare/lockbox/pkg/lockbox-controller"
	server "github.com/cloudflare/lockbox/pkg/lockbox-server"
	"github.com/cloudflare/lockbox/pkg/statemetrics"
	"github.com/kevinburke/nacl"
	"github.com/oklog/run"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	rlog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
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

	logger := zerolog.New(os.Stderr)
	log := zerologr.NewWithOptions(zerologr.Options{
		Logger: &logger,
	})
	rlog.SetLogger(log)

	logger = logger.With().Str("name", "main").Logger()

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

	err = lockboxv1.Install(scheme.Scheme)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to add lockbox schemes")
		os.Exit(1)
	}

	cfg, err := config.GetConfig()
	cfg.UserAgent = fmt.Sprintf("%s/%s (%s/%s)", os.Args[0], version, runtime.GOOS, runtime.GOARCH)

	if err != nil {
		logger.Fatal().Err(err).Msg("unnable to get kubeconfig")
		os.Exit(1)
	}

	mgr, err := manager.New(cfg, manager.Options{
		MetricsBindAddress: metricsAddr.Text,
		SyncPeriod:         &syncPeriod,
		Scheme:             scheme.Scheme,
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
		Reconciler: sr,
	})

	if err != nil {
		logger.Fatal().Err(err).Msg("unable to create controller")
		os.Exit(1)
	}

	if err := c.Watch(&source.Kind{Type: &lockboxv1.Lockbox{}}, mh); err != nil {
		logger.Fatal().Err(err).Msg("unable to watch Lockbox resources")
		os.Exit(1)
	}

	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &lockboxv1.Lockbox{},
		IsController: true,
	}); err != nil {
		logger.Fatal().Err(err).Msg("unable to watch Secret resources")
		os.Exit(1)
	}

	var g run.Group
	{
		stop := signals.SetupSignalHandler()
		cancel := make(chan struct{})
		g.Add(func() error {
			select {
			case <-stop:
				return nil
			case <-cancel:
				return nil
			}
		}, func(err error) {
			close(cancel)
		})
	}
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			if err := mgr.Start(cancel); err != nil {
				logger.Error().Err(err).Msg("unable to start manager")
				return err
			}
			return nil
		}, func(err error) {
			close(cancel)
		})
	}
	{
		mux := http.NewServeMux()
		mux.Handle("/v1/public", server.PublicKey(pubKey))
		ln, err := net.Listen("tcp", httpAddr.Text)
		if err != nil {
			logger.Error().Err(err).Msg("unable to start HTTP server")
			os.Exit(-1)
		}

		g.Add(func() error {
			s := http.Server{
				Handler:      mux,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 10 * time.Second,
			}
			return s.Serve(ln)
		}, func(err error) {
			ln.Close()
		})
	}

	g.Run()
}
