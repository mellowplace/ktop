package ktop

import corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
import apicorev1 "k8s.io/api/core/v1"
import meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type SimplifiedPodResources struct {
	PodName              string
	Namespace            string
	MemoryBytesRequested uint64
	CPUMillisRequested   uint64
}

type PodResourcesList struct {
	Pods            []*SimplifiedPodResources
	kubeConfigFile  string
	kubeContextName string
	coreClient      *corev1.CoreV1Client
}

func NewPodResourcesList(kubeConfigFile, kubeContextName string) PodResourcesList {
	return PodResourcesList{
		Pods:            []*SimplifiedPodResources{},
		kubeConfigFile:  kubeConfigFile,
		kubeContextName: kubeContextName,
	}
}

func (l *PodResourcesList) add(resources *SimplifiedPodResources) {
	l.Pods = append(l.Pods, resources)
}

func (l *PodResourcesList) Find(name, namespace string) (*SimplifiedPodResources, error) {
	for _, r := range l.Pods {
		if r.PodName == name && r.Namespace == namespace {
			return r, nil
		}
	}

	// if not cached then we should pull from k8s
	if l.coreClient == nil {
		config, _, err := getRestConfig(l.kubeConfigFile, l.kubeContextName)
		if err != nil {
			return nil, err
		}

		client, err := corev1.NewForConfig(config)
		if err != nil {
			return nil, err
		}

		l.coreClient = client
	}

	pod, err := l.coreClient.Pods(namespace).Get(name, meta_v1.GetOptions{
		ResourceVersion: "0",
	})

	if err != nil {
		return nil, err
	}

	totalCPU := uint64(0)
	totalMem := uint64(0)

	for _, c := range pod.Spec.Containers {
		containerMemQuantity := c.Resources.Requests[apicorev1.ResourceMemory]
		containerCPUQuantity := c.Resources.Requests[apicorev1.ResourceCPU]
		containerMem, _ := (&containerMemQuantity).AsDec().Unscaled()
		containerCPU, _ := (&containerCPUQuantity).AsDec().Unscaled()

		totalMem += uint64(containerMem)
		totalCPU += uint64(containerCPU)
	}

	podResources := &SimplifiedPodResources{
		CPUMillisRequested:   totalCPU,
		MemoryBytesRequested: totalMem,
	}

	l.add(podResources)

	return podResources, nil
}
