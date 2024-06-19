package health

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestNode_String(t *testing.T) {
	type fields struct {
		Nodes []v1.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "no nodes", fields: fields{Nodes: []v1.Node{}}, want: ""},
		{name: "no nodes", fields: fields{Nodes: nil}, want: ""},
		{name: "1 node", fields: fields{Nodes: []v1.Node{{ObjectMeta: metav1.ObjectMeta{Name: "hello-world"}}}}, want: "node: hello-world"},
		{name: "2 node", fields: fields{Nodes: []v1.Node{
			{ObjectMeta: metav1.ObjectMeta{Name: "hello"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "world"}},
		}}, want: "node: hello, node: world"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &Node{
				Nodes: tt.fields.Nodes,
			}
			if got := info.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKubeletEvictionFsByPercentage(t *testing.T) {
	type args struct {
		percentage int
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"-1", args{percentage: -1}, "", true},
		{"0", args{percentage: 0}, "imagefs.available<0%,nodefs.available<0%", false},
		{"50", args{percentage: 50}, "imagefs.available<50%,nodefs.available<50%", false},
		{"100", args{percentage: 100}, "imagefs.available<100%,nodefs.available<100%", false},
		{"101", args{percentage: 101}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := KubeletEvictionFsByPercentage(tt.args.percentage)
			if (err != nil) != tt.wantErr {
				t.Errorf("KubeletEvictionFsByPercentage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("KubeletEvictionFsByPercentage() got = %v, want %v", got, tt.want)
			}
		})
	}
}
