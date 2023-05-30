package iptables

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/util/iptables"
	kexec "k8s.io/utils/exec"
)

type Chain struct {
	Table iptables.Table
	Chain iptables.Chain
	Proto iptables.Protocol
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

type matchRule struct {
	rules []RuleArgs
	match *regexp.Regexp
}

// RuleArgs represents an iptables entry
type RuleArgs struct {
	Args []string
}

func (r RuleArgs) equal(r2 RuleArgs) bool {
	var found bool
	if len(r.Args) != len(r2.Args) {
		return false
	}
	// ensure all args for r are found in rule2 args
	for _, ruleArg := range r.Args {
		found = false
		for _, rule2Arg := range r2.Args {
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
	for _, rule2Arg := range r2.Args {
		found = false
		for _, ruleArg := range r.Args {
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

// Controller manages iptables for clients for NAT table only
type Controller struct {
	mu         *sync.Mutex
	chainRules map[Chain]matchRule
	iptV4      iptables.Interface
	iptV6      iptables.Interface
	v4         bool
	v6         bool
}

// NewController creates a controller to manage chains and rules for the NAT table only
func NewController(v4, v6 bool) *Controller {
	return &Controller{
		chainRules: make(map[Chain]matchRule, 0),
		mu:         &sync.Mutex{},
		iptV4:      iptables.New(kexec.New(), iptables.ProtocolIPv4),
		iptV6:      iptables.New(kexec.New(), iptables.ProtocolIPv6),
		v4:         v4,
		v6:         v6,
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
				c.reconcile()
				c.mu.Unlock()
			}
		}
	}()
}

// findIPInRulesInTableUsingRegex must be passed a regex that matches one IPv4 or IPv6 address which is added to a map as a key
// Chain will be selected with the associated regex match
func (c *Controller) findIPInRulesInTableUsingRegex(ipt iptables.Interface, table iptables.Table, ruleMatch *regexp.Regexp) ([]RuleArgs, error) {
	buf := bytes.NewBuffer(nil)
	if err := ipt.SaveInto(table, buf); err != nil {
		return nil, fmt.Errorf("failed to retrieve iptables table %s: %v", table, err)
	}
	rules := make([]RuleArgs, 0)
	for _, line := range strings.Split(string(buf.Bytes()), "\n") {
		match := ruleMatch.FindStringSubmatch(line)
		if len(match) != 3 {
			continue
		}
		rules = append(rules, RuleArgs{
			Args: []string{match[1], match[2], match[3]},
		})
	}
	return rules, nil
}

func processRules(ipt iptables.Interface, wantedChainRules map[Chain]matchRule, existingChainRules map[Chain][]RuleArgs) error {
	var err error
	for existingChain, existingRules := range existingChainRules {
		if existingChain.Proto != ipt.Protocol() {
			continue
		}
		wantedMR, found := wantedChainRules[existingChain]
		if !found {
			// we don't delete chains we don't manage, but we ensure rules are removed
			continue
		}
		err = execIPTablesWithRetry(func() error {
			_, err = ipt.EnsureChain(existingChain.Table, existingChain.Chain)
			return err
		})
		if err != nil {
			klog.Errorf("Failed to ensure chain %s in table %s: %v", existingChain.Chain, existingChain.Table, err)
		}
		// clean up rules which we don't care about in the chain
		for _, existingRule := range existingRules {
			found = false
			for _, wantedRule := range wantedMR.rules {
				if wantedRule.equal(existingRule) {
					found = true
					break
				}
			}
			if !found {
				err = execIPTablesWithRetry(func() error {
					return ipt.DeleteRule(existingChain.Table, existingChain.Chain, existingRule.Args...)
				})
				if err != nil {
					klog.Errorf("Failed to delete stale rule (%+v) in table %s and chain %s: %v", existingRule.Args,
						existingChain.Table, existingChain.Chain, err)
				}
			}
		}
		// ensure all rules we want are present
		for _, wantedRule := range wantedMR.rules {
			err = execIPTablesWithRetry(func() error {
				_, err = ipt.EnsureRule(iptables.Prepend, existingChain.Table, existingChain.Chain, wantedRule.Args...)
				return err
			})
			if err != nil {
				klog.Errorf("Failed to ensure rule (%+v) in table %s and chain %s: %v", wantedRule.Args,
					existingChain.Table, existingChain.Chain)
			}
		}
	}
	return nil
}

// reconcile configures IP tables to ensure the correct chains and rules within the NAT table.
// CPU starvation or iptables lock held by an external entity may cause this function to take some time to execute.
func (c *Controller) reconcile() {
	start := time.Now()
	defer func() {
		klog.V(4).Infof("reconciling IP tables rules took %v", time.Since(start))
	}()

	existingChainRules := make(map[Chain][]RuleArgs)
	for chain, mr := range c.chainRules {
		// -A POSTROUTING -s 10.244.2.4/32 -o dummy0 -j SNAT --to-source 1.1.1.1
		// change below to match above
		//"-A %s -s ([^ ]*) -o ([^ ]*) .* --to ([^ ]*)",
		if c.v4 && chain.Proto == iptables.ProtocolIPv4 {
			rules, err := c.findIPInRulesInTableUsingRegex(c.iptV4, chain.Table, mr.match)
			if err != nil {
				klog.Errorf("Failed to find rules for chain %s in table %s", chain.Chain, chain.Table)
			} else {
				existingChainRules[chain] = rules
			}
		} else if c.v6 && chain.Proto == iptables.ProtocolIPv6 {
			rules, err := c.findIPInRulesInTableUsingRegex(c.iptV6, chain.Table, mr.match)
			if err != nil {
				klog.Errorf("Failed to find rules for chain %s in table %s", chain.Chain, chain.Table)
			} else {
				existingChainRules[chain] = rules
			}
		}
	}

	if c.v4 {
		if err := processRules(c.iptV4, c.chainRules, existingChainRules); err != nil {
			klog.Errorf("Failed to process IPv4 rules: %v", err)
		}
	}
	if c.v6 {
		if err := processRules(c.iptV6, c.chainRules, existingChainRules); err != nil {
			klog.Errorf("Failed to process IPv6 rules: %v", err)
		}
	}
}

// OwnChain ensures this chain exists and any rules within it this component exclusively owns
func (c *Controller) OwnChain(table iptables.Table, chain iptables.Chain, proto iptables.Protocol) {
	c.mu.Lock()
	defer c.mu.Unlock()
	newChain := Chain{
		Table: table,
		Chain: chain,
		Proto: proto,
	}
	if _, found := c.chainRules[newChain]; !found {
		c.chainRules[newChain] = matchRule{}
	}
	c.reconcile()
}

// EnsureRulesOwnChain ensures only the subset of rules specified in the function description will be present on the chain
func (c *Controller) EnsureRulesOwnChain(table iptables.Table, chain iptables.Chain, proto iptables.Protocol, rules []RuleArgs, match string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	newChain := Chain{
		Table: table,
		Chain: chain,
		Proto: proto,
	}
	c.chainRules[newChain] = matchRule{rules: rules, match: regexp.MustCompile(match)}
	c.reconcile()
}

// iptablesBackoff will retry 10 times over a period of 13 seconds
var iptablesBackoff = wait.Backoff{
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
