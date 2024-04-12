# testclusters-go Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

[Unreleased]

## Added

- add `cluster.*K3dCluster.WriteKubeConfig()` to produce a KUBECONFIG for cluster debugging
- [#16] provide feedback on node health for more cluster transparency
- add convenience constructor `cluster.NewK3dClusterWithOpts()` for further cluster customizing
  - f. i. log level or custom cluster prefix

## Changed
- [#16] clean up cluster containers more robustly
  - containers may still prevail cleaning if the cluster test will be hard-terminated (f. i. pressing the Debug-Kill 💀 button in IntelliJ IDEA)
