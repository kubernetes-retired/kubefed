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

// Package common contains code shared between the subcommands of crinit.
package common

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/golang/glog"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	certutil "k8s.io/client-go/util/cert"
	"k8s.io/cluster-registry/pkg/crinit/util"
)

const (
	lbAddrRetryInterval = 5 * time.Second
	waitInterval        = 2 * time.Second
	waitTimeout         = 3 * time.Minute

	apiServerSecurePortName = "https"
	// Set the secure port to 8443 to avoid requiring root privileges
	// to bind to port < 1000.  The apiserver's service will still
	// expose on port 443.
	apiServerSecurePort = 8443
)

var (
	apiserverPodLabels = map[string]string{
		"app":    "clusterregistry",
		"module": "clusterregistry-apiserver",
	}

	ComponentLabel = map[string]string{
		"app": "clusterregistry",
	}

	apiserverSvcSelector = map[string]string{
		"app":    "clusterregistry",
		"module": "clusterregistry-apiserver",
	}
)

// CreateNamespace helper to create the cluster registry namespace object and return
// the object.
func CreateNamespace(clientset client.Interface, namespace string,
	dryRun bool) (*v1.Namespace, error) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	if dryRun {
		return ns, nil
	}

	return clientset.CoreV1().Namespaces().Create(ns)
}

// DeleteNamespace deletes the cluster registry namespace.
func DeleteNamespace(cmdOut io.Writer, clientset client.Interface,
	namespace string, dryRun bool) error {

	fmt.Fprintf(cmdOut, "Deleting cluster registry namespace %s...",
		namespace)
	glog.V(4).Infof("Deleting cluster registry namespace %s",
		namespace)

	err := deleteNamespaceObject(clientset, namespace, dryRun)

	if err != nil {
		return err
	}

	fmt.Fprintln(cmdOut, " done")
	return err
}

// deleteNamespaceObject deletes the cluster registry namespace object and
// returns any errors.
func deleteNamespaceObject(clientset client.Interface, namespace string,
	dryRun bool) error {

	if dryRun {
		return nil
	}

	return clientset.CoreV1().Namespaces().Delete(namespace,
		&metav1.DeleteOptions{})
}

// CreateService helper to create the cluster registry apiserver service object
// and return the object. If service type is load balancer, will wait for load
// balancer IP and return it and hostnames.
func CreateService(cmdOut io.Writer, clientset client.Interface, namespace,
	svcName, apiserverAdvertiseAddress string, apiserverPort *int32,
	apiserverServiceType v1.ServiceType,
	dryRun bool) (*v1.Service, []string, []string, error) {

	port := v1.ServicePort{
		Name:       "https",
		Protocol:   "TCP",
		Port:       443,
		TargetPort: intstr.FromString(apiServerSecurePortName),
	}

	if apiserverServiceType == v1.ServiceTypeNodePort && apiserverPort != nil {
		port.NodePort = *apiserverPort
	}

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: namespace,
			Labels:    ComponentLabel,
		},
		Spec: v1.ServiceSpec{
			Type:     v1.ServiceType(apiserverServiceType),
			Selector: apiserverSvcSelector,
			Ports:    []v1.ServicePort{port},
		},
	}

	if dryRun {
		return svc, nil, nil, nil
	}

	var err error
	svc, err = clientset.CoreV1().Services(namespace).Create(svc)
	if err != nil {
		return nil, nil, nil, err
	}

	ips := []string{}
	hostnames := []string{}
	if apiserverServiceType == v1.ServiceTypeLoadBalancer {
		ips, hostnames, err = WaitForLoadBalancerAddress(cmdOut, clientset, svc, dryRun)
	} else {
		if apiserverAdvertiseAddress != "" {
			ips = append(ips, apiserverAdvertiseAddress)
		} else {
			ips, err = GetClusterNodeIPs(clientset)
		}
	}
	if err != nil {
		return nil, nil, nil, err
	}

	return svc, ips, hostnames, err
}

// GetClusterNodeIPs returns a list of the IP addresses of nodes in the cluster,
// with a preference for external IP addresses.
func GetClusterNodeIPs(clientset client.Interface) ([]string, error) {
	preferredAddressTypes := []v1.NodeAddressType{
		v1.NodeExternalIP,
		v1.NodeInternalIP,
	}
	nodeList, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	nodeAddresses := []string{}
	for _, node := range nodeList.Items {
	OuterLoop:
		for _, addressType := range preferredAddressTypes {
			for _, address := range node.Status.Addresses {
				if address.Type == addressType {
					nodeAddresses = append(nodeAddresses, address.Address)
					break OuterLoop
				}
			}
		}
	}

	return nodeAddresses, nil
}

// CreateAPIServerCredentialsSecret helper to create secret object and return
// the object.
func CreateAPIServerCredentialsSecret(clientset client.Interface, namespace,
	credentialsName string, credentials *util.Credentials, dryRun bool) (*v1.Secret, error) {
	// Build the secret object with API server credentials.
	data := map[string][]byte{
		"ca.crt":     certutil.EncodeCertPEM(credentials.CertEntKeyPairs.CA.Cert),
		"server.crt": certutil.EncodeCertPEM(credentials.CertEntKeyPairs.Server.Cert),
		"server.key": certutil.EncodePrivateKeyPEM(credentials.CertEntKeyPairs.Server.Key),
	}
	if credentials.Password != "" {
		data["basicauth.csv"] = util.AuthFileContents(credentials.Username, credentials.Password)
	}
	if credentials.Token != "" {
		data["token.csv"] = util.AuthFileContents(credentials.Username, credentials.Token)
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      credentialsName,
			Namespace: namespace,
		},
		Data: data,
	}

	if dryRun {
		return secret, nil
	}

	return clientset.CoreV1().Secrets(namespace).Create(secret)
}

// CreatePVC helper to create the persistent volume claim object and
// return the object.
func CreatePVC(clientset client.Interface, namespace, svcName, etcdPVCapacity,
	etcdPVStorageClass string, dryRun bool) (*v1.PersistentVolumeClaim, error) {
	capacity, err := resource.ParseQuantity(etcdPVCapacity)
	if err != nil {
		return nil, err
	}

	var storageClassName *string
	if len(etcdPVStorageClass) > 0 {
		storageClassName = &etcdPVStorageClass
	}

	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-etcd-claim", svcName),
			Namespace: namespace,
			Labels:    ComponentLabel,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: capacity,
				},
			},
			StorageClassName: storageClassName,
		},
	}

	if dryRun {
		return pvc, nil
	}

	return clientset.CoreV1().PersistentVolumeClaims(namespace).Create(pvc)
}

// CreateAPIServer helper to create the apiserver deployment object and
// return the object.
func CreateAPIServer(clientset client.Interface, namespace, name, serverImage,
	etcdImage, advertiseAddress, credentialsName, serviceAccountName string, hasHTTPBasicAuthFile,
	hasTokenAuthFile bool, argOverrides map[string]string,
	pvc *v1.PersistentVolumeClaim, aggregated, dryRun bool) (*appsv1beta1.Deployment, error) {

	command := []string{"./clusterregistry"}
	argsMap := map[string]string{
		"--bind-address":         "0.0.0.0",
		"--etcd-servers":         "http://localhost:2379",
		"--secure-port":          fmt.Sprintf("%d", apiServerSecurePort),
		"--client-ca-file":       "/etc/clusterregistry/apiserver/ca.crt",
		"--tls-cert-file":        "/etc/clusterregistry/apiserver/server.crt",
		"--tls-private-key-file": "/etc/clusterregistry/apiserver/server.key",
	}

	if advertiseAddress != "" {
		argsMap["--advertise-address"] = advertiseAddress
	}
	if hasHTTPBasicAuthFile {
		argsMap["--basic-auth-file"] = "/etc/clusterregistry/apiserver/basicauth.csv"
	}
	if hasTokenAuthFile {
		argsMap["--token-auth-file"] = "/etc/clusterregistry/apiserver/token.csv"
	}

	if aggregated {
		command = append(command, "aggregated")
	} else {
		command = append(command, "standalone")
	}

	args := util.ArgMapsToArgStrings(argsMap, argOverrides)
	command = append(command, args...)

	replicas := int32(1)
	dep := &appsv1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    ComponentLabel,
		},
		Spec: appsv1beta1.DeploymentSpec{
			Replicas: &replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   name,
					Labels: apiserverPodLabels,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "clusterregistry",
							Image:           serverImage,
							ImagePullPolicy: v1.PullAlways,
							Command:         command,
							Ports: []v1.ContainerPort{
								{
									Name:          apiServerSecurePortName,
									ContainerPort: apiServerSecurePort,
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      credentialsName,
									MountPath: "/etc/clusterregistry/apiserver",
									ReadOnly:  true,
								},
							},
						},
						{
							Name:  "etcd",
							Image: etcdImage,
							Command: []string{
								"/usr/local/bin/etcd",
								"--data-dir",
								"/var/etcd/data",
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: credentialsName,
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: credentialsName,
								},
							},
						},
					},
				},
			},
		},
	}

	if pvc != nil {
		dataVolumeName := "etcddata"
		etcdVolume := v1.Volume{
			Name: dataVolumeName,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				},
			},
		}
		etcdVolumeMount := v1.VolumeMount{
			Name:      dataVolumeName,
			MountPath: "/var/etcd",
		}

		dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, etcdVolume)
		for i, container := range dep.Spec.Template.Spec.Containers {
			if container.Name == "etcd" {
				dep.Spec.Template.Spec.Containers[i].VolumeMounts = append(dep.Spec.Template.Spec.Containers[i].VolumeMounts, etcdVolumeMount)
			}
		}
	}

	if len(serviceAccountName) > 0 {
		dep.Spec.Template.Spec.ServiceAccountName = serviceAccountName
	}

	if dryRun {
		return dep, nil
	}

	return clientset.AppsV1beta1().Deployments(namespace).Create(dep)
}

// WaitForLoadBalancerAddress polls the apiserver for load balancer status
// to retrieve IPs and hostnames for it.
func WaitForLoadBalancerAddress(cmdOut io.Writer, clientset client.Interface, svc *v1.Service, dryRun bool) ([]string, []string, error) {
	ips := []string{}
	hostnames := []string{}

	if dryRun {
		return ips, hostnames, nil
	}

	err := wait.PollImmediateInfinite(lbAddrRetryInterval, func() (bool, error) {
		fmt.Fprint(cmdOut, ".")
		pollSvc, err := clientset.CoreV1().Services(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		if ings := pollSvc.Status.LoadBalancer.Ingress; len(ings) > 0 {
			for _, ing := range ings {
				if len(ing.IP) > 0 {
					ips = append(ips, ing.IP)
				}
				if len(ing.Hostname) > 0 {
					hostnames = append(hostnames, ing.Hostname)
				}
			}
			if len(ips) > 0 || len(hostnames) > 0 {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return nil, nil, err
	}

	return ips, hostnames, nil
}

func WaitForPods(cmdOut io.Writer, clientset client.Interface, pods []string, namespace string) error {
	err := wait.PollImmediate(waitInterval, waitTimeout, func() (bool, error) {
		fmt.Fprint(cmdOut, ".")
		podCheck := len(pods)
		podList, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{})
		if err != nil {
			return false, nil
		}
		for _, pod := range podList.Items {
			for _, fedPod := range pods {
				if strings.HasPrefix(pod.Name, fedPod) && pod.Status.Phase == "Running" {
					podCheck -= 1
				}
			}
			// ensure that all pods are in running state or keep waiting
			if podCheck == 0 {
				return true, nil
			}
		}
		return false, nil
	})
	return err
}

func WaitSrvHealthy(cmdOut io.Writer, crClientset client.Interface) error {
	discoveryClient := crClientset.Discovery()
	var innerErr error
	err := wait.PollImmediate(waitInterval, waitTimeout, func() (bool, error) {
		fmt.Fprint(cmdOut, ".")
		body, innerErr := discoveryClient.RESTClient().Get().AbsPath("/healthz").Do().Raw()
		if innerErr != nil {
			return false, nil
		}
		if strings.EqualFold(string(body), "ok") {
			return true, nil
		}
		return false, nil
	})

	if err != nil && innerErr != nil {
		return innerErr
	}
	return err
}

// DeleteKubeconfigEntry handles updating the kubeconfig to remove the cluster
// registry entry that was previously added when the cluster registry was
// initialized.
func DeleteKubeconfigEntry(cmdOut io.Writer, pathOptions *clientcmd.PathOptions,
	name, kubeconfig string, dryRun, ignoreErrors bool) error {

	fmt.Fprintf(cmdOut, "Delete kubeconfig entry %s...", name)
	glog.V(4).Infof("Delete kubeconfig entry %s", name)

	err := util.DeleteKubeconfigEntry(cmdOut, pathOptions, name, kubeconfig,
		dryRun, ignoreErrors)

	if err != nil {
		glog.V(4).Infof("Failed to delete kubeconfig entry %s: %v", name, err)
		return err
	}

	fmt.Fprintln(cmdOut, " done")
	glog.V(4).Info("Successfully deleted kubeconfig entry")
	return nil
}

// WaitForClusterRegistryDeletion waits a certain amount of time for the namespace
// requested to be deleted.
func WaitForClusterRegistryDeletion(cmdOut io.Writer, clientset client.Interface,
	namespace string, dryRun bool) error {

	if dryRun {
		_, err := fmt.Fprintln(cmdOut, "Cluster registry can be deleted (dry run)")
		glog.V(4).Infof("Cluster registry can be deleted (dry run)")
		return err
	}

	fmt.Fprint(cmdOut, "Waiting for the cluster registry to be deleted...")
	glog.V(4).Info("Waiting for the cluster registry to be deleted")

	var getErr error
	err := wait.PollImmediate(waitInterval, waitTimeout, func() (bool, error) {
		fmt.Fprintf(cmdOut, ".")
		_, getErr = clientset.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})

		if getErr != nil {
			statusErr := getErr.(*errors.StatusError)
			if statusErr.ErrStatus.Reason == metav1.StatusReasonNotFound {
				return true, nil
			}
		}

		return false, nil
	})

	if err != nil {
		return getErr
	}

	fmt.Fprintln(cmdOut, " done")
	glog.V(4).Info("Successfully deleted the cluster registry.")
	return nil
}
