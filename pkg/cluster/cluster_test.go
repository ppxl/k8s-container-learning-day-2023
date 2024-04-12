package cluster

import (
	"context"
	_ "embed"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fake "k8s.io/client-go/kubernetes/fake"
)

var testCtx = context.Background()

func TestK3dCluster_NodeInfo(t *testing.T) {
	t.Run("should return node info", func(t *testing.T) {
		// given
		node := &v1.Node{
			ObjectMeta: metav1.ObjectMeta{Namespace: "node-namespace", Name: "node-name"},
		}
		clientset := fake.NewSimpleClientset(node)
		sut := K3dCluster{clientSet: clientset}

		// when
		actual, err := sut.NodeInfo(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, NodeInfo{node}, actual)
	})
}
