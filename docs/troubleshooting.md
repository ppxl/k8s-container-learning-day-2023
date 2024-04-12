# Troubleshooting testclusters-go

## Remove remaining testclusters-go instances

Usually the cluster removes itself once the test exits in a normal way. But killing test processes might leave the started containers up. 

Remove remaining containers like this:

```bash
docker ps -f name=k3d-hello-world --format "{{.Names}}" | xargs docker rm -f
```

Remove remaining container networks like this:

```bash
docker network list -f name=k3d-hello --format "{{.Name}}" | xargs docker network rm
```

## Workloads do not arrive on node

You may want to check if the node was a failure condition. You can put a local breakpoint in your code and have testclusters-go export the respective KUBECONFIG:

```golang
func Test_yourTest(t *testing.T) {
	cl := cluster.NewK3dCluster(t)
	ctx := context.Background()
	pathToYouTempKubeConfig := cl.WriteKubeConfig(ctx, t)
	
	//put your break point after 
```
Events:                                                                                                                             │
│   Type     Reason                          Age                     From            Message                                          │
│   ----     ------                          ----                    ----            -------                                          │
│   Normal   Starting                        8m51s                   kube-proxy                                                       │
│   Normal   Starting                        8m55s                   kubelet         Starting kubelet.                                │
│   Warning  InvalidDiskCapacity             8m55s                   kubelet         invalid capacity 0 on image filesystem           │
│   Normal   NodeHasSufficientMemory         8m55s (x2 over 8m55s)   kubelet         Node k3d-tcg-hello-world-7d476345-server-0 statu │
│ s is now: NodeHasSufficientMemory                                                                                                   │
│   Normal   NodeHasNoDiskPressure           8m55s (x2 over 8m55s)   kubelet         Node k3d-tcg-hello-world-7d476345-server-0 statu │
│ s is now: NodeHasNoDiskPressure                                                                                                     │
│   Normal   NodeHasSufficientPID            8m55s (x2 over 8m55s)   kubelet         Node k3d-tcg-hello-world-7d476345-server-0 statu │
│ s is now: NodeHasSufficientPID                                                                                                      │
│   Normal   NodeReady                       8m55s                   kubelet         Node k3d-tcg-hello-world-7d476345-server-0 statu │
│ s is now: NodeReady                                                                                                                 │
│   Normal   NodeAllocatableEnforced         8m55s                   kubelet         Updated Node Allocatable limit across pods       │
│   Normal   NodePasswordValidationComplete  8m52s                   k3s-supervisor  Deferred node password secret validation complet │
│ e                                                                                                                                   │
│   Normal   NodeHasDiskPressure             8m45s                   kubelet         Node k3d-tcg-hello-world-7d476345-server-0 statu │
│ s is now: NodeHasDiskPressure                                                                                                       │
│   Warning  EvictionThresholdMet            6m35s (x14 over 8m45s)  kubelet         Attempting to reclaim ephemeral-storage          │
│   Warning  FreeDiskSpaceFailed             3m55s                   kubelet         Failed to garbage collect required amount of ima │
│ ges. Attempted to free 164183248076 bytes, but only found 0 bytes eligible to free.                                                 │
│                                                                                                                                     │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

### How can node taints like disk pressure be tested?

Create large files that fill up your disk.