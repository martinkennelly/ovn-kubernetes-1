package rulemanager

import (
	"github.com/vishvananda/netlink"
	errors2 "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"sync"
	"time"
)

type nlRule struct {
	rule   *netlink.Rule
	delete bool
}

type ruleManager struct {
	mu    *sync.Mutex
	rules []nlRule
	v4    bool
	v6    bool
}

func newRuleManager(v4, v6 bool) ruleManager {
	return ruleManager{
		mu:    &sync.Mutex{},
		rules: make([]nlRule, 0),
		v4:    v4,
		v6:    v6,
	}
}

func (rm *ruleManager) run(stopCh chan struct{}, wait sync.WaitGroup) {
	go func() {
		defer wait.Done()
		ticker := time.NewTicker(4 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				rm.mu.Lock()
				rm.reconcile()
				rm.mu.Unlock()
			}
		}
	}()
}

func (rm *ruleManager) addRule(rule *netlink.Rule) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.rules = append(rm.rules, nlRule{rule: rule})
	return rm.reconcile()
}

func (rm *ruleManager) deleteRule(rule *netlink.Rule) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	var reconcileNeeded bool
	for i, r := range rm.rules {
		if areNetlinkRulesEqual(r.rule, rule) {
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

func (rm *ruleManager) reconcile() error {
	start := time.Now()
	defer func() {
		klog.V(4).Infof("reconciling rules took %v", time.Since(start))
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
	tempRules := rm.rules[:0]
	for _, r := range rm.rules {
		// delete rule by first checking if it exists and if so, delete it
		if r.delete {
			if isNetlinkRuleInSlice(rulesFound, r.rule) {
				if err = netlink.RuleDel(r.rule); err != nil {
					// retry later
					tempRules = append(tempRules, r)
					errors = append(errors, err)
				}
			}
		} else {
			// add rule by first checking if it exists and if not, add it
			tempRules = append(tempRules, r)
			if !isNetlinkRuleInSlice(rulesFound, r.rule) {
				if err = netlink.RuleAdd(r.rule); err != nil {
					errors = append(errors, err)
				}
			}
		}
	}
	rm.rules = tempRules
	return errors2.NewAggregate(errors)
}

func areNetlinkRulesEqual(r1, r2 *netlink.Rule) bool {
	if r1.String() == r2.String() {
		return true
	}
	return false
}

func isNetlinkRuleInSlice(rules []netlink.Rule, candidate *netlink.Rule) bool {
	for _, r := range rules {
		if areNetlinkRulesEqual(&r, candidate) {
			return true
		}
	}
	return false
}
