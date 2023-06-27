package rulemanager

import (
	"fmt"
	"sync"
	"time"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"

	"github.com/vishvananda/netlink"
)

type nlRule struct {
	rule   *netlink.Rule
	delete bool
}

type Controller struct {
	mu    *sync.Mutex
	rules []nlRule
	// only explicit rules (via fn AddRule) are allowed when a priority is owned. Other rules will be removed.
	ownPriorities map[int]bool
	v4            bool
	v6            bool
}

// NewController creates a new linux rule manager
func NewController(v4, v6 bool) *Controller {
	return &Controller{
		mu:            &sync.Mutex{},
		rules:         make([]nlRule, 0),
		ownPriorities: make(map[int]bool, 0),
		v4:            v4,
		v6:            v6,
	}
}

// Run starts manages linux rules
func (rm *Controller) Run(stopCh <-chan struct{}, syncPeriod time.Duration) {
	var err error
	ticker := time.NewTicker(syncPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			rm.mu.Lock()
			if err = rm.reconcile(); err != nil {
				klog.Errorf("Rule manager: failed to reconcile (retry in %s): %v", syncPeriod.String(), err)
			}
			rm.mu.Unlock()
		}
	}
}

// AddRule ensures a rule is applied even if it is altered by something else, it will be restored
func (rm *Controller) AddRule(rule netlink.Rule) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	// check if we are already managing this route and if so, no-op
	for _, existingRule := range rm.rules {
		if areNetlinkRulesEqual(existingRule.rule, &rule) {
			return nil
		}
	}
	rm.rules = append(rm.rules, nlRule{rule: &rule})
	return rm.reconcile()
}

// DeleteRule stops managed a rule and ensures its deleted
func (rm *Controller) DeleteRule(rule netlink.Rule) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	var reconcileNeeded bool
	for i, r := range rm.rules {
		if areNetlinkRulesEqual(r.rule, &rule) {
			rm.rules[i].delete = true
			reconcileNeeded = true
			break
		}
	}
	if reconcileNeeded {
		return rm.reconcile()
	}
	return nil
}

// OwnPriority ensures any rules observed with priority 'priority' must be specified otherwise its removed
func (rm *Controller) OwnPriority(priority int) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.ownPriorities[priority] = true
	return rm.reconcile()
}

func (rm *Controller) reconcile() error {
	start := time.Now()
	defer func() {
		klog.V(5).Infof("Reconciling rules took %v", time.Since(start))
	}()
	var family int
	if rm.v4 && rm.v6 {
		family = netlink.FAMILY_ALL
	} else if rm.v4 {
		family = netlink.FAMILY_V4
	} else if rm.v6 {
		family = netlink.FAMILY_V6
	}

	rulesFound, err := netlink.RuleList(family)
	if err != nil {
		return err
	}
	var errors []error
	rulesToKeep := make([]nlRule, 0)
	for _, r := range rm.rules {
		// delete rule by first checking if it exists and if so, delete it
		if r.delete {
			if found, foundRoute := isNetlinkRuleInSlice(rulesFound, r.rule); found {
				if err = netlink.RuleDel(foundRoute); err != nil {
					// retry later
					rulesToKeep = append(rulesToKeep, r)
					errors = append(errors, err)
				}
			}
		} else {
			// add rule by first checking if it exists and if not, add it
			rulesToKeep = append(rulesToKeep, r)
			if found, _ := isNetlinkRuleInSlice(rulesFound, r.rule); !found {
				if err = netlink.RuleAdd(r.rule); err != nil {
					errors = append(errors, err)
				}
			}
		}
	}

	var found bool
	for priority := range rm.ownPriorities {
		for _, ruleFound := range rulesFound {
			if ruleFound.Priority != priority {
				continue
			}
			found = false
			for _, ruleWanted := range rm.rules {
				if ruleWanted.rule.Priority != priority {
					continue
				}
				if areNetlinkRulesEqual(ruleWanted.rule, &ruleFound) {
					found = true
					break
				}
			}
			if !found {
				klog.Infof("Rule manager: deleting stale rule (%s) found at priority %d", ruleFound.String(), priority)
				if err = netlink.RuleDel(&ruleFound); err != nil {
					errors = append(errors, fmt.Errorf("failed to delete stale rule (%s) found at priority %d: %v",
						ruleFound.String(), priority, err))
				}
			}
		}
	}

	rm.rules = rulesToKeep
	return utilerrors.NewAggregate(errors)
}

func areNetlinkRulesEqual(r1, r2 *netlink.Rule) bool {
	return r1.String() == r2.String()
}

func isNetlinkRuleInSlice(rules []netlink.Rule, candidate *netlink.Rule) (bool, *netlink.Rule) {
	for _, r := range rules {
		r := r
		if r.Priority != candidate.Priority {
			continue
		}
		if areNetlinkRulesEqual(&r, candidate) {
			return true, &r
		}
	}
	return false, netlink.NewRule()
}
