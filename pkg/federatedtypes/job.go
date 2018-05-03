/*
Copyright 2018 The Federation v2 Authors.

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

package federatedtypes

import (
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	fedclientset "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
)

const (
	JobKind          = "Job"
	FederatedJobKind = "FederatedJob"
)

var (
	jobNamespaced bool                = true
	JobTypeConfig FederatedTypeConfig = FederatedTypeConfig{
		ComparisonType: util.Generation,
		Template: FederationAPIResource{
			APIResource: apiResource(FederatedJobKind, "federatedjobs", jobNamespaced),
		},
		Placement: FederationAPIResource{
			APIResource: apiResource("FederatedJobPlacement", "federatedjobplacements", jobNamespaced),
		},
		Override: &FederationAPIResource{
			APIResource: apiResource("FederatedJobOverride", "federatedjoboverrides", jobNamespaced),
		},
		Target: metav1.APIResource{
			Name:       "jobs",
			Group:      "batch",
			Kind:       JobKind,
			Version:    "v1",
			Namespaced: jobNamespaced,
		},
		AdapterFactory: NewFederatedJobAdapter,
	}
)

func init() {
	RegisterFederatedTypeConfig(FederatedJobKind, JobTypeConfig)
	RegisterTestObjectsFunc(FederatedJobKind, NewFederatedJobObjectsForTest)
}

type FederatedJobAdapter struct {
	client fedclientset.Interface
}

func NewFederatedJobAdapter(client fedclientset.Interface) FederatedTypeAdapter {
	return &FederatedJobAdapter{client: client}
}

func (a *FederatedJobAdapter) FedClient() fedclientset.Interface {
	return a.client
}

func (a *FederatedJobAdapter) Template() FedApiAdapter {
	return NewFederatedJobTemplate(a.client)
}

func (a *FederatedJobAdapter) Placement() PlacementAdapter {
	return NewFederatedJobPlacement(a.client)
}

func (a *FederatedJobAdapter) PlacementAPIResource() *metav1.APIResource {
	return &JobTypeConfig.Placement.APIResource
}

func (a *FederatedJobAdapter) Override() OverrideAdapter {
	return NewFederatedJobOverride(a.client)
}

func (a *FederatedJobAdapter) Target() TargetAdapter {
	return JobAdapter{}
}

// TODO(marun) Copy the whole thing
func (a *FederatedJobAdapter) ObjectForCluster(template, override pkgruntime.Object, clusterName string) pkgruntime.Object {
	fedJob := template.(*fedv1a1.FederatedJob)
	templateJob := fedJob.Spec.Template

	job := &batchv1.Job{
		ObjectMeta: util.DeepCopyRelevantObjectMeta(templateJob.ObjectMeta),
		Spec:       *templateJob.Spec.DeepCopy(),
	}

	if override != nil {
		jobOverride := override.(*fedv1a1.FederatedJobOverride)
		for _, clusterOverride := range jobOverride.Spec.Overrides {
			if clusterOverride.ClusterName == clusterName {
				job.Spec.Parallelism = clusterOverride.Parallelism
				break
			}
		}
	}

	// Avoid having to duplicate these details in the template or have
	// the name/namespace vary between the federation api and member
	// clusters.
	//
	// TODO(marun) Document this
	job.Name = fedJob.Name
	job.Namespace = fedJob.Namespace

	return job
}

func (a *FederatedJobAdapter) ObjectForUpdateOp(desiredObj, clusterObj pkgruntime.Object) pkgruntime.Object {
	return desiredObj
}

type FederatedJobTemplate struct {
	client fedclientset.Interface
}

func NewFederatedJobTemplate(client fedclientset.Interface) FedApiAdapter {
	return &FederatedJobTemplate{client: client}
}

func (a *FederatedJobTemplate) Kind() string {
	return FederatedJobKind
}

func (a *FederatedJobTemplate) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedJob{}
}

func (a *FederatedJobTemplate) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedJob := obj.(*fedv1a1.FederatedJob)
	return a.client.FederationV1alpha1().FederatedJobs(fedJob.Namespace).Create(fedJob)
}

func (a *FederatedJobTemplate) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedJobs(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedJobTemplate) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedJobs(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedJobTemplate) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedJobs(namespace).List(options)
}

func (a *FederatedJobTemplate) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedJob := obj.(*fedv1a1.FederatedJob)
	updatedObj, err := a.client.FederationV1alpha1().FederatedJobs(fedJob.Namespace).Update(fedJob)
	return updatedObj, err
}

func (a *FederatedJobTemplate) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedJobs(namespace).Watch(options)
}

type FederatedJobPlacement struct {
	client fedclientset.Interface
}

func NewFederatedJobPlacement(client fedclientset.Interface) PlacementAdapter {
	return &FederatedJobPlacement{client: client}
}

func (a *FederatedJobPlacement) Kind() string {
	return "FederatedJobPlacement"
}

func (a *FederatedJobPlacement) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedJobPlacement{}
}

func (a *FederatedJobPlacement) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedJobPlacement := obj.(*fedv1a1.FederatedJobPlacement)
	return a.client.FederationV1alpha1().FederatedJobPlacements(fedJobPlacement.Namespace).Create(fedJobPlacement)
}

func (a *FederatedJobPlacement) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedJobPlacements(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedJobPlacement) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedJobPlacements(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedJobPlacement) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedJobPlacements(namespace).List(options)
}

func (a *FederatedJobPlacement) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedJobPlacement := obj.(*fedv1a1.FederatedJobPlacement)
	return a.client.FederationV1alpha1().FederatedJobPlacements(fedJobPlacement.Namespace).Update(fedJobPlacement)
}

func (a *FederatedJobPlacement) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedJobPlacements(namespace).Watch(options)
}

func (a *FederatedJobPlacement) ClusterNames(obj pkgruntime.Object) []string {
	fedJobPlacement := obj.(*fedv1a1.FederatedJobPlacement)
	clusterNames := []string{}
	for _, name := range fedJobPlacement.Spec.ClusterNames {
		clusterNames = append(clusterNames, name)
	}
	return clusterNames
}

func (a *FederatedJobPlacement) SetClusterNames(obj pkgruntime.Object, clusterNames []string) {
	fedJobPlacement := obj.(*fedv1a1.FederatedJobPlacement)
	fedJobPlacement.Spec.ClusterNames = clusterNames
}

type FederatedJobOverride struct {
	client fedclientset.Interface
}

func NewFederatedJobOverride(client fedclientset.Interface) OverrideAdapter {
	return &FederatedJobOverride{client: client}
}

func (a *FederatedJobOverride) Kind() string {
	return "FederatedJobOverride"
}

func (a *FederatedJobOverride) ObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedJobOverride{}
}

func (a *FederatedJobOverride) Create(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedJobOverride := obj.(*fedv1a1.FederatedJobOverride)
	return a.client.FederationV1alpha1().FederatedJobOverrides(fedJobOverride.Namespace).Create(fedJobOverride)
}

func (a *FederatedJobOverride) Delete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedJobOverrides(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedJobOverride) Get(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedJobOverrides(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedJobOverride) List(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedJobOverrides(namespace).List(options)
}

func (a *FederatedJobOverride) Update(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedJobOverride := obj.(*fedv1a1.FederatedJobOverride)
	return a.client.FederationV1alpha1().FederatedJobOverrides(fedJobOverride.Namespace).Update(fedJobOverride)
}

func (a *FederatedJobOverride) Watch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedJobOverrides(namespace).Watch(options)
}

type JobAdapter struct {
}

func (JobAdapter) Kind() string {
	return JobKind
}

func (JobAdapter) ObjectType() pkgruntime.Object {
	return &batchv1.Job{}
}

func (JobAdapter) VersionCompareType() util.VersionCompareType {
	return JobTypeConfig.ComparisonType
}

func (JobAdapter) Create(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	job := obj.(*batchv1.Job)
	createdObj, err := client.BatchV1().Jobs(job.Namespace).Create(job)
	return createdObj, err
}

func (JobAdapter) Delete(client kubeclientset.Interface, qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return client.BatchV1().Jobs(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (JobAdapter) Get(client kubeclientset.Interface, qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return client.BatchV1().Jobs(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (JobAdapter) List(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return client.BatchV1().Jobs(namespace).List(options)
}

func (JobAdapter) Update(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	job := obj.(*batchv1.Job)
	return client.BatchV1().Jobs(job.Namespace).Update(job)
}
func (JobAdapter) Watch(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return client.BatchV1().Jobs(namespace).Watch(options)
}

func NewFederatedJobObjectsForTest(namespace string, clusterNames []string) (template, placement, override pkgruntime.Object) {
	labels := map[string]string{"foo": "bar"}
	zero := int64(0)
	one := int32(1)
	manualSeelctorValue := true
	// TODO(marun) A job created in a member cluster will have
	// some fields set to defaults if no value is provided for a given
	// field.  Unless a federated resource has all such fields
	// populated, a reconcile loop may result.  A loop would be
	// characterized by one or more fields being populated in the
	// member cluster resource but not in the federated resource,
	// resulting in endless attempts to update the member resource.
	// Possible workarounds include:
	//
	//   - performing the same defaulting in the fed api
	//   - avoid comparison of fields that are not populated
	//
	// As a temporary workaround, ensure all defaulted fields are
	// populated and mark them with comments.
	template = &fedv1a1.FederatedJob{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-job-",
			Namespace:    namespace,
		},
		Spec: fedv1a1.FederatedJobSpec{
			Template: batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: batchv1.JobSpec{
					Parallelism: &one, // defaulted by APIserver
					Selector: &metav1.LabelSelector{
						MatchLabels: labels,
					},
					ManualSelector: &manualSeelctorValue,
					Template: apiv1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: labels,
						},
						Spec: apiv1.PodSpec{
							RestartPolicy:                 apiv1.RestartPolicyNever, // forced
							TerminationGracePeriodSeconds: &zero,
							Containers: []apiv1.Container{
								{
									Name:  "nginx",
									Image: "nginx",
								},
							},
						},
					},
				},
			},
		},
	}
	placement = &fedv1a1.FederatedJobPlacement{
		ObjectMeta: metav1.ObjectMeta{
			// Name will be set to match the template by the crud tester
			Namespace: namespace,
		},
		Spec: fedv1a1.FederatedJobPlacementSpec{
			ClusterNames: clusterNames,
		},
	}

	two := int32(2)
	clusterName := clusterNames[0]
	override = &fedv1a1.FederatedJobOverride{
		ObjectMeta: metav1.ObjectMeta{
			// Name will be set to match the template by the crud tester
			Namespace: namespace,
		},
		Spec: fedv1a1.FederatedJobOverrideSpec{
			Overrides: []fedv1a1.FederatedJobClusterOverride{
				{
					ClusterName: clusterName,
					Parallelism: &two,
				},
			},
		},
	}
	return template, placement, override
}
