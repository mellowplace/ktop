package ktop

import (
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
        _ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	metricsclientset "k8s.io/metrics/pkg/client/clientset_generated/clientset"
)

func pollPodMetrics(kubeConfigFile, kubeContextName, namespace string, podResourceList *PodResourcesList, metricsClose chan int, metricsReceive chan *SimplifiedPodMetricsList, errorChan chan error) {

	defer close(metricsReceive)
	defer close(errorChan)

	config, _, err := getRestConfig(kubeConfigFile, kubeContextName)
	if err != nil {
		errorChan <- fmt.Errorf("Failed to setup kube connection: %v", err)
		return
	}

	// create the clientset
	clientset, err := metricsclientset.NewForConfig(config)
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
				metricsReceive <- NewSimplifiedPodMetricsList(list, podResourceList)
			}
		}
	}
}

func pollKubeSummary(kubeConfigFile, kubeContextName string, summaryReceive chan *KubeSummary, closeChan chan int, errorChan chan error) {
	config, currentContext, err := getRestConfig(kubeConfigFile, kubeContextName)
	if err != nil {
		errorChan <- fmt.Errorf("Failed to setup kube connection: %v", err)
		return
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		errorChan <- fmt.Errorf("Failed to setup core API connection: %v", err)
		return
	}

	nodes, err := clientset.CoreV1().Nodes().List(v1.ListOptions{})
	if err != nil {
		errorChan <- fmt.Errorf("Failed to list nodes: %v", err)
		return
	}

	totalCPU := uint64(0)
	totalMem := uint64(0)
	for _, n := range nodes.Items {
		cpu, _ := n.Status.Allocatable.Cpu().AsDec().Unscaled()
		mem, _ := n.Status.Allocatable.Memory().AsDec().Unscaled()

		totalCPU += uint64(cpu)
		totalMem += uint64(mem)
	}

	serverInfo, _ := clientset.ServerVersion()
	summary := KubeSummary{
		TotalAllocatableCPUMillis:   totalCPU * 1000,
		TotalAllocatableMemoryBytes: totalMem,
		ServerInfo:                  currentContext + " " + serverInfo.String(),
		TotalNodes:                  len(nodes.Items),
	}

	// send the initial totals
	summaryReceive <- &summary

	// create the clientset
	metricsclientset, err := metricsclientset.NewForConfig(config)
	if err != nil {
		errorChan <- fmt.Errorf("Failed to get a metrics connection to kubernetes: %v", err)
		return
	}

	for {
		select {
		case <-closeChan:
			return
		default:
			nodeList, err := metricsclientset.MetricsV1beta1().NodeMetricses().List(v1.ListOptions{})
			if err != nil {
				errorChan <- fmt.Errorf("Could not get node stats: %v", err)
				return
			}

			usedCPU := uint64(0)
			usedMem := uint64(0)
			for _, n := range nodeList.Items {
				cpu, _ := n.Usage.Cpu().AsDec().Unscaled()
				mem, _ := n.Usage.Memory().AsDec().Unscaled()

				usedCPU += uint64(cpu)
				usedMem += uint64(mem)
			}

			summary.TotalUsedCPUMillis = usedCPU
			summary.TotalUsedMemoryBytes = usedMem

			summaryReceive <- &summary
		}

	}
}

func getRestConfig(kubeConfigFile, kubeContextName string) (*rest.Config, string, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigFile},
		&clientcmd.ConfigOverrides{
			ClusterInfo:    clientcmdapi.Cluster{Server: ""},
			CurrentContext: kubeContextName,
		})

	restConfig, err := config.ClientConfig()

	if err != nil {
		return nil, "", err
	}

	apiConfig, _ := config.ConfigAccess().GetStartingConfig()

	return restConfig, apiConfig.CurrentContext, nil
}
