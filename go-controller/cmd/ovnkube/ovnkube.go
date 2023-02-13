package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"text/tabwriter"
	"text/template"
	"time"

	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	libovsdbclient "github.com/ovn-org/libovsdb/client"
	"github.com/urfave/cli/v2"

	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/clustermanager"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/config"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/factory"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/libovsdb"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/metrics"
	controllerManager "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/network-controller-manager"
	ovnnode "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/types"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"

	kexec "k8s.io/utils/exec"
)

const (
	// CustomAppHelpTemplate helps in grouping options to ovnkube
	CustomAppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.HelpName}} [global options]

VERSION:
   {{.Version}}{{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}

COMMANDS:{{range .VisibleCategories}}{{if .Name}}

   {{.Name}}:{{end}}{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}

GLOBAL OPTIONS:{{range $title, $category := getFlagsByCategory}}
   {{upper $title}}
   {{range $index, $option := $category}}{{if $index}}
   {{end}}{{$option}}{{end}}
   {{end}}`
)

func getFlagsByCategory() map[string][]cli.Flag {
	m := map[string][]cli.Flag{}
	m["Generic Options"] = config.CommonFlags
	m["CNI Options"] = config.CNIFlags
	m["K8s-related Options"] = config.K8sFlags
	m["OVN Northbound DB Options"] = config.OvnNBFlags
	m["OVN Southbound DB Options"] = config.OvnSBFlags
	m["OVN Gateway Options"] = config.OVNGatewayFlags
	m["Master HA Options"] = config.MasterHAFlags
	m["OVN Kube Node Options"] = config.OvnKubeNodeFlags
	m["Monitoring Options"] = config.MonitoringFlags
	m["IPFIX Flow Tracing Options"] = config.IPFIXFlags

	return m
}

// borrowed from cli packages' printHelpCustom()
func printOvnKubeHelp(out io.Writer, templ string, data interface{}, customFunc map[string]interface{}) {
	funcMap := template.FuncMap{
		"join":               strings.Join,
		"upper":              strings.ToUpper,
		"getFlagsByCategory": getFlagsByCategory,
	}
	for key, value := range customFunc {
		funcMap[key] = value
	}

	w := tabwriter.NewWriter(out, 1, 8, 2, ' ', 0)
	t := template.Must(template.New("help").Funcs(funcMap).Parse(templ))
	err := t.Execute(w, data)
	if err == nil {
		_ = w.Flush()
	}
}

func main() {
	cli.HelpPrinterCustom = printOvnKubeHelp
	c := cli.NewApp()
	c.Name = "ovnkube"
	c.Usage = "run ovnkube to start master, node, and gateway services"
	c.Version = config.Version
	c.CustomAppHelpTemplate = CustomAppHelpTemplate
	c.Flags = config.GetFlags(nil)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	c.Action = func(ctx *cli.Context) error {
		return runOvnKube(ctx, cancel)
	}

	// trap SIGHUP, SIGINT, SIGTERM, SIGQUIT and
	// cancel the context
	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	defer func() {
		signal.Stop(exitCh)
		cancel()
	}()
	go func() {
		select {
		case s := <-exitCh:
			klog.Infof("Received signal %s. Shutting down", s)
			cancel()
		case <-ctx.Done():
		}
	}()

	if err := c.RunContext(ctx, os.Args); err != nil {
		klog.Exit(err)
	}
}

func delPidfile(pidfile string) {
	if pidfile != "" {
		if _, err := os.Stat(pidfile); err == nil {
			if err := os.Remove(pidfile); err != nil {
				klog.Errorf("%s delete failed: %v", pidfile, err)
			}
		}
	}
}

func setupPIDFile(pidfile string) error {
	// need to test if already there
	_, err := os.Stat(pidfile)

	// Create if it doesn't exist, else exit with error
	if os.IsNotExist(err) {
		if err := ioutil.WriteFile(pidfile, []byte(fmt.Sprintf("%d", os.Getpid())), 0o644); err != nil {
			klog.Errorf("Failed to write pidfile %s (%v). Ignoring..", pidfile, err)
		}
	} else {
		// get the pid and see if it exists
		pid, err := ioutil.ReadFile(pidfile)
		if err != nil {
			return fmt.Errorf("pidfile %s exists but can't be read: %v", pidfile, err)
		}
		_, err1 := os.Stat("/proc/" + string(pid[:]) + "/cmdline")
		if os.IsNotExist(err1) {
			// Left over pid from dead process
			if err := ioutil.WriteFile(pidfile, []byte(fmt.Sprintf("%d", os.Getpid())), 0o644); err != nil {
				klog.Errorf("Failed to write pidfile %s (%v). Ignoring..", pidfile, err)
			}
		} else {
			return fmt.Errorf("pidfile %s exists and ovnkube is running", pidfile)
		}
	}

	return nil
}

// ovnkubeRunMode object stores the run mode of the ovnkube
type ovnkubeRunMode struct {
	networkControllerManager bool // network controller manager (--init-network-controller-manager) is enabled
	clusterManager           bool // cluster manager (--init-cluster-manager) is enabled
	node                     bool // node (--init-node) is enabled
	cleanupNode              bool // cleanup (--cleanup-node) is enabled
}

// determineOvnkubeRunMode determines the run modes of ovnkube
// based on the init flags set.  It is possible to run ovnkube in
// multiple modes.  Allowed multiple modes are:
//   - network controller manager + cluster manager
//   - network controller manager + node
func determineOvnkubeRunMode(ctx *cli.Context) (*ovnkubeRunMode, error) {
	mode := &ovnkubeRunMode{false, false, false, false}

	if ctx.String("init-master") != "" {
		// If init-master is set, then both network controller manager and cluster manager
		// are enabled
		mode.networkControllerManager = true
		mode.clusterManager = true
	}

	if ctx.String("init-cluster-manager") != "" {
		mode.clusterManager = true
	}

	if ctx.String("init-network-controller-manager") != "" {
		mode.networkControllerManager = true
	}

	if ctx.String("init-node") != "" {
		mode.node = true
	}

	if ctx.String("cleanup-node") != "" {
		mode.cleanupNode = true
	}

	if mode.cleanupNode && (mode.clusterManager || mode.networkControllerManager || mode.node) {
		return nil, fmt.Errorf("cannot specify cleanup-node together with 'init-node or 'init-master or init-cluster-manager or init-ovndb-manager'")
	}

	if !mode.clusterManager && !mode.networkControllerManager && !mode.node && !mode.cleanupNode {
		return nil, fmt.Errorf("need to run ovnkube in master mode or cluster manager and/or network controller manager, and/or node mode")
	}

	if !mode.networkControllerManager && mode.clusterManager && mode.node {
		return nil, fmt.Errorf("cannot run in both cluster manager and node mode")
	}

	return mode, nil
}

func getNetworkControllerManagerIdentity(ctx *cli.Context) string {
	identity := ctx.String("init-master")
	if identity == "" {
		identity = ctx.String("init-network-controller-manager")
	}

	return identity
}

func runOvnKube(ctx *cli.Context, cancel context.CancelFunc) error {
	pidfile := ctx.String("pidfile")
	if pidfile != "" {
		defer delPidfile(pidfile)
		if err := setupPIDFile(pidfile); err != nil {
			return err
		}
	}

	exec := kexec.New()
	_, err := config.InitConfig(ctx, exec, nil)
	if err != nil {
		return err
	}

	if err = util.SetExec(exec); err != nil {
		return fmt.Errorf("failed to initialize exec helper: %v", err)
	}

	ovnClientset, err := util.NewOVNClientset(&config.Kubernetes)
	if err != nil {
		return err
	}

	runMode, err := determineOvnkubeRunMode(ctx)

	if err != nil {
		return err
	}

	if runMode.cleanupNode {
		return ovnnode.CleanupClusterNode(ctx.String("cleanup-node"))
	}

	stopChan := make(chan struct{})
	wg := &sync.WaitGroup{}
	defer func() {
		close(stopChan)
		wg.Wait()
	}()
	var masterWatchFactory *factory.WatchFactory
	var masterEventRecorder record.EventRecorder

	if runMode.networkControllerManager {
		var err error
		// create factory and start the controllers asked for
		masterWatchFactory, err = factory.NewMasterWatchFactory(ovnClientset.GetMasterClientset())
		if err != nil {
			return err
		}
		defer masterWatchFactory.Shutdown()
	}

	if runMode.clusterManager {
		var clusterManagerWatchFactory *factory.WatchFactory
		if runMode.networkControllerManager {
			clusterManagerWatchFactory = masterWatchFactory
		} else {
			clusterManagerWatchFactory, err = factory.NewClusterManagerWatchFactory(ovnClientset.GetClusterManagerClientset())
			if err != nil {
				return err
			}
			defer clusterManagerWatchFactory.Shutdown()
		}

		cm := clustermanager.NewClusterManager(ovnClientset.GetClusterManagerClientset(), clusterManagerWatchFactory,
			wg, util.EventRecorder(ovnClientset.KubeClient))
		err = cm.Start(ctx.String("init-network-controller-manager"), ctx.Context)
		defer cm.Stop()
		if err != nil {
			return fmt.Errorf("failed to start cluster manager: %w", err)
		}
	}

	if runMode.networkControllerManager {
		var libovsdbOvnNBClient, libovsdbOvnSBClient libovsdbclient.Client

		if libovsdbOvnNBClient, err = libovsdb.NewNBClient(stopChan); err != nil {
			return fmt.Errorf("error when trying to initialize libovsdb NB client: %v", err)
		}

		if libovsdbOvnSBClient, err = libovsdb.NewSBClient(stopChan); err != nil {
			return fmt.Errorf("error when trying to initialize libovsdb SB client: %v", err)
		}

		// register prometheus metrics that do not depend on becoming ovnkube-master leader
		metrics.RegisterMasterBase()

		masterEventRecorder = util.EventRecorder(ovnClientset.KubeClient)
		cm := controllerManager.NewNetworkControllerManager(ovnClientset, getNetworkControllerManagerIdentity(ctx),
			masterWatchFactory, libovsdbOvnNBClient, libovsdbOvnSBClient, masterEventRecorder, wg)
		err = cm.Start(ctx.Context, cancel)
		defer cm.Stop()
		if err != nil {
			return err
		}
	}

	node := ""
	if runMode.node {
		var nodeWatchFactory factory.NodeWatchFactory
		var nodeEventRecorder record.EventRecorder

		node = ctx.String("init-node")
		if masterWatchFactory == nil {
			var err error
			nodeWatchFactory, err = factory.NewNodeWatchFactory(ovnClientset.GetNodeClientset(), node)
			if err != nil {
				return err
			}
			defer nodeWatchFactory.Shutdown()
		} else {
			nodeWatchFactory = masterWatchFactory
		}

		if masterEventRecorder == nil {
			nodeEventRecorder = util.EventRecorder(ovnClientset.KubeClient)
		} else {
			nodeEventRecorder = masterEventRecorder
		}

		if config.Kubernetes.Token == "" {
			return fmt.Errorf("cannot initialize node without service account 'token'. Please provide one with --k8s-token argument")
		}
		// register ovnkube node specific prometheus metrics exported by the node
		metrics.RegisterNodeMetrics()
		start := time.Now()

		n := ovnnode.NewNode(ovnClientset.KubeClient, nodeWatchFactory, node, stopChan, wg, nodeEventRecorder)
		if err := n.Start(ctx.Context); err != nil {
			return err
		}
		end := time.Since(start)
		metrics.MetricNodeReadyDuration.Set(end.Seconds())
	}

	// now that ovnkube master/node are running, lets expose the metrics HTTP endpoint if configured
	// start the prometheus server to serve OVN K8s Metrics (default master port: 9409, node port: 9410)
	if config.Metrics.BindAddress != "" {
		metrics.StartMetricsServer(config.Metrics.BindAddress, config.Metrics.EnablePprof,
			config.Metrics.NodeServerCert, config.Metrics.NodeServerPrivKey, stopChan, wg)
	}

	// start the prometheus server to serve OVS and OVN Metrics (default port: 9476)
	// Note: for ovnkube node mode dpu-host no metrics is required as ovs/ovn is not running on the node.
	if config.OvnKubeNode.Mode != types.NodeModeDPUHost && config.Metrics.OVNMetricsBindAddress != "" {
		if config.Metrics.ExportOVSMetrics {
			metrics.RegisterOvsMetricsWithOvnMetrics(stopChan)
		}

		if runMode.node {
			metrics.RegisterOvnMetrics(ovnClientset.KubeClient, node, stopChan)
		}
		metrics.StartOVNMetricsServer(config.Metrics.OVNMetricsBindAddress,
			config.Metrics.NodeServerCert, config.Metrics.NodeServerPrivKey, stopChan, wg)
	}

	// run until cancelled
	<-ctx.Context.Done()
	return nil
}
