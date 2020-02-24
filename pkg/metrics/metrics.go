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

package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	kubefedClusterNotReadyCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubefedcluster_not_ready_total",
			Help: "Number of total not ready states of a kubefed cluster.",
		}, []string{"cluster"}
	)

	kubefedClusterReadyCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubefedcluster_ready_total",
			Help: "Number of total ready states of a kubefed cluster.",
		}, []string{"cluster"}
	)

	kubefedClusterOfflineCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kubefedcluster_offline_total",
			Help: "Number of total offline states of a kubefed cluster.",
		}, []string{"cluster"}
	)

	functionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "function_duration_seconds",
			Help:    "Time taken by various parts of Kubefed main loops.",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.5, 5.0, 7.5, 10.0, 12.5, 15.0, 17.5, 20.0, 22.5, 25.0, 27.5, 30.0, 50.0, 75.0, 100.0, 1000.0},
		}, []string{"function"},
	)

	functionDurationSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:   "function_duration_quantile_seconds",
			Help:   "Quantiles of time taken by various parts of Kubefed main loops.",
			MaxAge: time.Hour,
		}, []string{"function"},
	)
)

// FunctionLabel is a name of Kubefed operation for which
// we measure duration
type FunctionLabel string

const (
	// LogLongDurationThreshold defines the duration after which long function
	// duration will be logged.
	LogLongDurationThreshold = 5 * time.Second
)

// Names of Kubefed operations
const (
	ClusterHealthStatus         FunctionLabel = "clusterHealthStatus"
	ReconcileFederatedResources FunctionLabel = "reconcile:federatedResources"
	ClusterClientConnection     FunctionLabel = "clusterClientConnection"
)

// RegisterAll registers all metrics.
func RegisterAll() {
	metrics.Registry.MustRegister(kubefedClustersNotReadyCount, kubefedClustersReadyCount, kubefedClustersOfflineCount, functionDuration, functionDurationSummary)
}

// UpdateDurationFromStart records the duration of the step identified by the
// label using start time
func UpdateDurationFromStart(label FunctionLabel, start time.Time) {
	duration := time.Since(start)
	UpdateDuration(label, duration)
}

// RegisterKubefedClusterReadyCount records number of Ready states of a Kubefed cluster
func RegisterKubefedClusterReadyCount(clusterName string) {
	kubefedClusterReadyCount.WithLabelValues(clusterName).Inc()
}

// RegisterKubefedClusterOfflineCount records number of Offline states of a Kubefed cluster
func RegisterKubefedClusterOfflineCount(clusterName string) {
	kubefedClusterOfflineCount.WithLabelValues(clusterName).Inc()
}

// RegisterKubefedClusterReadyCount records number of NOT Ready states of a Kubefed cluster
func RegisterKubefedClusterNotReadyCount(clusterName string) {
	kubefedClusterNotReadyCount.WithLabelValues(clusterName).Inc()
}

// UpdateDuration records the duration of the step identified by the label
func UpdateDuration(label FunctionLabel, duration time.Duration) {
	if duration > LogLongDurationThreshold {
		klog.V(4).Infof("Function %s took %v to complete", label, duration)
	}

	functionDurationSummary.WithLabelValues(string(label)).Observe(duration.Seconds())
	functionDuration.WithLabelValues(string(label)).Observe(duration.Seconds())
}
