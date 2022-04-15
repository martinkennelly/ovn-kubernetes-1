//go:build !race

package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/ovn-org/libovsdb/client"
	"github.com/ovn-org/libovsdb/model"
	"github.com/ovn-org/libovsdb/ovsdb"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/kube"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/libovsdbops"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/nbdb"
	libovsdbtest "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/testing/libovsdb"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientgo "k8s.io/client-go/kubernetes/fake"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

type fakeHistogramInterface interface {
	prometheus.Metric
	prometheus.Collector
	Observe(float64)
}

type fakeHistogram struct {
	prometheus.Metric
	prometheus.Collector
	observeCalledFunc func(v float64)
}

func (f fakeHistogram) Observe(v float64) {
	f.observeCalledFunc(v)
}

func getHistoMock() (fakeHistogramInterface, *int, *float64) {
	f := fakeHistogram{}
	var numberOfTimesCalled int
	var totalAmountAdded float64
	f.observeCalledFunc = func(v float64) {
		numberOfTimesCalled += 1
		totalAmountAdded += v
	}
	return f, &numberOfTimesCalled, &totalAmountAdded
}

func setupOvn(nbData libovsdbtest.TestSetup) (client.Client, client.Client, *libovsdbtest.Cleanup) {
	nbClient, sbClient, cleanup, err := libovsdbtest.NewNBSBTestHarness(nbData)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return sbClient, nbClient, cleanup
}

func getKubeClient(nodeCount int) kube.Kube {
	var nodes []corev1.Node
	for i := 0; i < nodeCount; i++ {
		nodes = append(nodes, corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("nodeName-%d", i),
			},
		})
	}
	kubeFakeClient := fakeclientgo.NewSimpleClientset(&corev1.NodeList{Items: nodes})
	fakeClient := &util.OVNClientset{
		KubeClient: kubeFakeClient,
	}
	return kube.Kube{fakeClient.KubeClient, nil, nil, nil}
}

func setHvCfg(nbClient client.Client, hvCfg int, hvCfgTimestamp time.Time) {
	nbGlobal, err := libovsdbops.FindNBGlobal(nbClient)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	ops, err := nbClient.Where(nbGlobal).Mutate(nbGlobal, model.Mutation{
		Field:   &nbGlobal.HvCfg,
		Mutator: ovsdb.MutateOperationAdd,
		Value:   hvCfg,
	},
		model.Mutation{
			Field:   &nbGlobal.HvCfgTimestamp,
			Mutator: ovsdb.MutateOperationAdd,
			Value:   int(hvCfgTimestamp.UnixMilli()),
		})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(ops).To(gomega.HaveLen(1))
	_, err = libovsdbops.TransactAndCheck(nbClient, ops)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

var _ = ginkgo.Describe("Config duration recorder operations", func() {
	ginkgo.Context("recordConfigDuration()", func() {
		ginkgo.It("should record a result for kind metric", func() {
			_, nbClient, cleanup := setupOvn(libovsdbtest.TestSetup{
				NBData: []libovsdbtest.TestData{&nbdb.NBGlobal{UUID: "cd-op-uuid"}}})
			defer cleanup.Cleanup()
			// setup
			k := getKubeClient(1)
			cpr := GetConfigDurationRecorder()
			stop := make(chan struct{})
			defer close(stop)
			cpr.Run(nbClient, &k, time.Second, time.Hour, stop)
			startTime := time.Now()
			endTime := startTime.Add(time.Second)
			// start recording
			ops, txOkCallback, err := cpr.RecordConfigDuration(nbClient, PodKindName, startTime)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(ops).To(gomega.HaveLen(1))
			_, err = libovsdbops.TransactAndCheck(nbClient, ops)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			txOkCallback()
			fakeHist, numberOfTimesCalled, _ := getHistoMock()
			cpr.reportKind[PodKindName] = fakeHist
			setHvCfg(nbClient, 1, endTime)

			gomega.Eventually(func() bool {
				// expecting called once
				return *numberOfTimesCalled == 1
			}).Should(gomega.BeTrue())
		})

		ginkgo.It("should record correct time delta for kind metric", func() {
			_, nbClient, cleanup := setupOvn(libovsdbtest.TestSetup{
				NBData: []libovsdbtest.TestData{&nbdb.NBGlobal{UUID: "cd-op-uuid"}}})
			defer cleanup.Cleanup()
			// setup
			k := getKubeClient(1)
			cpr := GetConfigDurationRecorder()
			stop := make(chan struct{})
			defer close(stop)
			cpr.Run(nbClient, &k, time.Second, time.Hour, stop)
			startTime := time.Now()
			endTime := startTime.Add(time.Second)
			// start recording
			ops, txOkCallback, err := cpr.RecordConfigDuration(nbClient, PodKindName, startTime)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(ops).To(gomega.HaveLen(1))
			_, err = libovsdbops.TransactAndCheck(nbClient, ops)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			txOkCallback()
			fakeHist, _, totalAmountAdded := getHistoMock()
			cpr.reportKind[PodKindName] = fakeHist
			setHvCfg(nbClient, 1, endTime)

			gomega.Eventually(func() bool {
				// expecting correct time delta
				return *totalAmountAdded == 1
			}).Should(gomega.BeTrue())
		})

		ginkgo.It("should record a result for ovn metric", func() {
			_, nbClient, cleanup := setupOvn(libovsdbtest.TestSetup{
				NBData: []libovsdbtest.TestData{&nbdb.NBGlobal{UUID: "cd-op-uuid"}}})
			defer cleanup.Cleanup()
			// setup
			k := getKubeClient(1)
			cpr := GetConfigDurationRecorder()
			stop := make(chan struct{})
			defer close(stop)
			cpr.Run(nbClient, &k, time.Second, time.Hour, stop)
			startTime := time.Now()
			endTime := startTime.Add(time.Second)
			// start recording
			ops, txOkCallback, err := cpr.RecordConfigDuration(nbClient, PodKindName, startTime)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(ops).To(gomega.HaveLen(1))
			_, err = libovsdbops.TransactAndCheck(nbClient, ops)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			txOkCallback()
			fakeHist, numberOfTimesCalled, _ := getHistoMock()
			metricConfigDurationOvnLatency = fakeHist
			setHvCfg(nbClient, 1, endTime)

			gomega.Eventually(func() bool {
				// expecting called once
				return *numberOfTimesCalled == 1
			}).Should(gomega.BeTrue())
		})

		ginkgo.It("should record non-zero time delta for ovn metric", func() {
			_, nbClient, cleanup := setupOvn(libovsdbtest.TestSetup{
				NBData: []libovsdbtest.TestData{&nbdb.NBGlobal{UUID: "cd-op-uuid"}}})
			defer cleanup.Cleanup()
			// setup
			k := getKubeClient(1)
			cpr := GetConfigDurationRecorder()
			stop := make(chan struct{})
			defer close(stop)
			cpr.Run(nbClient, &k, time.Second, time.Hour, stop)
			startTime := time.Now()
			endTime := startTime.Add(time.Second)
			// start recording
			ops, txOkCallback, err := cpr.RecordConfigDuration(nbClient, PodKindName, startTime)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(ops).To(gomega.HaveLen(1))
			_, err = libovsdbops.TransactAndCheck(nbClient, ops)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			txOkCallback()
			fakeHist, _, totalAmountAdded := getHistoMock()
			metricConfigDurationOvnLatency = fakeHist
			setHvCfg(nbClient, 1, endTime)

			gomega.Eventually(func() bool {
				// expecting non-zero time added
				return *totalAmountAdded > 0
			}).Should(gomega.BeTrue())
		})

		ginkgo.It("should honor multiple sequential recordings", func() {
			_, nbClient, cleanup := setupOvn(libovsdbtest.TestSetup{
				NBData: []libovsdbtest.TestData{&nbdb.NBGlobal{UUID: "cd-op-uuid"}}})
			defer cleanup.Cleanup()
			// setup
			k := getKubeClient(1)
			cpr := GetConfigDurationRecorder()
			stop := make(chan struct{})
			defer close(stop)
			cpr.Run(nbClient, &k, time.Millisecond, time.Hour, stop)
			startTime := time.Now()
			endTime := startTime.Add(time.Second)
			// start multiple recordings
			ops, txOkCallback, err := cpr.RecordConfigDuration(nbClient, PodKindName, startTime)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(ops).To(gomega.HaveLen(1))
			_, err = libovsdbops.TransactAndCheck(nbClient, ops)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			txOkCallback()
			gomega.Eventually(func() int {
				ops, txOkCallback, err = cpr.RecordConfigDuration(nbClient, PodKindName, startTime)
				return len(ops)
			}).Should(gomega.BeNumerically(">", 0))
			_, err = libovsdbops.TransactAndCheck(nbClient, ops)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			txOkCallback()
			fakeHist, numberOfTimesCalled, totalAmountAdded := getHistoMock()
			cpr.reportKind[PodKindName] = fakeHist
			setHvCfg(nbClient, 2, endTime)

			gomega.Eventually(func() bool {
				// expecting correct time delta - two items added each with a delta of 1 seconds
				return *totalAmountAdded == 2 && *numberOfTimesCalled == 2
			}).Should(gomega.BeTrue())
		})

		ginkgo.It("should honor multiple parallel recordings", func() {
			_, nbClient, cleanup := setupOvn(libovsdbtest.TestSetup{
				NBData: []libovsdbtest.TestData{&nbdb.NBGlobal{UUID: "cd-op-uuid"}}})
			defer cleanup.Cleanup()
			// setup
			k := getKubeClient(1)
			cpr := GetConfigDurationRecorder()
			stop := make(chan struct{})
			defer close(stop)
			cpr.Run(nbClient, &k, time.Millisecond, time.Hour, stop)
			startTime1 := time.Now()
			// start parallel recordings
			ops1, txOkCallback1, err1 := cpr.RecordConfigDuration(nbClient, PodKindName, startTime1)
			gomega.Expect(err1).NotTo(gomega.HaveOccurred())
			gomega.Expect(ops1).To(gomega.HaveLen(1))
			// end time is 10 secs greater than startTime 1 and 9 secs greater than startTime 2
			startTime2 := startTime1.Add(1 * time.Second)
			endTime := startTime1.Add(10 * time.Second)
			var ops2 []ovsdb.Operation
			var err2 error
			var txOkCallback2 func()
			gomega.Eventually(func() int {
				ops2, txOkCallback2, err2 = cpr.RecordConfigDuration(nbClient, PodKindName, startTime2)
				return len(ops2)
			}).Should(gomega.BeNumerically(">", 0))

			_, err2 = libovsdbops.TransactAndCheck(nbClient, ops2)
			gomega.Expect(err2).NotTo(gomega.HaveOccurred())
			txOkCallback2()

			_, err1 = libovsdbops.TransactAndCheck(nbClient, ops1)
			gomega.Expect(err1).NotTo(gomega.HaveOccurred())
			txOkCallback1()

			fakeHist, numberOfTimesCalled, totalAmountAdded := getHistoMock()
			cpr.reportKind[PodKindName] = fakeHist
			setHvCfg(nbClient, 2, endTime)

			gomega.Eventually(func() bool {
				// expecting correct time delta which is 9s + 10s
				return *totalAmountAdded == 19 && *numberOfTimesCalled == 2
			}).Should(gomega.BeTrue())
		})

		ginkgo.It("should not record when node count is over limit from the beginning", func() {
			_, nbClient, cleanup := setupOvn(libovsdbtest.TestSetup{
				NBData: []libovsdbtest.TestData{&nbdb.NBGlobal{UUID: "cd-op-uuid"}}})
			defer cleanup.Cleanup()
			// setup
			k := getKubeClient(nodeLimit + 1)
			cpr := GetConfigDurationRecorder()
			stop := make(chan struct{})
			defer close(stop)
			cpr.Run(nbClient, &k, time.Second, time.Hour, stop)
			startTime := time.Now()
			// start recording
			ops, _, err := cpr.RecordConfigDuration(nbClient, PodKindName, startTime)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// expect no ops because functionality is disabled
			gomega.Expect(ops).To(gomega.HaveLen(0))
		})

		ginkgo.It("should not record when node count transitions to over node limit during runtime", func() {
			_, nbClient, cleanup := setupOvn(libovsdbtest.TestSetup{
				NBData: []libovsdbtest.TestData{&nbdb.NBGlobal{UUID: "cd-op-uuid"}}})
			defer cleanup.Cleanup()
			// setup fake client with less than limit
			k := getKubeClient(nodeLimit - 1)
			cpr := GetConfigDurationRecorder()
			stop := make(chan struct{})
			defer close(stop)
			// very short duration for check period, therefore, we can observe changed in enable
			cpr.Run(nbClient, &k, time.Second, 1*time.Millisecond, stop)
			gomega.Expect(cpr.enabled).Should(gomega.BeTrue())
			// increase node count above node limit
			_, err := k.KClient.CoreV1().Nodes().Create(context.TODO(), &corev1.Node{ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("nodeName-over-limit")}}, metav1.CreateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			nodes, err := k.KClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(nodes.Items).Should(gomega.HaveLen(nodeLimit))

			// testing internal variable as no reliable method otherwise
			gomega.Eventually(func() bool { return cpr.enabled }).Should(gomega.BeFalse())
		})

		ginkgo.It("should record when node count is over node limit but then transitions under node limit during runtime", func() {
			_, nbClient, cleanup := setupOvn(libovsdbtest.TestSetup{
				NBData: []libovsdbtest.TestData{&nbdb.NBGlobal{UUID: "cd-op-uuid"}}})
			defer cleanup.Cleanup()
			// setup fake client at node limit
			k := getKubeClient(nodeLimit)
			cpr := GetConfigDurationRecorder()
			stop := make(chan struct{})
			defer close(stop)
			// very short duration for check period, therefore, we can observe changed in enable
			cpr.Run(nbClient, &k, time.Second, 1*time.Millisecond, stop)
			gomega.Expect(cpr.enabled).Should(gomega.BeFalse())
			// decrease node count below node limit
			err := k.KClient.CoreV1().Nodes().Delete(context.TODO(), "nodeName-0", metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			nodes, err := k.KClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(nodes.Items).Should(gomega.HaveLen(nodeLimit - 1))

			// testing internal variable as no reliable method otherwise
			gomega.Eventually(func() bool { return cpr.enabled }).Should(gomega.BeTrue())
		})
	})
})
