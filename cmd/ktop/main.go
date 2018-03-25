package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/mellowplace/ktop/internal/app/ktop"
)

var (
	kubeContext string
	namespace   string
	kubeConfig  string
)

func init() {
	flag.StringVar(&kubeContext, "context", "", "kubectl context name, empty will use the current")
	flag.StringVar(&namespace, "namespace", "", "which namespace to grab Pod metrics from")
	if home := homeDir(); home != "" {
		flag.StringVar(&kubeConfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeConfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
}

func main() {

	flag.Parse()

	// use the current context in kubeconfig

	err := ktop.StartUI(kubeConfig, kubeContext, namespace)

	if err != nil {
		log.Fatalf("%v", err)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
