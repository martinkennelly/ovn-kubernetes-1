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

type Controller struct {
	mu            *sync.Mutex
	rules         []nlRule
	ownPriorities map[int]bool
	v4            bool
	v6            bool
}

func NewController(v4, v6 bool) *Controller {
	return &Controller{
		mu:            &sync.Mutex{},
		rules:         make([]nlRule, 0),
		ownPriorities: make(map[int]bool, 0),
		v4:            v4,
		v6:            v6,
	}
}

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
				klog.Errorf("Rule manager: failed to reconcile (retry in 4 minutes): %v", err)
			}
			rm.mu.Unlock()
		}
	}
}

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
	tempRules := make([]nlRule, 0)
	for _, r := range rm.rules {
		// delete rule by first checking if it exists and if so, delete it
		if r.delete {
			if found, foundRoute := isNetlinkRuleInSlice(rulesFound, r.rule); found {
				if err = netlink.RuleDel(foundRoute); err != nil {
					// retry later
					tempRules = append(tempRules, r)
					errors = append(errors, err)
				}
			}
		} else {
			// add rule by first checking if it exists and if not, add it
			tempRules = append(tempRules, r)
			if found, _ := isNetlinkRuleInSlice(rulesFound, r.rule); !found {
				if err = netlink.RuleAdd(r.rule); err != nil {
					errors = append(errors, err)
				}
			}
		}
	}
	/**
		var found bool
		for priority, _ := range rm.ownPriorities {
			klog.Errorf("## priority !!!")
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
						klog.Errorf("Rule manager: failed to delete stale rule (%s) found at priority %d", ruleFound.String(), priority)
					}
				} else {
					klog.Infof("### Rule manager: found rule %s which we think is not stale", ruleFound.String())
				}
			}
		}
	**/
	rm.rules = tempRules
	return errors2.NewAggregate(errors)
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
