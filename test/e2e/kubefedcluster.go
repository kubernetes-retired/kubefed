package e2e

import (
	"fmt"

	"k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"

	"sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/test/e2e/framework"

	. "github.com/onsi/ginkgo" //nolint:stylecheck
	. "github.com/onsi/gomega" //nolint:stylecheck
)

var _ = Describe("KubeFedCluster", func() {
	f := framework.NewKubeFedFramework("kubefedcluster")

	tl := framework.NewE2ELogger()

	It("should correctly report the Kubernetes git version of the cluster", func() {
		userAgent := "test-kubefedcluster-kubernetes-version"
		client := f.Client(userAgent)
		clusterList := framework.ListKubeFedClusters(tl, client, framework.TestContext.KubeFedSystemNamespace)

		for _, cluster := range clusterList.Items {
			config, err := util.BuildClusterConfig(&cluster, client, framework.TestContext.KubeFedSystemNamespace)
			Expect(err).NotTo(HaveOccurred())
			restclient.AddUserAgent(config, userAgent)

			client, err := discovery.NewDiscoveryClientForConfig(config)
			if err != nil {
				tl.Fatalf("Error creating discovery client for cluster %q", cluster.Name)
			}
			version, err := client.ServerVersion()
			if err != nil {
				tl.Fatalf("Error retrieving Kubernetes version of cluster %q", cluster.Name)
			}
			Expect(cluster.Status.KubernetesVersion).To(Equal(version.GitVersion), fmt.Sprintf(
				"the KubernetesVersion field of KubeFedCluster %q should be equal to the Kubernetes git version of the cluster", cluster.Name))
		}
	})
})
