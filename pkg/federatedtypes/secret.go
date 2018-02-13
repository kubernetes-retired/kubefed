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

package federatedtypes

import (
	fedv1a1 "github.com/marun/fnord/pkg/apis/federation/v1alpha1"
	fedclientset "github.com/marun/fnord/pkg/client/clientset_generated/clientset"
	"github.com/marun/fnord/pkg/controller/util"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	kubeclientset "k8s.io/client-go/kubernetes"
)

const (
	SecretKind          = "Secret"
	FederatedSecretKind = "FederatedSecret"
)

func init() {
	RegisterFederatedType(FederatedSecretKind, NewFederatedSecretAdapter)
	RegisterTestObjectFunc(FederatedSecretKind, NewFederatedSecretForTest)
}

type FederatedSecretAdapter struct {
	client fedclientset.Interface
}

func NewFederatedSecretAdapter(client fedclientset.Interface) FederatedTypeAdapter {
	return &FederatedSecretAdapter{client: client}
}

func (a *FederatedSecretAdapter) FedKind() string {
	return FederatedSecretKind
}

func (a *FederatedSecretAdapter) FedObjectMeta(obj pkgruntime.Object) *metav1.ObjectMeta {
	return &obj.(*fedv1a1.FederatedSecret).ObjectMeta
}

func (a *FederatedSecretAdapter) FedObjectType() pkgruntime.Object {
	return &fedv1a1.FederatedSecret{}
}

func (a *FederatedSecretAdapter) ObjectForCluster(obj pkgruntime.Object, clusterName string) pkgruntime.Object {
	// TODO(marun) support per-cluster overrides
	fedSecret := obj.(*fedv1a1.FederatedSecret)
	templateSecret := fedSecret.Spec.Template
	secret := &apiv1.Secret{
		ObjectMeta: util.DeepCopyRelevantObjectMeta(templateSecret.ObjectMeta),
		Data:       templateSecret.Data,
		Type:       templateSecret.Type,
	}

	// Avoid having to duplicate these details in the template
	secret.Name = fedSecret.Name
	secret.Namespace = fedSecret.Namespace

	return secret
}

func (a *FederatedSecretAdapter) FedCreate(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedSecret := obj.(*fedv1a1.FederatedSecret)
	return a.client.FederationV1alpha1().FederatedSecrets(fedSecret.Namespace).Create(fedSecret)
}

func (a *FederatedSecretAdapter) FedDelete(qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return a.client.FederationV1alpha1().FederatedSecrets(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedSecretAdapter) FedGet(qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedSecrets(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedSecretAdapter) FedList(namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return a.client.FederationV1alpha1().FederatedSecrets(namespace).List(options)
}

func (a *FederatedSecretAdapter) FedUpdate(obj pkgruntime.Object) (pkgruntime.Object, error) {
	fedSecret := obj.(*fedv1a1.FederatedSecret)
	return a.client.FederationV1alpha1().FederatedSecrets(fedSecret.Namespace).Update(fedSecret)
}

func (a *FederatedSecretAdapter) FedWatch(namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return a.client.FederationV1alpha1().FederatedSecrets(namespace).Watch(options)
}

func (a *FederatedSecretAdapter) Kind() string {
	return SecretKind
}

func (a *FederatedSecretAdapter) ObjectMeta(obj pkgruntime.Object) *metav1.ObjectMeta {
	return &obj.(*apiv1.Secret).ObjectMeta
}

func (a *FederatedSecretAdapter) ObjectType() pkgruntime.Object {
	return &corev1.Secret{}
}

func (a *FederatedSecretAdapter) Equivalent(obj1, obj2 pkgruntime.Object) bool {
	secret1 := obj1.(*corev1.Secret)
	secret2 := obj2.(*corev1.Secret)
	return util.SecretEquivalent(*secret1, *secret2)
}

func (a *FederatedSecretAdapter) Create(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	secret := obj.(*corev1.Secret)
	return client.CoreV1().Secrets(secret.Namespace).Create(secret)
}

func (a *FederatedSecretAdapter) Delete(client kubeclientset.Interface, qualifiedName QualifiedName, options *metav1.DeleteOptions) error {
	return client.CoreV1().Secrets(qualifiedName.Namespace).Delete(qualifiedName.Name, options)
}

func (a *FederatedSecretAdapter) Get(client kubeclientset.Interface, qualifiedName QualifiedName) (pkgruntime.Object, error) {
	return client.CoreV1().Secrets(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
}

func (a *FederatedSecretAdapter) List(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (pkgruntime.Object, error) {
	return client.CoreV1().Secrets(namespace).List(options)
}

func (a *FederatedSecretAdapter) Update(client kubeclientset.Interface, obj pkgruntime.Object) (pkgruntime.Object, error) {
	secret := obj.(*corev1.Secret)
	return client.CoreV1().Secrets(secret.Namespace).Update(secret)
}

func (a *FederatedSecretAdapter) Watch(client kubeclientset.Interface, namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return client.CoreV1().Secrets(namespace).Watch(options)
}

func NewFederatedSecretForTest(namespace string) pkgruntime.Object {
	return &fedv1a1.FederatedSecret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-secret-",
			Namespace:    namespace,
		},
		Spec: fedv1a1.FederatedSecretSpec{
			Template: corev1.Secret{
				Data: map[string][]byte{
					"A": []byte("ala ma kota"),
				},
				Type: corev1.SecretTypeOpaque,
			},
		},
	}
}
