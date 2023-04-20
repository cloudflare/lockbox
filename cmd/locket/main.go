package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"os"
	gruntime "runtime"
	"time"

	lockboxv1 "github.com/cloudflare/lockbox/pkg/apis/lockbox.k8s.cloudflare.com/v1"
	"github.com/cloudflare/lockbox/pkg/flagvar"
	"github.com/go-logr/zerologr"
	"github.com/kevinburke/nacl"
	"github.com/kevinburke/nacl/box"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	input        = flagvar.File{}
	kubeconfig   = flagvar.File{}
	output       = flagvar.Enum{Choices: []string{"json", "yaml"}, Value: "yaml"}
	version      = "dev"
	printVersion bool
	peerHex      string
	masterURL    string
	lockboxNS    string
	lockboxSvc   string
)

func main() {
	flag.Var(&input, "f", fmt.Sprintf("input file (%s)", input.Help()))
	flag.Var(&output, "o", fmt.Sprintf("output format (%s)", output.Help()))
	flag.Var(&kubeconfig, "kubeconfig", fmt.Sprintf("path to kubeconfig. (%s)", kubeconfig.Help()))
	flag.StringVar(&peerHex, "peer-hex", "", "peer public key (32-bit hex)")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&lockboxNS, "lockbox-namespace", "lockbox", "namespace of the lockbox controller")
	flag.StringVar(&lockboxSvc, "lockbox-service", "lockbox", "name of the lockbox service")
	flag.BoolVar(&printVersion, "version", false, "print version")
	flag.String("v", "", "log level for V logs")
	flag.Parse()

	ctx := context.Background()

	if printVersion {
		fmt.Printf("locket: %s\n", version)
		os.Exit(0)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerologr.NameFieldName = "logger"
	zerologr.NameSeparator = "/"

	zl := zerolog.New(os.Stderr).With().Caller().Timestamp().Logger()
	logf.SetLogger(zerologr.New(&zl))
	logger := zl.With().Str("name", "main").Logger()

	err := lockboxv1.AddToScheme(scheme.Scheme)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to add lockbox schemes")
		os.Exit(1)
	}

	var r io.Reader
	if input.String() == "" {
		r = os.Stdin
	} else {
		r, err = os.Open(input.String())
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to open secret file")
			os.Exit(1)
		}
	}

	w := os.Stdout

	cfg := GetConfig()

	cf := runtimeserializer.NewCodecFactory(scheme.Scheme)

	ib, err := io.ReadAll(r)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to read secret file")
		os.Exit(1)
	}
	var secret corev1.Secret
	if err = runtime.DecodeInto(cf.UniversalDecoder(), ib, &secret); err != nil {
		logger.Fatal().Err(err).Msg("unable to decode secret file")
		os.Exit(1)
	}

	pubKey, priKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not generate key")
		os.Exit(1)
	}

	var peerKey nacl.Key
	switch peerHex {
	case "":
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		cc, err := cfg.ClientConfig()
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to create API client configuration")
			os.Exit(1)
		}

		cc.UserAgent = fmt.Sprintf("%s/%s (%s/%s)", os.Args[0], version, gruntime.GOOS, gruntime.GOARCH)

		client, err := kubernetes.NewForConfig(cc)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to create API client")
			os.Exit(1)
		}

		b, err := GetRemotePublicKey(ctx, client, lockboxNS, lockboxSvc)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to fetch public key")
			os.Exit(1)
		}
		if len(b) != 32 {
			err = fmt.Errorf("incorrect peer key length: %d, should be 32", len(b))
			logger.Fatal().Err(err).Msg("unable to fetch peer key")
			return
		}

		peerKey = new([nacl.KeySize]byte)
		copy(peerKey[:], b)
	default:
		peerKey, err = nacl.Load(peerHex)
		if err != nil {
			logger.Fatal().Err(err).Msg("could not load --peer-hex")
			os.Exit(1)
		}
	}

	namespace := secret.Namespace
	if namespace == "" {
		namespace, _, _ = cfg.Namespace()
	}

	b := lockboxv1.NewFromSecret(secret, namespace, peerKey, pubKey, priKey)

	var ct string
	switch output.String() {
	case "yaml":
		ct = "application/yaml"
	case "json":
		ct = "application/json"
	}

	info, ok := runtime.SerializerInfoForMediaType(cf.SupportedMediaTypes(), ct)
	if !ok {
		logger.Fatal().Str("content-type", ct).Msg("can't serialize to content-type")
		os.Exit(1)
	}
	serial := info.Serializer
	if info.PrettySerializer != nil {
		serial = info.PrettySerializer
	}
	enc := cf.EncoderForVersion(serial, lockboxv1.GroupVersion)

	ob, err := runtime.Encode(enc, b)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to encode Lockbox")
		os.Exit(1)
	}

	if _, err := w.Write(ob); err != nil {
		logger.Fatal().Err(err).Send()
	}
	if _, err := w.WriteString("\n"); err != nil {
		logger.Fatal().Err(err).Send()
	}
}

func GetConfig() clientcmd.ClientConfig {
	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := clientcmd.ConfigOverrides{
		ClusterInfo: clientcmdapi.Cluster{
			Server: masterURL,
		},
	}
	loader.ExplicitPath = kubeconfig.String()
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, &overrides)
}

func GetRemotePublicKey(ctx context.Context, c kubernetes.Interface, ns, svc string) ([]byte, error) {
	return c.CoreV1().Services(ns).ProxyGet("http", svc, "", "/v1/public", nil).DoRaw(ctx)
}
