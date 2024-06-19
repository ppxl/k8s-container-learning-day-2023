package health

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
)

// Node provides data about a k3d cluster node.
type Node struct {
	Nodes []v1.Node
}

// String returns a printable list of all nodes and their namespace.
func (info *Node) String() string {
	sb := strings.Builder{}
	for i, node := range info.Nodes {
		sb.WriteString("node: ")
		sb.WriteString(node.Name)
		if i < len(info.Nodes)-1 {
			sb.WriteString(", ")
		}
	}

	return sb.String()
}

// FetchNodeInfo returns information about current nodes.
func FetchNodeInfo(ctx context.Context, clientset kubernetes.Interface) (info *Node, err error) {
	if clientset == nil {
		return info, fmt.Errorf("clientset must not be nil")
	}
	if err != nil {
		return info, fmt.Errorf("could not retrieve node info: %w", err)
	}

	list, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return info, fmt.Errorf("could not list nodes: %w", err)
	}

	if len(list.Items) == 0 {
		return info, fmt.Errorf("could not return node info because no node was found (was it killed in the meantime?)")
	}

	return &Node{list.Items}, nil
}

// KubeletEvictionFsByPercentage takes a percentage integer from 0 to 100 and
// returns a matching kubelet hard-eviction argument. A percentage of 10 means
// that with 10 % free space, the cluster will start to taint the nodes,
// effectively blocking workloads to be scheduled.
func KubeletEvictionFsByPercentage(percentage int) (string, error) {
	if percentage < 0 || percentage > 100 {
		return "", fmt.Errorf("percentage must be in rage of 0 and 100")
	}
	return fmt.Sprintf("imagefs.available<%d%%,nodefs.available<%d%%", percentage, percentage), nil
}
