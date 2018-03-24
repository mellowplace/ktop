package main

import (
	"flag"
	"fmt"
)

func main() {
	kubeContext := flag.String("context", "", "kubectl context name, empty will use the current")

	flag.Parse()

	fmt.Println(*kubeContext)
}
