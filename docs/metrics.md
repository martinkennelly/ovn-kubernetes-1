# Metrics
## OVN-Kubernetes master
This includes a description of a selective set of metrics and to explore the exhausted set, see `go-controller/pkg/metrics/master.go`
### Configuration duration
#### Setup
Enabled by default with the `kind.sh` (in directory `$ROOT/contrib`) [Kind](https://kind.sigs.k8s.io/) setup script.
Disabled by default for binary ovnkube-master and enabled with flag `--enable-config-duration`.
Disabled if node count is greater than or equal 100  to protect OVN from load.
#### High-level description
This set of metrics gives a result for the upper bound duration which means, it has taken at most this amount of seconds to apply the configuration to all nodes. It does not represent the exact accurate time to apply only this configuration.
Measurement accuracy can be impacted by other parallel processing that might be occurring while the measurement is in progress therefore, the accuracy of the measurements should only indicate upper bound duration to roll out configuration changes.
#### Metrics
| Name | Prometheus type | Description  |
|--|--|--|
|ovnkube_master_network_programming_ovn_duration_seconds| Histogram  | The upper bound duration in seconds between a transaction sent to OVN from OVN-Kubernetes master and the application of this transaction to all nodes.
|ovnkube_master_network_programming_pod_duration_seconds| Histogram  | The upper bound duration in seconds to configure a pod from when the pod handler function is executed and its configuration is applied to all nodes.
|ovnkube_master_network_programming_service_duration_seconds| Histogram  | The upper bounded duration in seconds to configure a service from when OVN-Kubernetes master starts processing it following popping it from a work queue to when its configuration is applied to all nodes.
