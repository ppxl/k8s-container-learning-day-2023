package health

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

const (
	testNodeName = "k3d-tc-123456-server0"
)

func TestCheckCondition(t *testing.T) {
	type args struct {
		nodeInfo *Node
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "has no conditions", args: args{nodeInfo: &Node{}}, wantErr: false},
		{name: "has ready condition",
			args: args{nodeInfo: &Node{Nodes: []v1.Node{{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status:     v1.NodeStatus{Conditions: []v1.NodeCondition{{Type: v1.NodeReady, Status: v1.ConditionTrue}}},
			}}}},
			wantErr: false},
		{name: "has ready condition (complex)",
			args: args{nodeInfo: &Node{Nodes: []v1.Node{{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status: v1.NodeStatus{Conditions: []v1.NodeCondition{
					{Type: v1.NodeReady, Status: v1.ConditionTrue},
					{Type: v1.NodeDiskPressure, Status: v1.ConditionFalse},
				}}}}}},
			wantErr: false},
		{name: "has unready condition",
			args: args{nodeInfo: &Node{Nodes: []v1.Node{{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status:     v1.NodeStatus{Conditions: []v1.NodeCondition{{Type: v1.NodeReady, Status: v1.ConditionFalse}}},
			}}}},
			wantErr: true},
		{name: "has disk pressure condition",
			args: args{nodeInfo: &Node{Nodes: []v1.Node{{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status:     v1.NodeStatus{Conditions: []v1.NodeCondition{{Type: v1.NodeDiskPressure, Status: v1.ConditionTrue}}},
			}}}},
			wantErr: true},
		{name: "has memory pressure condition",
			args: args{nodeInfo: &Node{Nodes: []v1.Node{{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status:     v1.NodeStatus{Conditions: []v1.NodeCondition{{Type: v1.NodeMemoryPressure, Status: v1.ConditionTrue}}},
			}}}},
			wantErr: true},
		{name: "has network unavailable condition",
			args: args{nodeInfo: &Node{Nodes: []v1.Node{{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status:     v1.NodeStatus{Conditions: []v1.NodeCondition{{Type: v1.NodeNetworkUnavailable, Status: v1.ConditionTrue}}},
			}}}},
			wantErr: true},
		{name: "has PID pressure condition",
			args: args{nodeInfo: &Node{Nodes: []v1.Node{{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status:     v1.NodeStatus{Conditions: []v1.NodeCondition{{Type: v1.NodePIDPressure, Status: v1.ConditionTrue}}},
			}}}},
			wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckCondition(tt.args.nodeInfo); (err != nil) != tt.wantErr {
				t.Errorf("CheckTaint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
