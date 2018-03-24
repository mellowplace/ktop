package ktop

import (
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	metrics "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset_generated/clientset"
)

func pollPodMetrics(kubeConfigFile, kubeContextName, namespace string, metricsClose chan int, metricsReceive chan *metrics.PodMetricsList, errorChan chan error) {

	defer close(metricsReceive)
	defer close(errorChan)

	clientset, err := getMetricsClientSet(kubeConfigFile, kubeContextName)
	if err != nil {
		errorChan <- fmt.Errorf("Failed to get a metrics connection to kubernetes: %v", err)
		return
	}

	for {

		select {
		case <-metricsClose:
			return
		default:
			list, err := clientset.MetricsV1beta1().PodMetricses(namespace).List(v1.ListOptions{})
			if err != nil {
				errorChan <- err
			} else {
				metricsReceive <- list
			}
		}
	}
}

func getMetricsClientSet(kubeConfigFile, kubeContextName string) (*metricsclientset.Clientset, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigFile},
		&clientcmd.ConfigOverrides{
			ClusterInfo:    clientcmdapi.Cluster{Server: ""},
			CurrentContext: kubeContextName,
		}).ClientConfig()

	if err != nil {
		return nil, err
	}

	// create the clientset
	metricsClient, err := metricsclientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return metricsClient, nil
}
