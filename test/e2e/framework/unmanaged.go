/*
Copyright 2018 The Kubernetes Authors.

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

package framework

import (
	"fmt"
	"sort"
	"time"

	fedcommon "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/common"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/federatedtypes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeclientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	crclientset "k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// Only check that the api is available once
var checkedApi bool

type UnmanagedFramework struct {
	// To make sure that this framework cleans up after itself, no matter what,
	// we install a Cleanup action before each test and clear it after.  If we
	// should abort, the AfterSuite hook should run all Cleanup actions.
	cleanupHandle CleanupActionHandle

	testNamespaceName string

	Config *restclient.Config

	BaseName string
}

func NewUnmanagedFramework(baseName string) FederationFramework {
	f := &UnmanagedFramework{
		BaseName: baseName,
	}
	return f
}

// BeforeEach checks for federation apiserver is ready and makes a namespace.
func (f *UnmanagedFramework) BeforeEach() {
	// The fact that we need this feels like a bug in ginkgo.
	// https://github.com/onsi/ginkgo/issues/222
	f.cleanupHandle = AddCleanupAction(f.AfterEach)

	if f.Config == nil {
		By("Reading cluster configuration")
		if TestContext.KubeConfig == "" {
			Failf("--kubeconfig or KUBECONFIG must be specified to load client config")
		}
		var err error
		f.Config, err = loadConfig(TestContext.KubeConfig, TestContext.KubeContext)
		Expect(err).NotTo(HaveOccurred())
	}

	if !checkedApi {
		// Check the health of the target federation
		By("Waiting for apiserver to be ready")
		client := f.FedClient(fmt.Sprintf("%s-setup", f.BaseName))
		err := waitForApiserver(client)
		Expect(err).NotTo(HaveOccurred())
		By("apiserver is ready")
		checkedApi = true
	}
}

// AfterEach deletes the namespace, after reading its events.
func (f *UnmanagedFramework) AfterEach() {
	RemoveCleanupAction(f.cleanupHandle)

	userAgent := fmt.Sprintf("%s-teardown", f.BaseName)

	// DeleteNamespace at the very end in defer, to avoid any
	// expectation failures preventing deleting the namespace.
	defer func() {
		if f.testNamespaceName == "" {
			return
		}
		// Clear the name first to ensure other tests always get a
		// fresh namespace even if namespace deletion fails
		namespaceName := f.testNamespaceName
		f.testNamespaceName = ""

		client := f.KubeClient(userAgent)
		deleteNamespace(client, namespaceName)
	}()

	// Print events if the test failed and ran in a namep.
	if CurrentGinkgoTestDescription().Failed && f.testNamespaceName != "" {
		kubeClient := f.KubeClient(userAgent)
		DumpEventsInNamespace(func(opts metav1.ListOptions, ns string) (*corev1.EventList, error) {
			return kubeClient.Core().Events(ns).List(opts)
		}, f.testNamespaceName)
	}
}

func (f *UnmanagedFramework) FedClient(userAgent string) fedclientset.Interface {
	restclient.AddUserAgent(f.Config, userAgent)
	return fedclientset.NewForConfigOrDie(f.Config)
}

func (f *UnmanagedFramework) KubeClient(userAgent string) kubeclientset.Interface {
	restclient.AddUserAgent(f.Config, userAgent)
	return kubeclientset.NewForConfigOrDie(f.Config)
}

func (f *UnmanagedFramework) CrClient(userAgent string) crclientset.Interface {
	restclient.AddUserAgent(f.Config, userAgent)
	return crclientset.NewForConfigOrDie(f.Config)
}

func (f *UnmanagedFramework) ClusterClients(userAgent string) map[string]kubeclientset.Interface {
	// TODO(marun) Avoid having to reload configuration on every call.
	// Clusters may be added or removed between calls, but
	// configuration is unlikely to change.
	//
	// Could also accept 'forceReload' parameter for tests that require it.

	By("Obtaining a list of federated clusters")
	fedClient := f.FedClient(userAgent)
	clusterList, err := fedClient.FederationV1alpha1().FederatedClusters().List(metav1.ListOptions{})
	ExpectNoError(err, fmt.Sprintf("Error retrieving list of federated clusters: %+v", err))

	if len(clusterList.Items) == 0 {
		Failf("No registered clusters found")
	}

	kubeClient := f.KubeClient(userAgent)
	crClient := f.CrClient(userAgent)
	clusterClients := make(map[string]kubeclientset.Interface)
	for _, cluster := range clusterList.Items {
		ClusterIsReadyOrFail(fedClient, cluster)
		config, err := util.BuildClusterConfig(&cluster, kubeClient, crClient)
		Expect(err).NotTo(HaveOccurred())
		restclient.AddUserAgent(config, userAgent)
		clusterClients[cluster.Name] = kubeclientset.NewForConfigOrDie(config)
	}
	return clusterClients
}

func (f *UnmanagedFramework) TestNamespaceName() string {
	if f.testNamespaceName == "" {
		By("Creating a namespace to execute the test in")
		client := f.KubeClient(fmt.Sprintf("%s-create-namespace", f.BaseName))
		namespaceName, err := createNamespace(client, f.BaseName)
		Expect(err).NotTo(HaveOccurred())
		f.testNamespaceName = namespaceName
		By(fmt.Sprintf("Created test namespace %s", namespaceName))
	}
	return f.testNamespaceName
}

func (f *UnmanagedFramework) SetUpControllerFixture(kind string, adapterFactory federatedtypes.AdapterFactory) {
	// Not supported
}

func createNamespace(client kubeclientset.Interface, baseName string) (string, error) {
	namespaceObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("e2e-tests-%v-", baseName),
		},
	}
	// Be robust about making the namespace creation call.
	// TODO(marun) should all api calls be made 'robustly'?
	var namespaceName string
	if err := wait.PollImmediate(PollInterval, SingleCallTimeout, func() (bool, error) {
		namespace, err := client.Core().Namespaces().Create(namespaceObj)
		if err != nil {
			Logf("Unexpected error while creating namespace: %v", err)
			return false, nil
		}
		namespaceName = namespace.Name
		return true, nil
	}); err != nil {
		return "", err
	}
	return namespaceName, nil
}

func deleteNamespace(client kubeclientset.Interface, namespaceName string) {
	orphanDependents := false
	if err := client.Core().Namespaces().Delete(namespaceName, &metav1.DeleteOptions{OrphanDependents: &orphanDependents}); err != nil {
		Failf("Error while deleting namespace %s: %s", namespaceName, err)
	}
	// TODO(marun) Deletion handling of namespaces in fedv1 relied on
	// a strict separation between a federated namespace and the
	// namespace in a federated cluster.  In fedv2 this distinction
	// has been lost where the host cluster is also a member cluster.
	// Deletion of a namespace cannot strictly depend on namespaces in
	// nested clusters having been removed.  It will be necessary to
	// identify that a given namespace is in the hosting cluster and
	// therefore does not have to be deleted before finalizer removal.
	err := wait.PollImmediate(PollInterval, SingleCallTimeout, func() (bool, error) {
		if _, err := client.Core().Namespaces().Get(namespaceName, metav1.GetOptions{}); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			Logf("Error while waiting for namespace to be removed: %v", err)
			return false, nil
		}
		return false, nil
	})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			Failf("Couldn't delete ns %q: %s", namespaceName, err)
		} else {
			Logf("Namespace %v was already deleted", namespaceName)
		}
	}
}

func loadConfig(configPath, context string) (*restclient.Config, error) {
	Logf(">>> kubeConfig: %s", configPath)
	c, err := clientcmd.LoadFromFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error loading kubeConfig %s: %v", configPath, err.Error())
	}
	if context != "" {
		Logf(">>> kubeContext: %s", context)
		c.CurrentContext = context
	}
	cfg, err := clientcmd.NewDefaultClientConfig(*c, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("error creating default client config: %v", err.Error())
	}
	return cfg, nil
}

// waitForApiserver waits for the apiserver to be ready.  It tests the
// readiness by listing a federation resource and expecting a response
// without error.
func waitForApiserver(client fedclientset.Interface) error {
	return wait.PollImmediate(time.Second, 1*time.Minute, func() (bool, error) {
		_, err := client.FederationV1alpha1().FederatedClusters().List(metav1.ListOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

// byFirstTimestamp sorts a slice of events by first timestamp, using their involvedObject's name as a tie breaker.
type byFirstTimestamp []corev1.Event

func (o byFirstTimestamp) Len() int      { return len(o) }
func (o byFirstTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

func (o byFirstTimestamp) Less(i, j int) bool {
	if o[i].FirstTimestamp.Equal(&o[j].FirstTimestamp) {
		return o[i].InvolvedObject.Name < o[j].InvolvedObject.Name
	}
	return o[i].FirstTimestamp.Before(&o[j].FirstTimestamp)
}

type EventsLister func(opts metav1.ListOptions, ns string) (*corev1.EventList, error)

func DumpEventsInNamespace(eventsLister EventsLister, namespace string) {
	By(fmt.Sprintf("Collecting events from namespace %q.", namespace))
	events, err := eventsLister(metav1.ListOptions{}, namespace)
	Expect(err).NotTo(HaveOccurred())

	By(fmt.Sprintf("Found %d events.", len(events.Items)))
	// Sort events by their first timestamp
	sortedEvents := events.Items
	if len(sortedEvents) > 1 {
		sort.Sort(byFirstTimestamp(sortedEvents))
	}
	for _, e := range sortedEvents {
		Logf("At %v - event for %v: %v %v: %v", e.FirstTimestamp, e.InvolvedObject.Name, e.Source, e.Reason, e.Message)
	}
	// Note that we don't wait for any Cleanup to propagate, which means
	// that if you delete a bunch of pods right before ending your test,
	// you may or may not see the killing/deletion/Cleanup events.
}

// ClusterIsReadyOrFail checks whether the named cluster has been
// marked as ready by the federated cluster controller.  The cluster
// controller records the results of health checks on member clusters
// in the status of federated clusters.
func ClusterIsReadyOrFail(client fedclientset.Interface, cluster fedv1a1.FederatedCluster) {
	clusterName := cluster.Name
	By(fmt.Sprintf("Checking readiness of cluster %q", clusterName))
	err := wait.PollImmediate(PollInterval, SingleCallTimeout, func() (bool, error) {
		for _, condition := range cluster.Status.Conditions {
			if condition.Type == fedcommon.ClusterReady && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
		_, err := client.FederationV1alpha1().FederatedClusters().Get(clusterName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return false, nil
	})
	ExpectNoError(err, fmt.Sprintf("Unexpected error in verifying if cluster %q is ready: %+v", clusterName, err))
	Logf("Cluster %s is Ready", clusterName)
}
