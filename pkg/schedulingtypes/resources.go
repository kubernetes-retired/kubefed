package schedulingtypes

import (
	"reflect"
	"strings"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
)

var (
	FederatedDeployment = GetResourceKind(&fedv1a1.FederatedDeployment{})
	Deployment          = GetResourceKind(&appsv1.Deployment{})
	FederatedReplicaSet = GetResourceKind(&fedv1a1.FederatedReplicaSet{})
	ReplicaSet          = GetResourceKind(&appsv1.ReplicaSet{})
	Pod                 = GetResourceKind(&corev1.Pod{})
)

var PodResource = &metav1.APIResource{
	Name:       GetPluralName(Pod),
	Group:      corev1.SchemeGroupVersion.Group,
	Version:    corev1.SchemeGroupVersion.Version,
	Kind:       Pod,
	Namespaced: true,
}

var ReplicaSechedulingResources = map[string]metav1.APIResource{
	FederatedDeployment: {
		Name:       GetPluralName(Deployment),
		Group:      appsv1.SchemeGroupVersion.Group,
		Version:    appsv1.SchemeGroupVersion.Version,
		Kind:       Deployment,
		Namespaced: true,
	},
	FederatedReplicaSet: {
		Name:       GetPluralName(ReplicaSet),
		Group:      appsv1.SchemeGroupVersion.Group,
		Version:    appsv1.SchemeGroupVersion.Version,
		Kind:       ReplicaSet,
		Namespaced: true,
	},
}

func GetResourceKind(obj pkgruntime.Object) string {
	t := reflect.TypeOf(obj)
	if t.Kind() != reflect.Ptr {
		panic("All types must be pointers to structs.")
	}

	t = t.Elem()
	return t.Name()
}

func GetPluralName(name string) string {
	return strings.ToLower(name) + "s"
}
