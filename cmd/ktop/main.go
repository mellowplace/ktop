package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mellowplace/ktop/internal/app/ktop"
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

	err := ktop.StartUI(*kubeConfig, *kubeContext, "kube-system")

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
