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

// Package util contains code shared between the subcommands of crinit.
package util

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
)

const (
	APIServerCN = "clusterregistry"
	AdminCN     = "admin"
)

type EntityKeyPairs struct {
	CA     *triple.KeyPair
	Server *triple.KeyPair
	Admin  *triple.KeyPair
}

type Credentials struct {
	Username        string
	Password        string
	Token           string
	CertEntKeyPairs *EntityKeyPairs
}

// generateCredentials helper to create the certs for the apiserver.
func GenerateCredentials(svcNamespace, name, svcName, localDNSZoneName string,
	ips, hostnames []string, enableHTTPBasicAuth, enableTokenAuth bool) (*Credentials, error) {

	credentials := Credentials{
		Username: AdminCN,
	}
	if enableHTTPBasicAuth {
		credentials.Password = string(uuid.NewUUID())
	}
	if enableTokenAuth {
		credentials.Token = string(uuid.NewUUID())
	}

	entKeyPairs, err := GenCerts(svcNamespace, name, svcName, localDNSZoneName, ips, hostnames)
	if err != nil {
		return nil, err
	}
	credentials.CertEntKeyPairs = entKeyPairs
	return &credentials, nil
}

func GenCerts(svcNamespace, name, svcName, localDNSZoneName string,
	ips, hostnames []string) (*EntityKeyPairs, error) {
	ca, err := triple.NewCA(name)

	if err != nil {
		return nil, fmt.Errorf("failed to create CA key and certificate: %v", err)
	}
	server, err := triple.NewServerKeyPair(ca, APIServerCN, svcName, svcNamespace, localDNSZoneName, ips, hostnames)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster registry API server key and certificate: %v", err)
	}
	admin, err := triple.NewClientKeyPair(ca, AdminCN, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create client key and certificate for an admin: %v", err)
	}
	return &EntityKeyPairs{
		CA:     ca,
		Server: server,
		Admin:  admin,
	}, nil
}

func ArgMapsToArgStrings(argsMap, overrides map[string]string) []string {
	for key, val := range overrides {
		argsMap[key] = val
	}
	args := []string{}
	for key, value := range argsMap {
		args = append(args, fmt.Sprintf("%s=%s", key, value))
	}
	// This is needed for the unit test deep copy to get an exact match.
	sort.Strings(args)
	return args
}

// UpdateKubeconfig helper to update the kubeconfig file based on input
// parameters.
func UpdateKubeconfig(pathOptions *clientcmd.PathOptions, name, endpoint,
	kubeConfigPath string, credentials *Credentials, dryRun bool) error {

	pathOptions.LoadingRules.ExplicitPath = kubeConfigPath
	kubeconfig, err := pathOptions.GetStartingConfig()
	if err != nil {
		return err
	}

	// Populate API server endpoint info.
	cluster := clientcmdapi.NewCluster()

	// Prefix "https" as the URL scheme to endpoint.
	if !strings.HasPrefix(endpoint, "https://") {
		endpoint = fmt.Sprintf("https://%s", endpoint)
	}

	cluster.Server = endpoint
	cluster.CertificateAuthorityData = certutil.EncodeCertPEM(credentials.CertEntKeyPairs.CA.Cert)

	// Populate credentials.
	authInfo := clientcmdapi.NewAuthInfo()
	authInfo.ClientCertificateData = certutil.EncodeCertPEM(credentials.CertEntKeyPairs.Admin.Cert)
	authInfo.ClientKeyData = certutil.EncodePrivateKeyPEM(credentials.CertEntKeyPairs.Admin.Key)
	authInfo.Token = credentials.Token

	var httpBasicAuthInfo *clientcmdapi.AuthInfo

	if credentials.Password != "" {
		httpBasicAuthInfo = clientcmdapi.NewAuthInfo()
		httpBasicAuthInfo.Password = credentials.Password
		httpBasicAuthInfo.Username = credentials.Username
	}

	// Populate context.
	context := clientcmdapi.NewContext()
	context.Cluster = name
	context.AuthInfo = name

	// Update the config struct with API server endpoint info,
	// credentials and context.
	kubeconfig.Clusters[name] = cluster
	kubeconfig.AuthInfos[name] = authInfo

	if httpBasicAuthInfo != nil {
		kubeconfig.AuthInfos[fmt.Sprintf("%s-basic-auth", name)] = httpBasicAuthInfo
	}

	kubeconfig.Contexts[name] = context

	if !dryRun {
		// Write the update kubeconfig.
		if err := clientcmd.ModifyConfig(pathOptions, *kubeconfig, true); err != nil {
			return err
		}
	}

	return nil
}

// DeleteKubeconfigEntry helper to delete the kubeconfig file entry based on input
// parameters.
func DeleteKubeconfigEntry(out io.Writer, pathOptions *clientcmd.PathOptions, name,
	kubeConfigPath string, dryRun, ignoreErrors bool) error {

	pathOptions.LoadingRules.ExplicitPath = kubeConfigPath
	kubeconfig, err := pathOptions.GetStartingConfig()
	if err != nil {
		return err
	}

	kubeconfigFile := pathOptions.GetDefaultFilename()
	if pathOptions.IsExplicitFile() {
		kubeconfigFile = pathOptions.GetExplicitFile()
	}

	if dryRun {
		return nil
	}

	// If we are not going to ignore errors, then return an error immediately
	// on the first error encountered. If ignoring errors, then only output
	// errors when verbose logging is enabled. If it turns out that all three:
	// context, cluster, and authinfo were not found, then return an error as
	// there is no point in updating the kubeconfig; returning an error in this
	// case is okay because the ignore errors flag will also be checked by a
	// caller of this function.
	errCount := 0

	_, ok := kubeconfig.Contexts[name]
	if !ok {
		if !ignoreErrors {
			return fmt.Errorf("cannot delete context %s, not in %s", name, kubeconfigFile)
		}
		glog.V(4).Infof("cannot delete context %s, not in %s", name, kubeconfigFile)
		errCount++
	} else {
		delete(kubeconfig.Contexts, name)
	}

	_, ok = kubeconfig.Clusters[name]
	if !ok {
		if !ignoreErrors {
			return fmt.Errorf("cannot delete cluster %s, not in %s", name, kubeconfigFile)
		}
		glog.V(4).Infof("cannot delete cluster %s, not in %s", name, kubeconfigFile)
		errCount++
	} else {
		delete(kubeconfig.Clusters, name)
	}

	_, ok = kubeconfig.AuthInfos[name]
	if !ok {
		if !ignoreErrors {
			return fmt.Errorf("cannot delete authinfo %s, not in %s", name, kubeconfigFile)
		}
		glog.V(4).Infof("cannot delete authinfo %s, not in %s", name, kubeconfigFile)
		errCount++
	} else {
		delete(kubeconfig.AuthInfos, name)
	}

	if errCount == 3 {
		return fmt.Errorf("Could not find any cluster registry context, cluster, or authinfo information for %s in your kubeconfig.",
			name)
	}

	// Write the updated kubeconfig.
	if err := clientcmd.ModifyConfig(pathOptions, *kubeconfig, true); err != nil {
		return err
	}

	glog.V(4).Infof("deleted kubeconfig entry %s from %s\n", name, kubeconfigFile)

	if kubeconfig.CurrentContext == name {
		fmt.Fprint(out,
			"warning: this removed your active context, use \"kubectl config use-context\" to select a different one\n")
	}

	return nil
}

func PrintSuccess(cmdOut io.Writer, ips, hostnames []string, svc *v1.Service) error {
	svcEndpoints := append(ips, hostnames...)
	endpoints := strings.Join(svcEndpoints, ", ")
	if svc.Spec.Type == v1.ServiceTypeNodePort {
		endpoints = ips[0] + ":" + strconv.Itoa(int(svc.Spec.Ports[0].NodePort))
		if len(ips) > 1 {
			endpoints = endpoints + ", ..."
		}
	}

	_, err := fmt.Fprintf(cmdOut, "Cluster registry API server is running at: %s\n", endpoints)
	return err
}

// GetClientConfig gets a ClientConfig for the proivided context in kubeconfig
// file referenced by kubeconfigPath.
func GetClientConfig(pathOptions *clientcmd.PathOptions, context, kubeconfigPath string) clientcmd.ClientConfig {
	loadingRules := *pathOptions.LoadingRules
	loadingRules.Precedence = pathOptions.GetLoadingPrecedence()
	loadingRules.ExplicitPath = kubeconfigPath
	overrides := &clientcmd.ConfigOverrides{
		CurrentContext: context,
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(&loadingRules, overrides)
}

// AuthFileContents returns a CSV string containing the contents of an
// authentication file in the format required by the cluster registry.
func AuthFileContents(username, authSecret string) []byte {
	return []byte(fmt.Sprintf("%s,%s,%s\n", authSecret, username, uuid.NewUUID()))
}

// GetCAKeyPair retrieves the CA key pair stored in the internal credentials
// structure.
func GetCAKeyPair(credentials *Credentials) *triple.KeyPair {
	if credentials == nil {
		glog.V(4).Info("credentials argument is nil!")
		return nil
	}

	return credentials.CertEntKeyPairs.CA
}
