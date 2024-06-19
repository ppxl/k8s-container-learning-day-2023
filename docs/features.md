# Features

- Full* cluster functionality at test time
   - *YMMV for full cluster features
      - f. e. storage snapshot controllers are not provided by K3s
- clean up containers during start-up failure
   - nobody likes to clean up after other tests ;)
- expose `kubeconfig` to test developer
   - :note: do you want to debug containers? It does not have to be containers :note:
- apply kubernetes resources at cluster start-up time
   - simplify repeated tasks
- kubectl-like applying of kubernetes YAML resources thanks to the
  Cloudogu [apply-lib](https://github.com/cloudogu/k8s-apply-lib)
   - do you want to have resources? Because that's how you get resources
- Enable external access to cluster pods
   - Loadbalancer/ingress testing
   - port forward
- generate cluster identifiers automatically
- allow user to choose a custom namespace
- Test framework agnostic
- checks node health on cluster start

## Configure hard eviction string for kubelet

Sometimes the developer's computer has very little resources. Often this leads during testing to tainted nodes along
with an associated node condition.

But developers often know better which resources are needed and when the cluster is to bust. With help of `cluster.Opts` this can be configured
(currently only for node disk pressure).

```golang
package yourtestpackage

import (
  "github.com/test-clusters/testclusters-go/pkg/cluster/health"
  "github.com/test-clusters/testclusters-go/pkg/cluster"
  "testing"
)

func TestYourTestname(t *testing.T) {
  hardEviction2percent, _ := health.KubeletEvictionFsByPercentage(2) // configure 2 % free hard disk to have no disk pressure condition on the nodes 
  cl := cluster.NewK3dClusterWithOpts(t, cluster.Opts{NodeConditionEvictionHardArg: hardEviction2percent})
  ...
}
```