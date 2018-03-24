# Steps to get metrics working on Minikube

1. `minikube addons enable heapster` (not sure if this is essential)
2. Clone https://github.com/kubernetes-incubator/metrics-server and run
   `kubectl create -f deploy/1.8+/`
3. Wait for a bit and your stats will appear
