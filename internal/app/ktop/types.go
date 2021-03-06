package ktop

import (
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	metrics "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type SimplifiedPodMetrics struct {
	PodName              string
	Namespace            string
	MemoryBytes          uint64
	MemoryBytesRequested uint64
	CPUMillis            uint64 `info:"Measured in Milli CPU's"`
	CPUMillisRequested   uint64
}

type SimplifiedPodMetricsList struct {
	Pods     []*SimplifiedPodMetrics
	PollTime time.Time
}

type KubeSummary struct {
	TotalAllocatableCPUMillis   uint64
	TotalAllocatableMemoryBytes uint64
	TotalUsedCPUMillis          uint64
	TotalUsedMemoryBytes        uint64
	TotalNodes                  int
	ServerInfo                  string
}

func newSimplifiedPodMetrics(pod *metrics.PodMetrics, podResources *SimplifiedPodResources) *SimplifiedPodMetrics {
	// calc total byte mem
	simple := SimplifiedPodMetrics{
		PodName:              pod.GetName(),
		Namespace:            pod.GetNamespace(),
		MemoryBytes:          0,
		MemoryBytesRequested: podResources.MemoryBytesRequested,
		CPUMillis:            0,
		CPUMillisRequested:   podResources.CPUMillisRequested,
	}

	for _, c := range pod.Containers {
		cMem, _ := c.Usage.Memory().AsDec().Unscaled()
		cCPU, _ := c.Usage.Cpu().AsDec().Unscaled()
		simple.CPUMillis = simple.CPUMillis + uint64(cCPU)
		simple.MemoryBytes = simple.MemoryBytes + uint64(cMem)
	}

	return &simple
}

func NewSimplifiedPodMetricsList(list *metrics.PodMetricsList, podResourceList *PodResourcesList) *SimplifiedPodMetricsList {
	simpleList := SimplifiedPodMetricsList{
		Pods:     make([]*SimplifiedPodMetrics, 0),
		PollTime: time.Now(),
	}

	for _, p := range list.Items {
		podResources, err := podResourceList.Find(p.GetName(), p.GetNamespace())
		if err != nil {

		}
		simpleList.Pods = append(simpleList.Pods, newSimplifiedPodMetrics(&p, podResources))
	}

	return &simpleList
}

func (list *SimplifiedPodMetricsList) OrderByHighestMemUsage() {
	sort.Slice(list.Pods, func(i, j int) bool {
		return list.Pods[i].MemoryBytes > list.Pods[j].MemoryBytes
	})
}

func (list *SimplifiedPodMetricsList) OrderByLowestMemUsage() {
	sort.Slice(list.Pods, func(i, j int) bool {
		return list.Pods[i].MemoryBytes < list.Pods[j].MemoryBytes
	})
}

func (list *SimplifiedPodMetricsList) OrderByHighestCPUUsage() {
	sort.Slice(list.Pods, func(i, j int) bool {
		return list.Pods[i].CPUMillis > list.Pods[j].CPUMillis
	})
}

func (list *SimplifiedPodMetricsList) OrderByLowestCPUUsage() {
	sort.Slice(list.Pods, func(i, j int) bool {
		return list.Pods[i].CPUMillis < list.Pods[j].CPUMillis
	})
}

func (m *SimplifiedPodMetrics) CPUMillisString() string {
	return strconv.FormatUint(m.CPUMillis, 10)
}

func (m *SimplifiedPodMetrics) CPUMillisRequestedString() string {
	if m.CPUMillisRequested == 0 {
		return "-"
	} else {
		return strconv.FormatUint(m.CPUMillisRequested, 10)
	}
}

func (m *SimplifiedPodMetrics) MemoryBytesString() string {
	return humanize.Bytes(m.MemoryBytes)
}

func (m *SimplifiedPodMetrics) MemoryBytesRequestedString() string {
	if m.MemoryBytesRequested == 0 {
		return "-"
	} else {
		return humanize.Bytes(m.MemoryBytesRequested)
	}
}

func (s *KubeSummary) GetTotalAllocatableMemory() string {
	return humanize.Bytes(s.TotalAllocatableMemoryBytes)
}

func (s *KubeSummary) GetTotalUsedMemory() string {
	return humanize.Bytes(s.TotalUsedMemoryBytes)
}

func (s *KubeSummary) GetMemPercentUsed() string {
	dec := float64(s.TotalUsedMemoryBytes) / float64(s.TotalAllocatableMemoryBytes) * 100
	return strconv.FormatFloat(math.Ceil(dec), 'f', 0, 64) + "%"
}

func (s *KubeSummary) GetCPUPercentUsed() string {
	dec := float64(s.TotalUsedCPUMillis) / float64(s.TotalAllocatableCPUMillis) * 100
	return strconv.FormatFloat(math.Ceil(dec), 'f', 0, 64) + "%"
}
