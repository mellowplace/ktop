package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	metricsclientset "k8s.io/metrics/pkg/client/clientset_generated/clientset"
)

func main() {
	kubeContext := flag.String("context", "", "kubectl context name, empty will use the current")
	var kubeConfig *string
	if home := homeDir(); home != "" {
		kubeConfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeConfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.Parse()

	fmt.Println(*kubeContext)

	// use the current context in kubeconfig

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: *kubeConfig},
		&clientcmd.ConfigOverrides{
			ClusterInfo:    clientcmdapi.Cluster{Server: ""},
			CurrentContext: *kubeContext,
		}).ClientConfig()

	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	metricsClient, err := metricsclientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	metrics, err := metricsClient.MetricsV1beta1().PodMetricses("kube-system").List(v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	spew.Dump(metrics)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
