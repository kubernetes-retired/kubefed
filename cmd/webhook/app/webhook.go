package app

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	ctrwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	"sigs.k8s.io/kubefed/pkg/controller/webhook/federatedtypeconfig"
	"sigs.k8s.io/kubefed/pkg/controller/webhook/kubefedcluster"
	"sigs.k8s.io/kubefed/pkg/controller/webhook/kubefedconfig"
	"sigs.k8s.io/kubefed/pkg/version"
)

var (
	certDir, kubeconfig, masterURL string
	port                           = 8443
)

// NewWebhookCommand creates a *cobra.Command object with default parameters
func NewWebhookCommand(stopChan <-chan struct{}) *cobra.Command {
	verFlag := false

	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Start a kubefed webhook server",
		Long:  "Start a kubefed webhook server",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stdout, "KubeFed webhook version: %s\n", fmt.Sprintf("%#v", version.Get()))
			if verFlag {
				os.Exit(0)
			}
			// PrintFlags(cmd.Flags())

			if err := Run(stopChan); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	// Add the command line flags from other dependencies(klog, kubebuilder, etc.)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)

	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	cmd.Flags().StringVar(&certDir, "cert-dir", "", "The directory where the TLS certs are located.")
	cmd.Flags().IntVar(&port, "secure-port", port, "The port on which to serve HTTPS.")

	return cmd
}

// Run runs the webhook with options. This should never exit.
func Run(stopChan <-chan struct{}) error {
	logs.InitLogs()
	defer logs.FlushLogs()

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("error setting up webhook's config: %s", err)
	}
	mgr, err := manager.New(config, manager.Options{
		Port:    port,
		CertDir: certDir,
	})
	if err != nil {
		klog.Fatalf("error setting up webhook manager: %s", err)
	}
	hookServer := mgr.GetWebhookServer()

	hookServer.Register("/validate-federatedtypeconfigs", &ctrwebhook.Admission{Handler: &federatedtypeconfig.FederatedTypeConfigAdmissionHook{}})
	hookServer.Register("/validate-kubefedcluster", &ctrwebhook.Admission{Handler: &kubefedcluster.KubeFedClusterAdmissionHook{}})
	hookServer.Register("/validate-kubefedconfig", &ctrwebhook.Admission{Handler: &kubefedconfig.KubeFedConfigValidator{}})
	hookServer.Register("/default-kubefedconfig", &ctrwebhook.Admission{Handler: &kubefedconfig.KubeFedConfigDefaulter{}})

	hookServer.WebhookMux.Handle("/readyz/", http.StripPrefix("/readyz/", &healthz.Handler{}))

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		klog.Fatalf("unable to run manager: %s", err)
	}

	return nil
}
