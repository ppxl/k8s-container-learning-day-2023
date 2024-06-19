package cluster

import (
	"k8s.io/client-go/kubernetes"
	"testing"
)

// Lookout provides convenience functionalities for cluster resources.
type Lookout struct {
	t *testing.T
	c kubernetes.Interface
}

// Pods returns a PodListSelector to address multiple pods.
func (l *Lookout) Pods(namespace string) *PodListSelector {
	return &PodListSelector{
		podClient: l.c.CoreV1().Pods(namespace),
	}
}

// Pod returns a single PodSelector to address a single pod.
func (l *Lookout) Pod(namespace, name string) *PodSelector {
	return &PodSelector{
		podClient:   l.c.CoreV1().Pods(namespace),
		eventClient: l.c.CoreV1().Events(namespace),
		name:        name,
	}
}
