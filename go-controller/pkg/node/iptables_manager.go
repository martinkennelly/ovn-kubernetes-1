package node

import (
	"fmt"
	"sync"
	"time"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/util/iptables"
	kexec "k8s.io/utils/exec"
)

type chain struct {
	table  iptables.Table
	chain  iptables.Chain
	proto  iptables.Protocol
	delete bool
}

func (c chain) equal(c2 chain) bool {
	if c.table != c2.table {
		return false
	}
	if c.chain != c2.chain {
		return false
	}
	if c.proto != c2.proto {
		return false
	}
	return true
}

// rule represents an iptables entry
type rule struct {
	table  iptables.Table
	chain  iptables.Chain
	proto  iptables.Protocol
	args   []string
	delete bool
}

func (r rule) equal(r2 rule) bool {
	if r.table != r2.table {
		return false
	}
	if r.chain != r2.chain {
		return false
	}
	if r.proto != r2.proto {
		return false
	}
	if len(r.args) != len(r2.args) {
		return false
	}
	// ensure all args for r are found in rule2 args
	for _, ruleArg := range r.args {
		var found bool
		for _, rule2Arg := range r2.args {
			if ruleArg == rule2Arg {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	// ensure all args for rule2 are found in r args
	for _, rule2Arg := range r2.args {
		var found bool
		for _, ruleArg := range r.args {
			if rule2Arg == ruleArg {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// iptablesManager manages iptables for clients
type iptablesManager struct {
	mu     *sync.Mutex
	chains []chain
	rules  []rule
	iptV4  iptables.Interface
	iptV6  iptables.Interface
	v4     bool
	v6     bool
}

func newIPTablesManager(v4, v6 bool) *iptablesManager {
	return &iptablesManager{
		chains: make([]chain, 0),
		rules:  make([]rule, 0),
		mu:     &sync.Mutex{},
		iptV4:  iptables.New(kexec.New(), iptables.ProtocolIPv4),
		iptV6:  iptables.New(kexec.New(), iptables.ProtocolIPv6),
		v4:     v4,
		v6:     v6,
	}
}

func (iptm *iptablesManager) run(stopCh chan struct{}, wait sync.WaitGroup) {
	go func() {
		defer wait.Done()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				iptm.mu.Lock()
				if err := iptm.reconcile(); err != nil {
					klog.Errorf("IP tables manager: failed to reconcile: %v", err)
				}
				iptm.mu.Unlock()
			}
		}
	}()
}

// reconcile configures IP tables to make the state in rules. CPU starvation or iptables lock held by an external
// entity may cause this function to take some time to execute.
func (iptm *iptablesManager) reconcile() error {
	start := time.Now()
	defer func() {
		klog.V(4).Infof("reconciling IP tables rules took %v", time.Since(start))
	}()
	var errors []error
	tempChains := iptm.chains[:0]
	for _, c := range iptm.chains {
		if c.delete {
			if err := execIPTablesWithRetry(func() error {
				if c.proto == iptables.ProtocolIPv4 {
					return iptm.iptV4.DeleteChain(c.table, c.chain)
				} else {
					return iptm.iptV6.DeleteChain(c.table, c.chain)
				}
			}); err != nil {
				return fmt.Errorf("failed to delete chain %v: %v", c, err)
			}
		} else {
			// ensure chains
			tempChains = append(tempChains, c)
			err := execIPTablesWithRetry(func() error {
				var err error
				if c.proto == iptables.ProtocolIPv4 {
					_, err = iptm.iptV4.EnsureChain(c.table, c.chain)
				} else if c.proto == iptables.ProtocolIPv6 {
					_, err = iptm.iptV6.EnsureChain(c.table, c.chain)
				}
				return err
			})
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to ensure chain %v exists: %v", c, err))
			}
		}
	}
	iptm.chains = tempChains
	// in-order to avoid creating a new underlying array, use the existing one. We can copy rules we want to persist here
	// and ignore rules we want to delete
	tempRules := iptm.rules[:0]
	for _, r := range iptm.rules {
		// cleanup rules that are marked for deletion
		if r.delete {
			if err := execIPTablesWithRetry(func() error {
				if r.proto == iptables.ProtocolIPv4 {
					return iptm.iptV4.DeleteRule(r.table, r.chain, r.args...)
				} else {
					return iptm.iptV6.DeleteRule(r.table, r.chain, r.args...)
				}
			}); err != nil {
				return fmt.Errorf("failed to delete rule: %v", err)
			}
		} else {
			// ensure rules
			tempRules = append(tempRules, r)
			err := execIPTablesWithRetry(func() error {
				var err error
				if r.proto == iptables.ProtocolIPv4 {
					_, err = iptm.iptV4.EnsureRule(iptables.Append, r.table, r.chain, r.args...)
				} else if r.proto == iptables.ProtocolIPv6 {
					_, err = iptm.iptV6.EnsureRule(iptables.Append, r.table, r.chain, r.args...)
				}
				return err
			})
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to ensure rule %v exists: %v", r, err))
			}
		}
	}
	iptm.rules = tempRules
	return utilerrors.NewAggregate(errors)
}

func (iptm *iptablesManager) ensureChain(table iptables.Table, c iptables.Chain, proto iptables.Protocol) error {
	iptm.mu.Lock()
	defer iptm.mu.Unlock()

	newChain := chain{
		table: table,
		chain: c,
		proto: proto,
	}
	// do nothing if it already exists
	for _, existingChain := range iptm.chains {
		if existingChain.equal(newChain) {
			return nil
		}
	}
	iptm.chains = append(iptm.chains, newChain)
	return iptm.reconcile()
}

func (iptm *iptablesManager) ensureRule(table iptables.Table, c iptables.Chain, proto iptables.Protocol, args ...string) error {
	iptm.mu.Lock()
	defer iptm.mu.Unlock()
	newRule := rule{
		table: table,
		chain: c,
		proto: proto,
		args:  args,
	}
	// do nothing if it already exists
	for _, existingRule := range iptm.rules {
		if existingRule.equal(newRule) {
			return nil
		}
	}
	iptm.rules = append(iptm.rules)
	return iptm.reconcile()
}

func (iptm *iptablesManager) deleteRule(table iptables.Table, chain iptables.Chain, proto iptables.Protocol, args ...string) error {
	iptm.mu.Lock()
	defer iptm.mu.Unlock()
	deleteRule := rule{
		table: table,
		chain: chain,
		proto: proto,
		args:  args,
	}
	var reconcileNeeded bool
	for i, existingRule := range iptm.rules {
		if existingRule.equal(deleteRule) {
			iptm.rules[i].delete = true
			reconcileNeeded = true
			break
		}
	}
	if reconcileNeeded {
		return iptm.reconcile()
	}
	return nil
}

// iptablesBackoff will retry 10 times over a period of 13 seconds
var iptablesBackoff wait.Backoff = wait.Backoff{
	Duration: 500 * time.Millisecond,
	Factor:   1.25,
	Steps:    10,
}

// execIPTablesWithRetry allows a simple way to retry iptables commands if they fail the first time
func execIPTablesWithRetry(f func() error) error {
	return wait.ExponentialBackoff(iptablesBackoff, func() (bool, error) {
		if err := f(); err != nil {
			if isResourceError(err) {
				klog.V(5).Infof("Call to iptables failed with transient failure: %v", err)
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

const iptablesStatusResourceProblem = 4

// isResourceError returns true if the error indicates that iptables ran into a "resource
// problem" and was unable to attempt the request. In particular, this will be true if it
// times out trying to get the iptables lock.
func isResourceError(err error) bool {
	if ee, isExitError := err.(kexec.ExitError); isExitError {
		return ee.ExitStatus() == iptablesStatusResourceProblem
	}
	return false
}
