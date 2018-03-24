package ktop

import (
	"sort"
	"strconv"

	metrics "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type SimplifiedPodMetrics struct {
	PodName     string
	Namespace   string
	MemoryBytes uint64
	CPUMillis   uint64 `info:"Measured in Milli CPU's"`
}

type SimplifiedPodMetricsList struct {
	Pods []*SimplifiedPodMetrics
}

func newSimplifiedPodMetrics(pod *metrics.PodMetrics) *SimplifiedPodMetrics {
	// calc total byte mem
	simple := SimplifiedPodMetrics{
		PodName:     pod.GetName(),
		Namespace:   pod.GetNamespace(),
		MemoryBytes: 0,
		CPUMillis:   0,
	}

	for _, c := range pod.Containers {
		cMem, _ := c.Usage.Memory().AsDec().Unscaled()
		cCPU, _ := c.Usage.Cpu().AsDec().Unscaled()
		simple.CPUMillis = simple.CPUMillis + uint64(cCPU)
		simple.MemoryBytes = simple.MemoryBytes + uint64(cMem)
	}

	return &simple
}

func NewSimplifiedPodMetricsList(list *metrics.PodMetricsList) *SimplifiedPodMetricsList {
	simpleList := SimplifiedPodMetricsList{
		Pods: make([]*SimplifiedPodMetrics, 0),
	}

	for _, p := range list.Items {
		simpleList.Pods = append(simpleList.Pods, newSimplifiedPodMetrics(&p))
	}

	return &simpleList
}

func (list *SimplifiedPodMetricsList) OrderByHighestMemUsage() {
	sort.Slice(list.Pods, func(i, j int) bool {
		return list.Pods[i].MemoryBytes > list.Pods[j].MemoryBytes
	})
}

func (m *SimplifiedPodMetrics) CPUMillisString() string {
	return strconv.FormatUint(m.CPUMillis, 10)
}

func (m *SimplifiedPodMetrics) MemoryBytesString() string {
	return strconv.FormatUint(m.MemoryBytes, 10)
}
