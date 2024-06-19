package health

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
)

func CheckCondition(nodeInfo *Node) error {
	for _, node := range nodeInfo.Nodes {
		for _, condition := range node.Status.Conditions {
			nodeUsable := true
			switch condition.Type {
			case v1.NodeReady:
				if condition.Status != v1.ConditionTrue {
					nodeUsable = false
				}
			case v1.NodeDiskPressure:
				fallthrough
			case v1.NodeMemoryPressure:
				fallthrough
			case v1.NodeNetworkUnavailable:
				fallthrough
			case v1.NodePIDPressure:
				if condition.Status != v1.ConditionFalse {
					nodeUsable = false
				}
			default:
				return fmt.Errorf("unsupported node condition %s", condition.Type)
			}
			if !nodeUsable {
				return fmt.Errorf("node is not healthy: condition list for %s indicates a problem: %#v",
					node.Name, node.Status.Conditions)
			}
		}
	}

	return nil
}
