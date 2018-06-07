/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package options contains implemention for SubcommandOptions.
package options

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/api/core/v1"

	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/cluster-registry/pkg/crinit/common"
	"k8s.io/cluster-registry/pkg/crinit/util"
)

const (
	// DefaultClusterRegistryNamespace is the default namespace in which
	// cluster registry components are hosted.
	DefaultClusterRegistryNamespace = "clusterregistry"

	HostClusterLocalDNSZoneName = "cluster.local."
	APIServerNameSuffix         = "apiserver"
	CredentialSuffix            = "credentials"

	APIServerAdvertiseAddressFlag = "api-server-advertise-address"
	APIServerServiceTypeFlag      = "api-server-service-type"
	apiserverPortFlag             = "api-server-port"
)

var (
	serverName     string
	serverCredName string
)

// SubcommandOptions holds the configuration required by the subcommands of
// `clusterregistry`.
type SubcommandOptions struct {
	Name                      string
	Host                      string
	ClusterRegistryNamespace  string
	Kubeconfig                string
	ServerImage               string
	EtcdImage                 string
	EtcdPVCapacity            string
	EtcdPVStorageClass        string
	EtcdPersistentStorage     bool
	DryRun                    bool
	IgnoreErrors              bool
	APIServerOverridesString  string
	APIServerOverrides        map[string]string
	APIServerServiceType      v1.ServiceType
	APIServerAdvertiseAddress string
	APIServerNodePortPort     int32
	APIServerNodePortPortPtr  *int32
}

// BindCommon adds the common options that are shared by all commands to the list
// of flags.
func (o *SubcommandOptions) BindCommon(flags *pflag.FlagSet) {
	flags.StringVar(&o.Kubeconfig, "kubeconfig", "",
		"Path to the kubeconfig file to use for CLI requests.")
	flags.StringVar(&o.Host, "host-cluster-context", "",
		"Context of the cluster in which to host the cluster registry.")
	flags.StringVar(&o.ClusterRegistryNamespace, "cluster-registry-namespace",
		DefaultClusterRegistryNamespace,
		"Namespace in the host cluster where the cluster registry components are installed")
	flags.BoolVar(&o.DryRun, "dry-run", false,
		"Run the command in dry-run mode, without making any server requests.")
}

// BindCommonInit adds the common options that are shared by the aggregated and
// standalone init sub-commands to the list of flags.
func (o *SubcommandOptions) BindCommonInit(flags *pflag.FlagSet, defaultServerImage, defaultEtcdImage string) {
	flags.StringVar(&o.ServerImage, "image", defaultServerImage,
		"Image to use for the cluster registry API server binary.")
	flags.StringVar(&o.EtcdImage, "etcd-image", defaultEtcdImage,
		"Image to use for the etcd server binary.")
	flags.StringVar(&o.EtcdPVCapacity, "etcd-pv-capacity", "10Gi",
		"Size of the persistent volume claim to be used for etcd.")
	flags.StringVar(&o.EtcdPVStorageClass, "etcd-pv-storage-class", "",
		"The storage class of the persistent volume claim used for etcd. Must be provided if a default storage class is not enabled for the host cluster.")
	flags.BoolVar(&o.EtcdPersistentStorage, "etcd-persistent-storage", true,
		"Use a persistent volume for etcd. Defaults to 'true'.")
	flags.StringVar(&o.APIServerOverridesString, "apiserver-arg-overrides", "",
		"Comma-separated list of cluster registry API server arguments to override, e.g., \"--arg1=value1,--arg2=value2...\"")
	flags.StringVar(&o.APIServerAdvertiseAddress, APIServerAdvertiseAddressFlag, "",
		"Preferred address at which to advertise the cluster registry API server NodePort service. Valid only if '"+APIServerServiceTypeFlag+"=NodePort'.")
	flags.Int32Var(&o.APIServerNodePortPort, apiserverPortFlag, 0,
		"Preferred port to use for the cluster registry API server NodePort service. Set to 0 to randomly assign a port. Valid only if '"+APIServerServiceTypeFlag+"=NodePort'.")
}

// BindCommonDelete adds the common options that are shared by the aggregated and
// standalone delete sub-commands to the list of flags.
func (o *SubcommandOptions) BindCommonDelete(flags *pflag.FlagSet) {
	flags.BoolVar(&o.IgnoreErrors, "ignore-errors", false,
		"Run the command and ignore errors encountered while deleting all resources previously created by init.")
}

// SetName sets the name of the cluster registry.
func (o *SubcommandOptions) SetName(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("NAME is required")
	}
	o.Name = args[0]
	return nil
}

// ValidateCommonOptions validates the options that are shared across
// different sub-commands.
func (o *SubcommandOptions) ValidateCommonOptions() error {
	serverName = fmt.Sprintf("%s-%s", o.Name, APIServerNameSuffix)
	serverCredName = fmt.Sprintf("%s-%s", serverName, CredentialSuffix)

	if o.APIServerServiceType != v1.ServiceTypeLoadBalancer &&
		o.APIServerServiceType != v1.ServiceTypeNodePort {
		return fmt.Errorf("invalid %s: %s, should be either %s or %s",
			APIServerServiceTypeFlag, o.APIServerServiceType,
			v1.ServiceTypeLoadBalancer, v1.ServiceTypeNodePort)
	}

	if o.APIServerAdvertiseAddress != "" {
		ip := net.ParseIP(o.APIServerAdvertiseAddress)
		if ip == nil {
			return fmt.Errorf("invalid %s: %s, should be a valid ip address",
				APIServerAdvertiseAddressFlag, o.APIServerAdvertiseAddress)
		}
		if o.APIServerServiceType != v1.ServiceTypeNodePort {
			return fmt.Errorf("%s should be passed only with '%s=NodePort'",
				APIServerAdvertiseAddressFlag, APIServerServiceTypeFlag)
		}
	}

	if o.APIServerNodePortPort != 0 {
		if o.APIServerServiceType != v1.ServiceTypeNodePort {
			return fmt.Errorf("%s should be passed only with '%s=NodePort'",
				apiserverPortFlag, APIServerServiceTypeFlag)
		}
		o.APIServerNodePortPortPtr = &o.APIServerNodePortPort
	} else {
		o.APIServerNodePortPortPtr = nil
	}

	if o.APIServerNodePortPort < 0 || o.APIServerNodePortPort > 65535 {
		return fmt.Errorf("Please provide a valid port number for %s", apiserverPortFlag)
	}

	return nil
}

// marshalOptions marshals options if necessary.
func (o *SubcommandOptions) MarshalOptions() error {
	if o.APIServerOverridesString == "" {
		return nil
	}

	argsMap := make(map[string]string)
	overrideArgs := strings.Split(o.APIServerOverridesString, ",")
	for _, overrideArg := range overrideArgs {
		splitArg := strings.SplitN(overrideArg, "=", 2)
		if len(splitArg) != 2 {
			return fmt.Errorf("wrong format for override arg: %s", overrideArg)
		}
		key := strings.TrimSpace(splitArg[0])
		val := strings.TrimSpace(splitArg[1])
		if len(key) == 0 {
			return fmt.Errorf("wrong format for override arg: %s, arg name cannot be empty", overrideArg)
		}
		argsMap[key] = val
	}

	o.APIServerOverrides = argsMap

	return nil
}

// CreateNamespace creates the cluster registry namespace.
func (o *SubcommandOptions) CreateNamespace(cmdOut io.Writer,
	clientset client.Interface) error {

	fmt.Fprintf(cmdOut, "Creating a namespace %s for the cluster registry...",
		o.ClusterRegistryNamespace)
	glog.V(4).Infof("Creating a namespace %s for the cluster registry",
		o.ClusterRegistryNamespace)

	_, err := common.CreateNamespace(clientset, o.ClusterRegistryNamespace, o.DryRun)

	if err != nil {
		return err
	}

	fmt.Fprintln(cmdOut, " done")
	return err
}

// CreateService creates the cluster registry apiserver service.
func (o *SubcommandOptions) CreateService(cmdOut io.Writer,
	clientset client.Interface) (*v1.Service, []string, []string, error) {

	fmt.Fprint(cmdOut, "Creating cluster registry API server service...")
	glog.V(4).Info("Creating cluster registry API server service")

	svc, ips, hostnames, err := common.CreateService(cmdOut, clientset,
		o.ClusterRegistryNamespace, o.Name, o.APIServerAdvertiseAddress,
		o.APIServerNodePortPortPtr, o.APIServerServiceType, o.DryRun)

	if err != nil {
		return nil, nil, nil, err
	}

	fmt.Fprintln(cmdOut, " done")
	glog.V(4).Infof("Created service named %s with IP addresses %v, hostnames %v",
		svc.Name, ips, hostnames)

	return svc, ips, hostnames, err
}

// GenerateCredentials creates the credentials for apiserver secret.
func (o *SubcommandOptions) GenerateCredentials(cmdOut io.Writer, svcName string,
	ips, hostnames []string, apiServerEnableHTTPBasicAuth,
	apiServerEnableTokenAuth bool) (*util.Credentials, error) {

	fmt.Fprint(cmdOut,
		"Creating cluster registry objects (credentials, persistent volume claim)...")
	glog.V(4).Info("Generating TLS certificates and credentials for communicating with the cluster registry API server")

	credentials, err := util.GenerateCredentials(o.ClusterRegistryNamespace, o.Name,
		svcName, HostClusterLocalDNSZoneName, ips, hostnames,
		apiServerEnableHTTPBasicAuth, apiServerEnableTokenAuth)

	if err != nil {
		return nil, err
	}

	return credentials, nil
}

// CreateAPIServerCredentialsSecret creates the secret containing the
// apiserver credentials passed in.
func (o *SubcommandOptions) CreateAPIServerCredentialsSecret(clientset client.Interface,
	credentials *util.Credentials) error {

	_, err := common.CreateAPIServerCredentialsSecret(clientset,
		o.ClusterRegistryNamespace, serverCredName, credentials, o.DryRun)

	if err != nil {
		return err
	}

	glog.V(4).Info("Certificates and credentials generated")
	return nil
}

func (o *SubcommandOptions) CreatePVC(cmdOut io.Writer,
	clientset client.Interface, svcName string) (*v1.PersistentVolumeClaim, error) {

	if !o.EtcdPersistentStorage {
		return nil, nil
	}

	glog.V(4).Info("Creating a persistent volume and a claim to store the cluster registry API server's state, including etcd data")

	pvc, err := common.CreatePVC(clientset, o.ClusterRegistryNamespace, svcName,
		o.EtcdPVCapacity, o.EtcdPVStorageClass, o.DryRun)

	if err != nil {
		return nil, err
	}

	glog.V(4).Info("Persistent volume and claim created")
	fmt.Fprintln(cmdOut, " done")

	return pvc, nil
}

func (o *SubcommandOptions) CreateAPIServer(cmdOut io.Writer, clientset client.Interface,
	apiServerEnableHTTPBasicAuth, apiServerEnableTokenAuth, aggregated bool, ips []string,
	pvc *v1.PersistentVolumeClaim, serviceAccountName string) error {
	// Since only one IP address can be specified as advertise address,
	// we arbitrarily pick the first available IP address.
	// Pick user provided APIServerAdvertiseAddress over other available IP addresses.
	advertiseAddress := o.APIServerAdvertiseAddress
	if advertiseAddress == "" && len(ips) > 0 {
		advertiseAddress = ips[0]
	}

	fmt.Fprint(cmdOut, "Creating cluster registry deployment...")
	glog.V(4).Info("Creating cluster registry deployment")

	_, err := common.CreateAPIServer(clientset, o.ClusterRegistryNamespace,
		serverName, o.ServerImage, o.EtcdImage, advertiseAddress, serverCredName,
		serviceAccountName, apiServerEnableHTTPBasicAuth, apiServerEnableTokenAuth,
		o.APIServerOverrides, pvc, aggregated, o.DryRun)

	if err != nil {
		glog.V(4).Infof("Failed to create API server: %v", err)
		return err
	}

	fmt.Fprintln(cmdOut, " done")
	glog.V(4).Info("Successfully created cluster registry deployment")

	return nil
}

// UpdateKubeconfig handles updating the kubeconfig by building up the endpoint
// while printing and logging progress.
func (o *SubcommandOptions) UpdateKubeconfig(cmdOut io.Writer,
	pathOptions *clientcmd.PathOptions, svc *v1.Service, ips, hostnames []string,
	credentials *util.Credentials) error {

	fmt.Fprint(cmdOut, "Updating kubeconfig...")
	glog.V(4).Info("Updating kubeconfig")

	// Pick the first ip/hostname to update the api server endpoint in kubeconfig
	// and also to give information to user.
	// In case of NodePort Service for api server, ips are node external ips.
	endpoint := ""
	if len(ips) > 0 {
		endpoint = ips[0]
	} else if len(hostnames) > 0 {
		endpoint = hostnames[0]
	}

	// If the service is nodeport, need to append the port to endpoint as it is
	// non-standard port.
	if o.APIServerServiceType == v1.ServiceTypeNodePort {
		endpoint = endpoint + ":" + strconv.Itoa(int(svc.Spec.Ports[0].NodePort))
	}

	err := util.UpdateKubeconfig(pathOptions, o.Name, endpoint, o.Kubeconfig,
		credentials, o.DryRun)

	if err != nil {
		glog.V(4).Infof("Failed to update kubeconfig: %v", err)
		return err
	}

	fmt.Fprintln(cmdOut, " done")
	glog.V(4).Info("Successfully updated kubeconfig")
	return nil
}

func (o *SubcommandOptions) WaitForAPIServer(cmdOut io.Writer,
	clientset client.Interface, pathOptions *clientcmd.PathOptions,
	ips, hostnames []string, svc *v1.Service) error {

	if o.DryRun {
		_, err := fmt.Fprintln(cmdOut, "Cluster registry can be run (dry run)")
		glog.V(4).Info("Cluster registry can be run (dry run)")
		return err
	}

	fmt.Fprint(cmdOut, "Waiting for the cluster registry API server to come up...")
	glog.V(4).Info("Waiting for the cluster registry API server to come up")

	err := common.WaitForPods(cmdOut, clientset, []string{serverName},
		o.ClusterRegistryNamespace)

	if err != nil {
		return err
	}

	switch o.APIServerServiceType {
	case v1.ServiceTypeLoadBalancer:
		crConfig, err := util.GetClientConfig(pathOptions, o.Name, o.Kubeconfig).ClientConfig()
		if err != nil {
			return err
		}

		crClientset, err := client.NewForConfig(crConfig)
		if err != nil {
			return err
		}

		err = common.WaitSrvHealthy(cmdOut, crClientset)
		if err != nil {
			return err
		}
	case v1.ServiceTypeNodePort:
		fmt.Fprintf(cmdOut, "\nThe cluster registry API pods are running, but "+
			"they are exposed by a NodePort service, so this tool cannot reliably "+
			"verify that they are accessible. You will need to ensure that your "+
			"nodes are routable from this machine in order for kubectl to work "+
			"as-is with the '%v' context.\n", o.Name)
	default:
		return fmt.Errorf("Unexpected service type: %v", o.APIServerServiceType)
	}

	glog.V(4).Info("Cluster registry running")
	fmt.Fprintln(cmdOut, " done")
	return util.PrintSuccess(cmdOut, ips, hostnames, svc)
}
