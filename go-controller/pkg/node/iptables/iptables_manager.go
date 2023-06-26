package iptables

import (
	"bytes"
	"fmt"
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
	chainRules map[Chain][]RuleArgs
	chains     []Chain
	iptV4      iptables.Interface
	iptV6      iptables.Interface
	v4         bool
	v6         bool
}

// NewController creates a controller to manage chains and rules for the NAT table only
func NewController(v4, v6 bool) *Controller {
	var iptV4, iptV6 iptables.Interface
	if v4 {
		iptV4 = iptables.New(kexec.New(), iptables.ProtocolIPv4)
	}
	if v6 {
		iptV6 = iptables.New(kexec.New(), iptables.ProtocolIPv6)
	}
	return &Controller{
		chainRules: make(map[Chain][]RuleArgs, 0),
		chains:     make([]Chain, 0),
		mu:         &sync.Mutex{},
		iptV4:      iptV4,
		iptV6:      iptV6,
		v4:         v4,
		v6:         v6,
	}
}

func (c *Controller) Run(stopCh <-chan struct{}, syncPeriod time.Duration) {
	var err error
	ticker := time.NewTicker(syncPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			c.mu.Lock()
			if err = c.reconcile(); err != nil {
				klog.Errorf("IPTables manager failed to reconcile (will be retried in %s): %v", syncPeriod.String(), err)
			}
			c.mu.Unlock()
		}
	}
}

// OwnChain ensures this chain exists and any rules within it this component exclusively owns
func (c *Controller) OwnChain(table iptables.Table, chain iptables.Chain, proto iptables.Protocol) error {
	klog.Errorf("## calling own chain for chain %s", chain)
	c.mu.Lock()
	defer c.mu.Unlock()
	newChain := Chain{
		Table: table,
		Chain: chain,
		Proto: proto,
	}
	var alreadyExists bool
	for _, existingChain := range c.chains {
		if existingChain.equal(newChain) {
			alreadyExists = true
		}
	}
	if !alreadyExists {
		c.chains = append(c.chains, newChain)
	}
	return c.reconcile()
}

func (c *Controller) DeleteRule(table iptables.Table, chain iptables.Chain, proto iptables.Protocol, ruleArg RuleArgs) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error
	if proto == iptables.ProtocolIPv4 {
		err = execIPTablesWithRetry(func() error {
			err = c.iptV4.DeleteRule(table, chain, ruleArg.Args...)
			if err != nil {
				klog.Errorf("###err while deleting rule %v chain %v", ruleArg.Args, err)
			}
			return err
		})
		if err != nil {
			return fmt.Errorf("failed to delete IPv4 rule %v on table %s and chain %s: %v", ruleArg.Args, table, chain, err)
		}
	} else {
		err = execIPTablesWithRetry(func() error {
			err = c.iptV6.DeleteRule(table, chain, ruleArg.Args...)
			if err != nil {
				klog.Errorf("###err while deleting rule %v chain %v", ruleArg.Args, err)
			}
			return err
		})
		if err != nil {
			return fmt.Errorf("failed to IPv6 delete rule %v on table %s and chain %s: %v", ruleArg.Args, table, chain, err)
		}
	}
	ch := Chain{
		Table: table,
		Chain: chain,
		Proto: proto,
	}
	temp := make([]RuleArgs, 0)
	for _, existingRuleArg := range c.chainRules[ch] {
		klog.Errorf("### Deleterule() checking if %v is equal to %v", existingRuleArg.Args, ruleArg.Args)
		if !existingRuleArg.equal(ruleArg) {
			klog.Errorf("### DeleteRule() its not equal!!")
			temp = append(temp, existingRuleArg)
		}
	}
	klog.Errorf("## DeleteRule() after processing tmp is %v", temp)
	c.chainRules[ch] = temp
	return nil
}

// EnsureRules ensures only the subset of rules specified in the function description will be present on the chain
func (c *Controller) EnsureRules(table iptables.Table, chain iptables.Chain, proto iptables.Protocol, newRuleArgs []RuleArgs) error {
	klog.Errorf("### Ensuring new rules: %+v", newRuleArgs)
	c.mu.Lock()
	defer c.mu.Unlock()
	newChain := Chain{
		Table: table,
		Chain: chain,
		Proto: proto,
	}
	wantedRuleArgs, alreadyExists := c.chainRules[newChain]
	if !alreadyExists {
		c.chainRules[newChain] = newRuleArgs
	} else {
		for _, newRuleArg := range newRuleArgs {
			alreadyExists = false

			for _, existingRuleArg := range wantedRuleArgs {
				klog.Errorf("### Ensurerules() checking if existing rule %v is equal to new rule %v", existingRuleArg.Args, newRuleArg.Args)
				if existingRuleArg.equal(newRuleArg) {
					klog.Errorf("###### Ensurerules() its equal")
					alreadyExists = true
					break
				}
			}
			if !alreadyExists {
				c.chainRules[newChain] = append(c.chainRules[newChain], newRuleArg)
			}
		}
	}
	return c.reconcile()
}

// GetIPv4ChainRuleArgs must be passed a regex that matches one IPv4 or IPv6 address which is added to a map as a key
// Chain will be selected with the associated regex match
func (c *Controller) GetIPv4ChainRuleArgs(table iptables.Table, chain iptables.Chain) ([]RuleArgs, error) {
	return getChainRuleArgs(c.iptV4, table, chain)
}

// GetIPv6ChainRuleArgs must be passed a regex that matches one IPv4 or IPv6 address which is added to a map as a key
// Chain will be selected with the associated regex match
func (c *Controller) GetIPv6ChainRuleArgs(table iptables.Table, chain iptables.Chain) ([]RuleArgs, error) {
	return getChainRuleArgs(c.iptV6, table, chain)
}

// GetIPv4ChainRuleArgs must be passed a regex that matches one IPv4 or IPv6 address which is added to a map as a key
// Chain will be selected with the associated regex match
func getChainRuleArgs(ipt iptables.Interface, table iptables.Table, chain iptables.Chain) ([]RuleArgs, error) {
	buf := bytes.NewBuffer(nil)

	if err := ipt.SaveInto(table, buf); err != nil {
		return nil, fmt.Errorf("failed to retrieve iptables table %s: %v", table, err)
	}
	rules := make([]RuleArgs, 0)
	for _, line := range strings.Split(buf.String(), "\n") {
		klog.Errorf("## found iptables line %q", line)
		if strings.HasPrefix(line, fmt.Sprintf("-A %s", string(chain))) {
			rules = append(rules, RuleArgs{
				// cleave off the -A ${chain_name}
				Args: strings.Split(line, " ")[2:],
			})
		}

	}
	return rules, nil
}

func processRules(ipt iptables.Interface, ownedChains []Chain, wantedChainRules map[Chain][]RuleArgs, existingChainRules map[Chain][]RuleArgs) error {
	var err error
	for _, wantedChain := range ownedChains {
		err = execIPTablesWithRetry(func() error {
			_, err = ipt.EnsureChain(wantedChain.Table, wantedChain.Chain)
			if err != nil {
				return fmt.Errorf("failed to ensure chain %s in table %s: %v", wantedChain.Chain, wantedChain.Table, err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to process iptables rules in table %s: %v", wantedChain.Table, err)
		}
	}
	for existingChain, existingRuleArgs := range existingChainRules {
		if existingChain.Proto != ipt.Protocol() {
			continue
		}
		wantedRuleArgs, found := wantedChainRules[existingChain]
		if !found {
			// we don't delete chains we don't manage, but we ensure rules are removed
			continue
		}
		klog.Errorf("## processRules(): ensuring chain %s in table %s rules: %+v", existingChain.Chain, existingChain.Table, wantedRuleArgs)
		err = execIPTablesWithRetry(func() error {
			_, err = ipt.EnsureChain(existingChain.Table, existingChain.Chain)
			if err != nil {
				return fmt.Errorf("failed to ensure chain %s in table %s: %v", existingChain.Chain, existingChain.Table, err)
			}
			return nil
		})
		if err != nil {
			klog.Errorf("IPTable manager: failed to process iptables rules: %v", err)
		}
		var isOwnedChain bool
		for _, ownedChain := range ownedChains {
			if ownedChain.equal(existingChain) {
				isOwnedChain = true
				break
			}
		}
		// owned chains means we exclusively control all rules in the chain and therefore remove any rules which we do not want
		if isOwnedChain {
			for _, existingRuleArg := range existingRuleArgs {
				found = false
				for _, wantedRuleArg := range wantedRuleArgs {
					if wantedRuleArg.equal(existingRuleArg) {
						found = true
						break
					}
				}
				if !found {
					if err = ipt.DeleteRule(existingChain.Table, existingChain.Chain, existingRuleArg.Args...); err != nil {
						klog.Errorf("IPTable manager: failed to delete stale rule (%s) in table %s and chain %s: %v",
							strings.Join(existingRuleArg.Args, " "), existingChain.Table, existingChain.Chain, err)
					}
				}
			}
		}
		// add the rules we want
		for _, wantedRule := range wantedRuleArgs {
			klog.Errorf("## about to insent args %v into chain %s", wantedRule.Args, existingChain.Chain)
			err = execIPTablesWithRetry(func() error {
				_, err = ipt.EnsureRule(iptables.Prepend, existingChain.Table, existingChain.Chain, wantedRule.Args...)
				if err != nil {
					return fmt.Errorf("failed to ensure rule (%v) in chain %s and table %s: %v", wantedRule.Args,
						existingChain.Chain, existingChain.Table, err)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to process iptables rules (%+v) in chain %s and table %s: %v", wantedRule.Args,
					existingChain.Chain, existingChain.Table, err)
			}
		}
	}
	return nil
}

// reconcile configures IP tables to ensure the correct chains and rules within the NAT table.
// CPU starvation or iptables lock held by an external entity may cause this function to take some time to execute.
func (c *Controller) reconcile() error {
	start := time.Now()
	defer func() {
		klog.V(5).Infof("Reconciling IP tables rules took %v", time.Since(start))
	}()

	existingChainRulesV4 := make(map[Chain][]RuleArgs)
	existingChainRulesV6 := make(map[Chain][]RuleArgs)

	// Gather existing rules from tables and chains
	for chain := range c.chainRules {
		// -A POSTROUTING -s 10.244.2.4/32 -o dummy0 -j SNAT --to-source 1.1.1.1
		// change below to match above
		//"-A %s -s ([^ ]*) -o ([^ ]*) .* --to ([^ ]*)",
		if c.v4 && chain.Proto == iptables.ProtocolIPv4 {
			rules, err := c.GetIPv4ChainRuleArgs(chain.Table, chain.Chain)
			if err != nil {
				return fmt.Errorf("failed to find IPv4 rules for chain %s in table %s", chain.Chain, chain.Table)
			} else {
				existingChainRulesV4[chain] = rules
			}
		} else if c.v6 && chain.Proto == iptables.ProtocolIPv6 {
			rules, err := c.GetIPv6ChainRuleArgs(chain.Table, chain.Chain)
			if err != nil {
				return fmt.Errorf("failed to find IPv6 rules for chain %s in table %s", chain.Chain, chain.Table)
			} else {
				existingChainRulesV6[chain] = rules
			}
		}
	}
	if c.v4 {
		if err := processRules(c.iptV4, c.chains, c.chainRules, existingChainRulesV4); err != nil {
			return fmt.Errorf("failed to process IPv4 rules: %v", err)
		}
	}
	if c.v6 {
		if err := processRules(c.iptV6, c.chains, c.chainRules, existingChainRulesV6); err != nil {
			return fmt.Errorf("failed to process IPv6 rules: %v", err)
		}
	}
	return nil
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
