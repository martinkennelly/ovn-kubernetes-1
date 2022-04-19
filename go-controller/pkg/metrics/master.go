package metrics

import (
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/ovn-org/libovsdb/cache"
	libovsdbclient "github.com/ovn-org/libovsdb/client"
	"github.com/ovn-org/libovsdb/model"
	"github.com/ovn-org/libovsdb/ovsdb"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/config"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/kube"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/libovsdbops"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/nbdb"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/sbdb"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"

	"github.com/prometheus/client_golang/prometheus"
	kapi "k8s.io/api/core/v1"
	kapimtypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	klog "k8s.io/klog/v2"
)

// metricNbE2eTimestamp is the UNIX timestamp value set to NB DB. Northd will eventually copy this
// timestamp from NB DB to SB DB. The metric 'sb_e2e_timestamp' stores the timestamp that is
// read from SB DB. This is registered within func RunTimestamp in order to allow gathering this
// metric on the fly when metrics are scraped.
var metricNbE2eTimestamp = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "nb_e2e_timestamp",
	Help:      "The current e2e-timestamp value as written to the northbound database"},
)

// metricDbTimestamp is the UNIX timestamp seen in NB and SB DBs.
var metricDbTimestamp = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: MetricOvnNamespace,
	Subsystem: MetricOvnSubsystemDB,
	Name:      "e2e_timestamp",
	Help:      "The current e2e-timestamp value as observed in this instance of the database"},
	[]string{
		"db_name",
	},
)

// metricPodCreationLatency is the time between a pod being scheduled and the
// ovn controller setting the network annotations.
var metricPodCreationLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "pod_creation_latency_seconds",
	Help:      "The latency between pod creation and setting the OVN annotations",
	Buckets:   prometheus.ExponentialBuckets(.1, 2, 15),
})

// metricOvnCliLatency is the time between a pod being scheduled and the
// ovn controller setting the network annotations.
var metricOvnCliLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "ovn_cli_latency_seconds",
	Help:      "The latency of various OVN commands. Currently, ovn-nbctl and ovn-sbctl",
	Buckets:   prometheus.ExponentialBuckets(.1, 2, 15)},
	// labels
	[]string{"command"},
)

// MetricResourceUpdateCount is the number of times a particular resource's UpdateFunc has been called.
var MetricResourceUpdateCount = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "resource_update_total",
	Help:      "The number of times a given resource event (add, update, or delete) has been handled"},
	[]string{
		"name",
		"event",
	},
)

// MetricResourceAddLatency is the time taken to complete resource update by an handler.
// This measures the latency for all of the handlers for a given resource.
var MetricResourceAddLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "resource_add_latency_seconds",
	Help:      "The duration to process all handlers for a given resource event - add.",
	Buckets:   prometheus.ExponentialBuckets(.1, 2, 15)},
)

// MetricResourceUpdateLatency is the time taken to complete resource update by an handler.
// This measures the latency for all of the handlers for a given resource.
var MetricResourceUpdateLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "resource_update_latency_seconds",
	Help:      "The duration to process all handlers for a given resource event - update.",
	Buckets:   prometheus.ExponentialBuckets(.1, 2, 15)},
)

// MetricResourceDeleteLatency is the time taken to complete resource update by an handler.
// This measures the latency for all of the handlers for a given resource.
var MetricResourceDeleteLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "resource_delete_latency_seconds",
	Help:      "The duration to process all handlers for a given resource event - delete.",
	Buckets:   prometheus.ExponentialBuckets(.1, 2, 15)},
)

// MetricRequeueServiceCount is the number of times a particular service has been requeued.
var MetricRequeueServiceCount = prometheus.NewCounter(prometheus.CounterOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "requeue_service_total",
	Help:      "A metric that captures the number of times a service is requeued after failing to sync with OVN"},
)

// MetricSyncServiceCount is the number of times a particular service has been synced.
var MetricSyncServiceCount = prometheus.NewCounter(prometheus.CounterOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "sync_service_total",
	Help:      "A metric that captures the number of times a service is synced with OVN load balancers"},
)

// MetricSyncServiceLatency is the time taken to sync a service with the OVN load balancers.
var MetricSyncServiceLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "sync_service_latency_seconds",
	Help:      "The latency of syncing a service with the OVN load balancers",
	Buckets:   prometheus.ExponentialBuckets(.1, 2, 15)},
)

var MetricMasterReadyDuration = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "ready_duration_seconds",
	Help:      "The duration for the master to get to ready state",
})

// MetricMasterLeader identifies whether this instance of ovnkube-master is a leader or not
var MetricMasterLeader = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "leader",
	Help:      "Identifies whether the instance of ovnkube-master is a leader(1) or not(0).",
})

var metricV4HostSubnetCount = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "num_v4_host_subnets",
	Help:      "The total number of v4 host subnets possible",
})

var metricV6HostSubnetCount = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "num_v6_host_subnets",
	Help:      "The total number of v6 host subnets possible",
})

var metricV4AllocatedHostSubnetCount = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "allocated_v4_host_subnets",
	Help:      "The total number of v4 host subnets currently allocated",
})

var metricV6AllocatedHostSubnetCount = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "allocated_v6_host_subnets",
	Help:      "The total number of v6 host subnets currently allocated",
})

var metricEgressIPCount = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "num_egress_ips",
	Help:      "The number of defined egress IP addresses",
})

var metricEgressFirewallRuleCount = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "num_egress_firewall_rules",
	Help:      "The number of egress firewall rules defined"},
)

var metricIPsecEnabled = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "ipsec_enabled",
	Help:      "Specifies whether IPSec is enabled for this cluster(1) or not enabled for this cluster(0)",
})

var metricEgressRoutingViaHost = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "egress_routing_via_host",
	Help:      "Specifies whether egress gateway mode is via host networking stack(1) or not(0)",
})

var metricEgressFirewallCount = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "num_egress_firewalls",
	Help:      "The number of egress firewall policies",
})

// metricFirstSeenLSPLatency is the time between a pod first seen in OVN-Kubernetes and its Logical Switch Port is created
var metricFirstSeenLSPLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "pod_first_seen_lsp_created_duration_seconds",
	Help:      "The duration between a pod first observed in OVN-Kubernetes and Logical Switch Port created",
	Buckets:   prometheus.ExponentialBuckets(.01, 2, 15),
})

var metricLSPPortBindingLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "pod_lsp_created_port_binding_duration_seconds",
	Help:      "The duration between a pods Logical Switch Port created and port binding observed in cache",
	Buckets:   prometheus.ExponentialBuckets(.01, 2, 15),
})

var metricPortBindingChassisLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "pod_port_binding_port_binding_chassis_duration_seconds",
	Help:      "The duration between a pods port binding observed and port binding chassis update observed in cache",
	Buckets:   prometheus.ExponentialBuckets(.01, 2, 15),
})

var metricPortBindingUpLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "pod_port_binding_chassis_port_binding_up_duration_seconds",
	Help:      "The duration between a pods port binding chassis update and port binding up observed in cache",
	Buckets:   prometheus.ExponentialBuckets(.01, 2, 15),
})

var metricConfigDurationStartCount = prometheus.NewCounter(prometheus.CounterOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "config_duration_start_total",
	Help:      "The total number of configuration duration monitors started",
})

// metricConfigDurationEndCount in conjunction with metricConfigDurationStartCount can be used to detect hung OVN-Controllers
var metricConfigDurationEndCount = prometheus.NewCounter(prometheus.CounterOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "config_duration_end_total",
	Help:      "The total number of configuration duration monitors ended",
})

var metricConfigDurationOvnLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
	Namespace: MetricOvnkubeNamespace,
	Subsystem: MetricOvnkubeSubsystemMaster,
	Name:      "network_programming_ovn_duration_seconds",
	Help:      "The duration between a transaction sent to OVN and applied to all nodes",
	Buckets: merge(
		prometheus.LinearBuckets(0.25, 0.25, 2), // 0.25s, 0.50s
		prometheus.LinearBuckets(1, 1, 59),      // 1s, 2s, 3s, ... 59s
		prometheus.LinearBuckets(60, 5, 12),     // 60s, 65s, 70s, ... 115s
		prometheus.LinearBuckets(120, 30, 11),   // 2min, 2.5min, 3min, ..., 7min
	),
})

const (
	globalOptionsTimestampField     = "e2e_timestamp"
	globalOptionsProbeIntervalField = "northd_probe_interval"
)

// RegisterMasterBase registers ovnkube master base metrics with the Prometheus registry.
// This function should only be called once.
func RegisterMasterBase() {
	prometheus.MustRegister(MetricMasterLeader)
	prometheus.MustRegister(MetricMasterReadyDuration)
	prometheus.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: MetricOvnkubeNamespace,
			Subsystem: MetricOvnkubeSubsystemMaster,
			Name:      "build_info",
			Help: "A metric with a constant '1' value labeled by version, revision, branch, " +
				"and go version from which ovnkube was built and when and who built it",
			ConstLabels: prometheus.Labels{
				"version":    "0.0",
				"revision":   config.Commit,
				"branch":     config.Branch,
				"build_user": config.BuildUser,
				"build_date": config.BuildDate,
				"goversion":  runtime.Version(),
			},
		},
		func() float64 { return 1 },
	))
}

// RegisterMasterPerformance registers metrics that help us understand ovnkube-master performance. Call once after LE is won.
func RegisterMasterPerformance(nbClient libovsdbclient.Client) {
	// No need to unregister because process exits when leadership is lost.
	prometheus.MustRegister(metricPodCreationLatency)
	prometheus.MustRegister(MetricResourceUpdateCount)
	prometheus.MustRegister(MetricResourceAddLatency)
	prometheus.MustRegister(MetricResourceUpdateLatency)
	prometheus.MustRegister(MetricResourceDeleteLatency)
	prometheus.MustRegister(MetricRequeueServiceCount)
	prometheus.MustRegister(MetricSyncServiceCount)
	prometheus.MustRegister(MetricSyncServiceLatency)
	prometheus.MustRegister(metricOvnCliLatency)
	// This is set to not create circular import between metrics and util package
	util.MetricOvnCliLatency = metricOvnCliLatency
	registerWorkqueueMetrics(MetricOvnkubeNamespace, MetricOvnkubeSubsystemMaster)
	prometheus.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: MetricOvnNamespace,
			Subsystem: MetricOvnSubsystemNorthd,
			Name:      "northd_probe_interval",
			Help: "The maximum number of milliseconds of idle time on connection to the OVN SB " +
				"and NB DB before sending an inactivity probe message",
		}, func() float64 {
			return getGlobalOptionsValue(nbClient, globalOptionsProbeIntervalField)
		},
	))
}

// RegisterMasterFunctional is a collection of metrics that help us understand ovnkube-master functions. Call once after
// LE is won.
func RegisterMasterFunctional() {
	// No need to unregister because process exits when leadership is lost.
	prometheus.MustRegister(metricV4HostSubnetCount)
	prometheus.MustRegister(metricV6HostSubnetCount)
	prometheus.MustRegister(metricV4AllocatedHostSubnetCount)
	prometheus.MustRegister(metricV6AllocatedHostSubnetCount)
	prometheus.MustRegister(metricEgressIPCount)
	prometheus.MustRegister(metricEgressFirewallRuleCount)
	prometheus.MustRegister(metricEgressFirewallCount)
	prometheus.MustRegister(metricEgressRoutingViaHost)
}

// RunTimestamp adds a goroutine that registers and updates timestamp metrics.
// This is so we can determine 'freshness' of the components NB/SB DB and northd.
// Function must be called once.
func RunTimestamp(stopChan <-chan struct{}, sbClient, nbClient libovsdbclient.Client) {
	// Metric named nb_e2e_timestamp is the UNIX timestamp this instance wrote to NB DB. Updated every 30s with the
	// current timestamp.
	prometheus.MustRegister(metricNbE2eTimestamp)

	// Metric named sb_e2e_timestamp is the UNIX timestamp observed in SB DB. The value is read from the SB DB
	// cache when metrics HTTP endpoint is scraped.
	prometheus.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: MetricOvnkubeNamespace,
			Subsystem: MetricOvnkubeSubsystemMaster,
			Name:      "sb_e2e_timestamp",
			Help:      "The current e2e-timestamp value as observed in the southbound database",
		}, func() float64 {
			return getGlobalOptionsValue(sbClient, globalOptionsTimestampField)
		}))

	// Metric named e2e_timestamp is the UNIX timestamp observed in NB and SB DBs cache with the DB name
	// (OVN_Northbound|OVN_Southbound) set as a label. Updated every 30s.
	prometheus.MustRegister(metricDbTimestamp)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				currentTime := time.Now().Unix()
				if setNbE2eTimestamp(nbClient, currentTime) {
					metricNbE2eTimestamp.Set(float64(currentTime))
				} else {
					metricNbE2eTimestamp.Set(0)
				}

				metricDbTimestamp.WithLabelValues(nbClient.Schema().Name).Set(getGlobalOptionsValue(nbClient, globalOptionsTimestampField))
				metricDbTimestamp.WithLabelValues(sbClient.Schema().Name).Set(getGlobalOptionsValue(sbClient, globalOptionsTimestampField))
			case <-stopChan:
				return
			}
		}
	}()
}

// RecordPodCreated extracts the scheduled timestamp and records how long it took
// us to notice this and set up the pod's scheduling.
func RecordPodCreated(pod *kapi.Pod) {
	t := time.Now()

	// Find the scheduled timestamp
	for _, cond := range pod.Status.Conditions {
		if cond.Type != kapi.PodScheduled {
			continue
		}
		if cond.Status != kapi.ConditionTrue {
			return
		}
		creationLatency := t.Sub(cond.LastTransitionTime.Time).Seconds()
		metricPodCreationLatency.Observe(creationLatency)
		return
	}
}

// RecordSubnetUsage records the number of subnets allocated for nodes
func RecordSubnetUsage(v4SubnetsAllocated, v6SubnetsAllocated float64) {
	metricV4AllocatedHostSubnetCount.Set(v4SubnetsAllocated)
	metricV6AllocatedHostSubnetCount.Set(v6SubnetsAllocated)
}

// RecordSubnetCount records the number of available subnets per configuration
// for ovn-kubernetes
func RecordSubnetCount(v4SubnetCount, v6SubnetCount float64) {
	metricV4HostSubnetCount.Set(v4SubnetCount)
	metricV6HostSubnetCount.Set(v6SubnetCount)
}

// RecordEgressIPCount records the total number of Egress IPs.
// This total may include multiple Egress IPs per EgressIP CR.
func RecordEgressIPCount(count float64) {
	metricEgressIPCount.Set(count)
}

// UpdateEgressFirewallRuleCount records the number of Egress firewall rules.
func UpdateEgressFirewallRuleCount(count float64) {
	metricEgressFirewallRuleCount.Add(count)
}

// RecordEgressRoutingViaHost records the egress gateway mode of the cluster
// The values are:
// 0: If it is shared gateway mode
// 1: If it is local gateway mode
// 2: invalid mode
func RecordEgressRoutingViaHost() {
	if config.Gateway.Mode == config.GatewayModeLocal {
		// routingViaHost is enabled
		metricEgressRoutingViaHost.Set(1)
	} else if config.Gateway.Mode == config.GatewayModeShared {
		// routingViaOVN is enabled
		metricEgressRoutingViaHost.Set(0)
	} else {
		// invalid mode
		metricEgressRoutingViaHost.Set(2)
	}
}

// MonitorIPSec will register a metric to determine if IPSec is enabled/disabled. It will also add a handler
// to NB libovsdb cache to update the IPSec metric.
// This function should only be called once.
func MonitorIPSec(ovnNBClient libovsdbclient.Client) {
	prometheus.MustRegister(metricIPsecEnabled)
	ovnNBClient.Cache().AddEventHandler(&cache.EventHandlerFuncs{
		AddFunc: func(table string, model model.Model) {
			ipsecMetricHandler(table, model)
		},
		UpdateFunc: func(table string, _, new model.Model) {
			ipsecMetricHandler(table, new)
		},
		DeleteFunc: func(table string, model model.Model) {
			ipsecMetricHandler(table, model)
		},
	})
}

func ipsecMetricHandler(table string, model model.Model) {
	if table != "NB_Global" {
		return
	}
	entry := model.(*nbdb.NBGlobal)
	if entry.Ipsec {
		metricIPsecEnabled.Set(1)
	} else {
		metricIPsecEnabled.Set(0)
	}
}

// IncrementEgressFirewallCount increments the number of Egress firewalls
func IncrementEgressFirewallCount() {
	metricEgressFirewallCount.Inc()
}

// DecrementEgressFirewallCount decrements the number of Egress firewalls
func DecrementEgressFirewallCount() {
	metricEgressFirewallCount.Dec()
}

type (
	timestampType int
	operation     int
)

const (
	// pod event first handled by OVN-Kubernetes control plane
	firstSeen timestampType = iota
	// OVN-Kubernetes control plane created Logical Switch Port in northbound database
	logicalSwitchPort
	// port binding seen in OVN-Kubernetes control plane southbound database libovsdb cache
	portBinding
	// port binding with updated chassis seen in OVN-Kubernetes control plane southbound database libovsdb cache
	portBindingChassis
	// queue operations
	addPortBinding operation = iota
	updatePortBinding
	addPod
	cleanPod
	addLogicalSwitchPort
	queueCheckPeriod = time.Millisecond * 50
	// prevent OOM by limiting queue size
	queueLimit       = 10000
	portBindingTable = "Port_Binding"
)

type record struct {
	timestamp time.Time
	timestampType
}

type item struct {
	op        operation
	timestamp time.Time
	old       model.Model
	new       model.Model
	uid       kapimtypes.UID
}

type PodRecorder struct {
	records map[kapimtypes.UID]*record
	queue   workqueue.Interface
}

func NewPodRecorder() PodRecorder {
	return PodRecorder{}
}

var podRecorderRegOnce sync.Once

//Run monitors pod setup latency
func (pr *PodRecorder) Run(sbClient libovsdbclient.Client, stop <-chan struct{}) {
	podRecorderRegOnce.Do(func() {
		prometheus.MustRegister(metricFirstSeenLSPLatency)
		prometheus.MustRegister(metricLSPPortBindingLatency)
		prometheus.MustRegister(metricPortBindingUpLatency)
		prometheus.MustRegister(metricPortBindingChassisLatency)
	})

	pr.queue = workqueue.New()
	pr.records = make(map[kapimtypes.UID]*record)

	sbClient.Cache().AddEventHandler(&cache.EventHandlerFuncs{
		AddFunc: func(table string, model model.Model) {
			if table != portBindingTable {
				return
			}
			if !pr.queueFull() {
				pr.queue.Add(item{op: addPortBinding, old: model, timestamp: time.Now()})
			}
		},
		UpdateFunc: func(table string, old model.Model, new model.Model) {
			if table != portBindingTable {
				return
			}
			if !pr.queueFull() {
				pr.queue.Add(item{op: updatePortBinding, old: old, new: new, timestamp: time.Now()})
			}
		},
	})

	go func() {
		wait.Until(pr.runWorker, queueCheckPeriod, stop)
		pr.queue.ShutDown()
	}()
}

func (pr *PodRecorder) AddPod(podUID kapimtypes.UID) {
	if pr.queue != nil && !pr.queueFull() {
		pr.queue.Add(item{op: addPod, uid: podUID, timestamp: time.Now()})
	}
}

func (pr *PodRecorder) CleanPod(podUID kapimtypes.UID) {
	if pr.queue != nil && !pr.queueFull() {
		pr.queue.Add(item{op: cleanPod, uid: podUID})
	}
}

func (pr *PodRecorder) AddLSP(podUID kapimtypes.UID) {
	if pr.queue != nil && !pr.queueFull() {
		pr.queue.Add(item{op: addLogicalSwitchPort, uid: podUID, timestamp: time.Now()})
	}
}

func (pr *PodRecorder) addLSP(podUID kapimtypes.UID, t time.Time) {
	var r *record
	if r = pr.getRecord(podUID); r == nil {
		klog.V(5).Infof("Add Logical Switch Port event expected pod with UID %q in cache", podUID)
		return
	}
	if r.timestampType != firstSeen {
		klog.V(5).Infof("Unexpected last event type (%d) in cache for pod with UID %q", r.timestampType, podUID)
		return
	}
	metricFirstSeenLSPLatency.Observe(t.Sub(r.timestamp).Seconds())
	r.timestamp = t
	r.timestampType = logicalSwitchPort
}

func (pr *PodRecorder) addPortBinding(m model.Model, t time.Time) {
	var r *record
	row := m.(*sbdb.PortBinding)
	podUID := getPodUIDFromPortBinding(row)
	if podUID == "" {
		return
	}
	if r = pr.getRecord(podUID); r == nil {
		klog.V(5).Infof("Add port binding event expected pod with UID %q in cache", podUID)
		return
	}
	if r.timestampType != logicalSwitchPort {
		klog.V(5).Infof("Unexpected last event entry (%d) in cache for pod with UID %q", r.timestampType, podUID)
		return
	}
	metricLSPPortBindingLatency.Observe(t.Sub(r.timestamp).Seconds())
	r.timestamp = t
	r.timestampType = portBinding
}

func (pr *PodRecorder) updatePortBinding(old, new model.Model, t time.Time) {
	var r *record
	oldRow := old.(*sbdb.PortBinding)
	newRow := new.(*sbdb.PortBinding)
	podUID := getPodUIDFromPortBinding(newRow)
	if podUID == "" {
		return
	}
	if r = pr.getRecord(podUID); r == nil {
		klog.V(5).Infof("Port binding update expected pod with UID %q in cache", podUID)
		return
	}
	if oldRow.Chassis == nil && newRow.Chassis != nil && r.timestampType == portBinding {
		metricPortBindingChassisLatency.Observe(t.Sub(r.timestamp).Seconds())
		r.timestamp = t
		r.timestampType = portBindingChassis
	}
	if oldRow.Up != nil && !*oldRow.Up && newRow.Up != nil && *newRow.Up && r.timestampType == portBindingChassis {
		metricPortBindingUpLatency.Observe(t.Sub(r.timestamp).Seconds())
		delete(pr.records, podUID)
	}
}

func (pr *PodRecorder) queueFull() bool {
	return pr.queue.Len() >= queueLimit
}

func (pr *PodRecorder) runWorker() {
	for pr.processNextItem() {
	}
}

func (pr *PodRecorder) processNextItem() bool {
	i, term := pr.queue.Get()
	if term {
		return false
	}
	pr.processItem(i.(item))
	pr.queue.Done(i)
	return true
}

func (pr *PodRecorder) processItem(i item) {
	switch i.op {
	case addPortBinding:
		pr.addPortBinding(i.old, i.timestamp)
	case updatePortBinding:
		pr.updatePortBinding(i.old, i.new, i.timestamp)
	case addPod:
		pr.records[i.uid] = &record{timestamp: i.timestamp, timestampType: firstSeen}
	case cleanPod:
		delete(pr.records, i.uid)
	case addLogicalSwitchPort:
		pr.addLSP(i.uid, i.timestamp)
	}
}

// getRecord returns record from map with func argument as the key
func (pr *PodRecorder) getRecord(podUID kapimtypes.UID) *record {
	r, ok := pr.records[podUID]
	if !ok {
		klog.V(5).Infof("Cache entry expected pod with UID %q but failed to find it", podUID)
		return nil
	}
	return r
}

func getPodUIDFromPortBinding(row *sbdb.PortBinding) kapimtypes.UID {
	if isPod, ok := row.ExternalIDs["pod"]; !ok || isPod != "true" {
		return ""
	}
	podUID, ok := row.Options["iface-id-ver"]
	if !ok {
		return ""
	}
	return kapimtypes.UID(podUID)
}

const (
	startRecordChSize = 350
	endRecordChSize   = 400
	stopMsg           = "Stopping config duration monitor due to node count greater than %d " +
		"to protect southbound database from overload"
	chFullMsg = "Skip record config duration due to channel full"
	// node limit to prevent SB DB pressure when large node counts
	nodeLimit       = 100
	nbGlobalTable   = "NB_Global"
	PodKindName     = "pod"
	ServiceKindName = "service"
)

type startRecordEvent struct {
	kind               string
	kindStartTimestamp int
	txStartTimestamp   int
	nbCfg              int
}

type endRecordEvent struct {
	hvCfgTimestamp int
	hvCfg          int
}

type ConfigDurationRecorder struct {
	mu              sync.Mutex
	enabled         bool
	recordAvailable bool
	reportKind      map[string]prometheus.Histogram
	startRecords    []*startRecordEvent
	startRecordCh   chan startRecordEvent
	endRecordCh     chan endRecordEvent
}

// global variable is needed because this functionality is accessed in many functions
var cdr *ConfigDurationRecorder

func GetConfigDurationRecorder() *ConfigDurationRecorder {
	if cdr == nil {
		cdr = &ConfigDurationRecorder{}
	}
	return cdr
}

var configDurationRegOnce sync.Once

//Run monitors the config duration for OVN-Kube master to configure k8 kinds
func (cdr *ConfigDurationRecorder) Run(nbClient libovsdbclient.Client, kube kube.Interface, delayPerNode,
	checkPeriod time.Duration, stop <-chan struct{}) {
	// ** configuration duration monitor - intro **
	// We measure the duration to configure whatever k8 kind (pod, services, etc.) object to all nodes and reports
	// this as a metric. This will give a rough upper bound of how long it takes OVN-Kubernetes master container (CMS)
	// and OVN to configure all nodes under its control.

	// When the CMS is about to make a transaction to configure whatever kind, config duration monitor provides a
	// mechanism to allow the caller to pass the time when the CMS first started processing the kind it wants to measure.
	// This time is stored for later processing and is used as the start time.
	// An operation is returned to the caller, which they can bundle with their existing transactions sent to OVN which
	// will tell OVN to measure how long it takes to configure all nodes with the config in the transaction.
	// Config duration then waits for OVN to configure all nodes and calculates the time delta.
	// The measurements are limited and are only allowed during certain time periods to reduce the stress on OVN.

	// ** configuration duration monitor - caveats **
	// The duration described above does not give you an exact time duration for how long it takes to configure your
	// k8 kind. When you are recording how long it takes OVN to complete your configuration to all nodes, other
	// transactions may have occurred which may increases the overall time. You may also get longer processing times if one
	// or more nodes are unavailable because we are measuring how long the functionality takes to apply to ALL nodes.

	// ** configuration duration monitor - How the duration of the config is measured within OVN **
	// We increment the nb_cfg integer value in the NB_Global table.
	// ovn-northd notices the nb_cfg change and copies the nb_cfg value to SB_Global table field nb_cfg along with any
	// other configuration that is changed in OVN Northbound database.
	// All ovn-controllers detect nb_cfg value change and generate a 'barrier' on the openflow connection to the
	// nodes ovs-vswitchd. Once ovn-controllers receive the 'barrier processed' reply from ovs-vswitchd which
	// indicates that all relevant openflow operations associated with NB_Globals nb_cfg value have been
	// propagated to the nodes OVS, it copies the SB_Global nb_cfg value to its Chassis_Private table nb_cfg record.
	// ovn-northd detects changes to the Chassis_Private startRecords and computes the minimum nb_cfg for all Chassis_Private
	// nb_cfg and stores this in NB_Global hv_cfg field along with a timestamp to field hv_cfg_timestamp which
	// reflects the time when the slowest chassis catches up with the northbound configuration.

	// create a map with key as the name of the k8 kind you want to measure and value with the histogram to report
	// the duration to configure the kind to all nodes. Suggestion to name the key like so: {k8 kind}. E.g. pod
	cdr.reportKind = make(map[string]prometheus.Histogram)
	// register metrics which will take account of configuration start time and the time it
	// takes OVN to configure all nodes. Note that the result will only give an upper bound limit of how long it
	// takes to roll out the configuration to all nodes. To make matters more uncertain, the measurements are currently
	// sampled.
	cdr.reportKind[PodKindName] = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: MetricOvnkubeNamespace,
		Subsystem: MetricOvnkubeSubsystemMaster,
		Name:      "network_programming_pod_duration_seconds",
		Help:      "Upperbound time to configure a pod from when its seen and config is applied to all nodes",
		Buckets:   prometheus.ExponentialBuckets(.01, 3, 11),
	})
	cdr.reportKind[ServiceKindName] = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: MetricOvnkubeNamespace,
		Subsystem: MetricOvnkubeSubsystemMaster,
		Name:      "network_programming_service_duration_seconds",
		Help:      "Upperbound time to configure a service from when its seen and config is applied to all nodes",
		Buckets:   prometheus.ExponentialBuckets(.01, 3, 11),
	})
	configDurationRegOnce.Do(func() {
		prometheus.MustRegister(cdr.reportKind[PodKindName])
		prometheus.MustRegister(cdr.reportKind[ServiceKindName])
		prometheus.MustRegister(metricConfigDurationStartCount)
		prometheus.MustRegister(metricConfigDurationEndCount)
		prometheus.MustRegister(metricConfigDurationOvnLatency)
	})

	// we currently do not clean the follow channels up upon exit
	cdr.startRecordCh = make(chan startRecordEvent, startRecordChSize)
	cdr.endRecordCh = make(chan endRecordEvent, endRecordChSize)
	go cdr.processChannelEvents(stop)

	// determine if we should be enabled or not during runtime
	cdr.enabled = true
	cdr.runEnable(kube, checkPeriod, nodeLimit, stop)

	// record measurements are only available during periods of time. We use a ticker to set the availability of measurements.
	// The ticker duration is continuously updated and is proportional to the node count.
	// Determine the period (for the ticker) which record measurements are available. This can be updated during the runtime.
	availableTicker := time.NewTicker(10 * time.Minute)
	cdr.runRecordAvailableDuration(kube, availableTicker, delayPerNode, checkPeriod, stop)
	// update the availability of measurements based on the duration determined above
	cdr.runRecordAvailable(availableTicker, stop)

	nbClient.Cache().AddEventHandler(&cache.EventHandlerFuncs{
		UpdateFunc: func(table string, old model.Model, new model.Model) {
			if table != nbGlobalTable {
				return
			}
			// disable adding items to queue if functionality is disabled due to large node count
			if !cdr.enabled {
				return
			}
			oldRow := old.(*nbdb.NBGlobal)
			newRow := new.(*nbdb.NBGlobal)

			if oldRow.HvCfg != newRow.HvCfg && oldRow.HvCfgTimestamp != newRow.HvCfgTimestamp && newRow.HvCfgTimestamp > 0 {
				select {
				case cdr.endRecordCh <- endRecordEvent{hvCfg: newRow.HvCfg, hvCfgTimestamp: newRow.HvCfgTimestamp}:
				default:
					klog.Warning(chFullMsg)
				}
			}
		},
	})
}

// RecordConfigDuration allows the caller to request measurement of a configuration, as a metric,
// the duration between the argument start and OVN applying the configuration to all nodes.
// It will return ovsdb operations which a user can add to existing operations they wish to track.
// Upon successful transaction of the operations to the ovsdb server, the user must call a call-back function to
// lock-in the request to measure and report. Failure to call the call-back function, will result in no measurement and
// no metrics reported. Not every call will return operations to report the duration and if so, the call-back will be a
// no-op if the function is called. If start time is zero, we return a no-op.
func (cdr *ConfigDurationRecorder) RecordConfigDuration(nbClient libovsdbclient.Client, kind string, start time.Time) (
	[]ovsdb.Operation, func(), error) {
	if !cdr.enabled || !cdr.isPeriodToTrack() || start.IsZero() {
		return []ovsdb.Operation{}, func() {}, nil
	}
	if len(cdr.startRecordCh) >= startRecordChSize {
		klog.Warning(chFullMsg)
		return []ovsdb.Operation{}, func() {}, nil
	}
	if nbClient.Schema().Name != "OVN_Northbound" {
		return []ovsdb.Operation{}, func() {}, fmt.Errorf("expected OVN_Northbound libovsdb client but got %q",
			nbClient.Schema().Name)
	}

	_, ok := cdr.reportKind[kind]
	if !ok {
		return []ovsdb.Operation{}, func() {}, fmt.Errorf("unknown kind %q", kind)
	}

	nbGlobal, err := libovsdbops.FindNBGlobal(nbClient)
	if err != nil {
		return []ovsdb.Operation{}, func() {}, fmt.Errorf("failed to find OVN Northbound NB_Global table"+
			" entry: %v", err)
	}
	ops, err := nbClient.Where(nbGlobal).Mutate(nbGlobal, model.Mutation{
		Field:   &nbGlobal.NbCfg,
		Mutator: ovsdb.MutateOperationAdd,
		Value:   1,
	})
	if err != nil {
		return []ovsdb.Operation{}, func() {}, fmt.Errorf("failed to create update operation: %v", err)
	}

	return ops, func() {
		// there can be a race condition here where we queue the wrong (nb)Cfg value, but it is ok as long as it is
		// less than or equal the hv_cfg value we see and this is the case because of atomic increments for nb_cfg
		select {
		case cdr.startRecordCh <- startRecordEvent{kind: kind, kindStartTimestamp: int(start.UnixMilli()),
			txStartTimestamp: int(time.Now().UnixMilli()), nbCfg: nbGlobal.NbCfg + 1}:
		default:
			klog.Warningf("Unable to start config duration recording due to queue full")
		}
	}, nil
}

func (cdr *ConfigDurationRecorder) processChannelEvents(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case sRec := <-cdr.startRecordCh:
			metricConfigDurationStartCount.Inc()
			cdr.startRecords = append(cdr.startRecords, &sRec)
		case eRec := <-cdr.endRecordCh:
			cdr.processEndRecord(eRec.hvCfg, eRec.hvCfgTimestamp)
		}
	}
}

// runRecordAvailableDuration will adjust the availability of measurements based on the number of nodes in the cluster
func (cdr *ConfigDurationRecorder) runRecordAvailableDuration(kube kube.Interface, ticker *time.Ticker, delayPerNode,
	nodeCheckPeriod time.Duration, stop <-chan struct{}) {
	var currentDuration time.Duration

	updateTickerDuration := func() {
		if nodeCount, err := getNodeCount(kube); err != nil {
			klog.Errorf("Failed to update configuration duration available ticker duration considering node"+
				" count: %v", err)
		} else {
			newDuration := time.Duration(nodeCount) * delayPerNode
			if newDuration != currentDuration {
				cdr.mu.Lock()
				if newDuration > 0 && ticker != nil {
					currentDuration = newDuration
					ticker.Reset(currentDuration)
				}
				cdr.mu.Unlock()
				klog.V(5).Infof("Updated configuration duration available measurement period to every %f seconds",
					newDuration.Seconds())
			}
		}
	}

	// initial ticker duration adjustment
	updateTickerDuration()

	go func() {
		nodeCheckTicker := time.NewTicker(nodeCheckPeriod)
		defer nodeCheckTicker.Stop()
		for {
			select {
			case <-nodeCheckTicker.C:
				updateTickerDuration()
			case <-stop:
				return
			}
		}
	}()
}

func (cdr *ConfigDurationRecorder) runRecordAvailable(ticker *time.Ticker, stop <-chan struct{}) {
	cdr.recordAvailable = true
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cdr.mu.Lock()
				cdr.recordAvailable = true
				cdr.mu.Unlock()
			case <-stop:
				return
			}
		}
	}()
}

// runEnable will disable, if arg stop triggers or node count size is above nodeLimit
func (cdr *ConfigDurationRecorder) runEnable(kube kube.Interface, checkPeriod time.Duration, nodeLimit int, stop <-chan struct{}) {
	// determine when to serve and stop serving

	configEnable := func() {
		if count, err := getNodeCount(kube); err == nil && count >= nodeLimit {
			if cdr.enabled {
				cdr.enabled = false
				klog.Infof(stopMsg, nodeLimit)
			}
		} else {
			if !cdr.enabled {
				cdr.enabled = true
				klog.Infof("Starting configuration duration recorder because node count under %d", nodeLimit)
			}
		}
	}

	configEnable()

	go func() {
		tick := time.NewTicker(checkPeriod)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				configEnable()
			case <-stop:
				cdr.enabled = false
				return
			}
		}
	}()
}

func (cdr *ConfigDurationRecorder) isPeriodToTrack() bool {
	cdr.mu.Lock()
	defer cdr.mu.Unlock()
	if cdr.recordAvailable {
		cdr.recordAvailable = false
		return true
	}
	return false
}

func getNodeCount(kube kube.Interface) (int, error) {
	nodes, err := kube.GetNodes()
	if err != nil {
		return 0, fmt.Errorf("unable to retrieve node list: %v", err)
	}
	return len(nodes.Items), nil
}

func (cdr *ConfigDurationRecorder) processEndRecord(hvCfg, hvCfgTimestamp int) {
	var gcIndex, delta int
	for _, cr := range cdr.startRecords {
		// OVN-Controllers may not report back that they have installed the config associated with every nb_cfg value.
		// For example if we incremented nb_cfg rapidly to 1,2 and 3, the CMS may only see hv_cfg value of 3.
		// We assume that the updates associated with 1,2 are completed when we see a larger value.
		if cr.nbCfg <= hvCfg {
			metricConfigDurationEndCount.Inc()
			h, ok := cdr.reportKind[cr.kind]
			if !ok {
				klog.Errorf("Failed to find kind %q. Unable to report its metric", cr.kind)
				continue
			}
			// delta is milliseconds
			delta = hvCfgTimestamp - cr.kindStartTimestamp
			if delta < 0 {
				klog.Error("Unexpected negative timestamp between hv_cfg timestamp and kind start timestamp. Discarding")
				continue
			}
			h.Observe(float64(delta) / 1000)
			delta = hvCfgTimestamp - cr.txStartTimestamp
			if delta < 0 {
				klog.Error("Unexpected negative timestamp between hv_cfg timestamp and tx start timestamp. Discarding")
				continue
			}
			metricConfigDurationOvnLatency.Observe(float64(delta) / 1000)
		} else {
			// In order to not incur the overhead of copying to a new slice, we save c pointer for
			// later processing
			cdr.startRecords[gcIndex] = cr
			gcIndex++
		}
	}
	// Set processed pointers to nil to allow GC
	for i := gcIndex; i < len(cdr.startRecords); i++ {
		cdr.startRecords[i] = nil
	}
	// Re-adjust slice size. Keep underlying array and compact it
	cdr.startRecords = cdr.startRecords[:gcIndex]
}

// setNbE2eTimestamp return true if setting timestamp to NB global options is successful
func setNbE2eTimestamp(ovnNBClient libovsdbclient.Client, timestamp int64) bool {
	// assumption that only first row is relevant in NB_Global table
	options := map[string]string{globalOptionsTimestampField: fmt.Sprintf("%d", timestamp)}
	if err := libovsdbops.UpdateNBGlobalOptions(ovnNBClient, options); err != nil {
		klog.Errorf("Unable to update NB global options E2E timestamp metric err: %v", err)
		return false
	}
	return true
}

func getGlobalOptionsValue(client libovsdbclient.Client, field string) float64 {
	var options map[string]string
	dbName := client.Schema().Name

	if dbName == "OVN_Northbound" {
		if nbGlobal, err := libovsdbops.FindNBGlobal(client); err != nil && err != libovsdbclient.ErrNotFound {
			klog.Errorf("Failed to get NB_Global table err: %v", err)
			return 0
		} else {
			options = nbGlobal.Options
		}
	}

	if dbName == "OVN_Southbound" {
		if sbGlobal, err := libovsdbops.FindSBGlobal(client); err != nil && err != libovsdbclient.ErrNotFound {
			klog.Errorf("Failed to get SB_Global table err: %v", err)
			return 0
		} else {
			options = sbGlobal.Options
		}
	}

	if v, ok := options[field]; !ok {
		klog.V(5).Infof("Failed to find %q from %s options. This may occur at startup.", field, dbName)
		return 0
	} else {
		if value, err := strconv.ParseFloat(v, 64); err != nil {
			klog.Errorf("Failed to parse %q value to float64 err: %v", field, err)
			return 0
		} else {
			return value
		}
	}
}

// merge direct copy from k8 pkg/proxy/metrics/metrics.go
func merge(slices ...[]float64) []float64 {
	result := make([]float64, 1)
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}
