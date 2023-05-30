package interface_manager

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

type Chain struct {
	Table  iptables.Table
	Chain  iptables.Chain
	Proto  iptables.Protocol
	delete bool
}

func (c Chain) equal(c2 Chain) bool {
	if c.Table != c2.Table {
		return false
	}
	if c.Chain != c2.Chain {
		return false
	}
	if c.Proto != c2.Proto {
		return false
	}
	return true
}

// Rule represents an iptables entry
type Rule struct {
	table  iptables.Table
	chain  iptables.Chain
	proto  iptables.Protocol
	args   []string
	delete bool
}

func (r Rule) equal(r2 Rule) bool {
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

// Controller manages iptables for clients
type Controller struct {
	mu     *sync.Mutex
	chains []Chain
	rules  []Rule
	iptV4  iptables.Interface
	iptV6  iptables.Interface
	v4     bool
	v6     bool
}

func NewController(v4, v6 bool) *Controller {
	return &Controller{
		chains: make([]Chain, 0),
		rules:  make([]Rule, 0),
		mu:     &sync.Mutex{},
		iptV4:  iptables.New(kexec.New(), iptables.ProtocolIPv4),
		iptV6:  iptables.New(kexec.New(), iptables.ProtocolIPv6),
		v4:     v4,
		v6:     v6,
	}
}

func (c *Controller) Run(stopCh <-chan struct{}, wait *sync.WaitGroup) {
	go func() {
		defer wait.Done()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				c.mu.Lock()
				if err := c.reconcile(); err != nil {
					klog.Errorf("IP tables manager: failed to reconcile: %v", err)
				}
				c.mu.Unlock()
			}
		}
	}()
}

// reconcile configures IP tables to make the state in rules. CPU starvation or iptables lock held by an external
// entity may cause this function to take some time to execute.
func (c *Controller) reconcile() error {
	start := time.Now()
	defer func() {
		klog.V(4).Infof("reconciling IP tables rules took %v", time.Since(start))
	}()
	var errors []error
	tempChains := c.chains[:0]
	for _, chain := range c.chains {
		if chain.delete {
			if err := execIPTablesWithRetry(func() error {
				if chain.Proto == iptables.ProtocolIPv4 {
					return c.iptV4.DeleteChain(chain.Table, chain.Chain)
				} else {
					return c.iptV6.DeleteChain(chain.Table, chain.Chain)
				}
			}); err != nil {
				return fmt.Errorf("failed to delete chain %v: %v", c, err)
			}
		} else {
			// ensure chains
			tempChains = append(tempChains, c)
			err := execIPTablesWithRetry(func() error {
				var err error
				if chain.Proto == iptables.ProtocolIPv4 {
					_, err = c.iptV4.EnsureChain(chain.Table, chain.Chain)
				} else if chain.Proto == iptables.ProtocolIPv6 {
					_, err = c.iptV6.EnsureChain(chain.Table, chain.Chain)
				}
				return err
			})
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to ensure chain %v exists: %v", c, err))
			}
		}
	}
	c.chains = tempChains
	// in-order to avoid creating a new underlying array, use the existing one. We can copy rules we want to persist here
	// and ignore rules we want to delete
	tempRules := c.rules[:0]
	for _, r := range c.rules {
		// cleanup rules that are marked for deletion
		if r.delete {
			if err := execIPTablesWithRetry(func() error {
				if r.proto == iptables.ProtocolIPv4 {
					return c.iptV4.DeleteRule(r.table, r.chain, r.args...)
				} else {
					return c.iptV6.DeleteRule(r.table, r.chain, r.args...)
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
					_, err = c.iptV4.EnsureRule(iptables.Append, r.table, r.chain, r.args...)
				} else if r.proto == iptables.ProtocolIPv6 {
					_, err = c.iptV6.EnsureRule(iptables.Append, r.table, r.chain, r.args...)
				}
				return err
			})
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to ensure rule %v exists: %v", r, err))
			}
		}
	}
	c.rules = tempRules
	return utilerrors.NewAggregate(errors)
}

func (c *Controller) ensureChain(table iptables.Table, chain iptables.Chain, proto iptables.Protocol) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	newChain := Chain{
		Table: table,
		Chain: chain,
		Proto: proto,
	}
	// do nothing if it already exists
	for _, existingChain := range c.chains {
		if existingChain.equal(newChain) {
			return nil
		}
	}
	c.chains = append(c.chains, newChain)
	return c.reconcile()
}

func (c *Controller) ensureRule(table iptables.Table, chain iptables.Chain, proto iptables.Protocol, args ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	newRule := Rule{
		table: table,
		chain: chain,
		proto: proto,
		args:  args,
	}
	// do nothing if it already exists
	for _, existingRule := range c.rules {
		if existingRule.equal(newRule) {
			return nil
		}
	}
	c.rules = append(c.rules)
	return c.reconcile()
}

func (c *Controller) deleteRule(table iptables.Table, chain iptables.Chain, proto iptables.Protocol, args ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	deleteRule := Rule{
		table: table,
		chain: chain,
		proto: proto,
		args:  args,
	}
	var reconcileNeeded bool
	for i, existingRule := range c.rules {
		if existingRule.equal(deleteRule) {
			c.rules[i].delete = true
			reconcileNeeded = true
			break
		}
	}
	if reconcileNeeded {
		return c.reconcile()
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
