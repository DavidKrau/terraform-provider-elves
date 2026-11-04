package provider

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/DavidKrau/elves-go-client"
)

// A multi-target assignment manages one rule with one action on a list of
// targets (SCIM users or configurations). Targets are handled as string keys
// so both resources share the logic; only the targets it lists are managed,
// other assignments of the same rule are left alone.

// assignedTarget is one target the rule is assigned to.
type assignedTarget struct {
	id     int64
	keys   []string // keys that refer to this target, normalized
	name   string   // key written to state on import
	action string
}

type multiTarget struct {
	kind      string // for messages, e.g. "SCIM user"
	normalize func(key string) string
	// resolve returns the target ID for a key, or an error if there is none
	resolve  func(c *elves.Client, key string) (int64, error)
	assigned func(c *elves.Client, ruleID int64) ([]assignedTarget, error)
	assign   func(c *elves.Client, targetID, ruleID int64, action string) error
	unassign func(c *elves.Client, targetID, ruleID int64) error
}

func multiAssignmentID(ruleID int64, action string) string {
	return fmt.Sprintf("%d/%s", ruleID, action)
}

func parseMultiAssignmentID(id string) (int64, string, error) {
	ruleStr, action, ok := strings.Cut(id, "/")
	ruleID, err := strconv.ParseInt(ruleStr, 10, 64)
	if !ok || err != nil || (action != "include" && action != "exclude") {
		return 0, "", fmt.Errorf("expected <rule_id>/<include|exclude>, got: %s", id)
	}
	return ruleID, action, nil
}

// byKey indexes the assigned targets by every key that refers to them.
func (m multiTarget) byKey(targets []assignedTarget) map[string]assignedTarget {
	index := map[string]assignedTarget{}
	for _, t := range targets {
		for _, k := range t.keys {
			index[k] = t
		}
	}
	return index
}

// resolveAll maps every key to its target ID and reports all unknown keys at once.
func (m multiTarget) resolveAll(c *elves.Client, keys []string) (map[string]int64, error) {
	ids := map[string]int64{}
	seen := map[int64]string{}
	var unknown []string
	for _, key := range keys {
		id, err := m.resolve(c, key)
		if err != nil {
			unknown = append(unknown, fmt.Sprintf("%s (%v)", key, err))
			continue
		}
		if other, ok := seen[id]; ok {
			return nil, fmt.Errorf("%q and %q are the same %s", other, key, m.kind)
		}
		seen[id] = key
		ids[key] = id
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown %s: %s", m.kind, strings.Join(unknown, ", "))
	}
	return ids, nil
}

// create assigns the rule to every key. A target that already has the rule
// was set up elsewhere (UI or another resource), so it is refused.
func (m multiTarget) create(c *elves.Client, ruleID int64, action string, keys []string) error {
	ids, err := m.resolveAll(c, keys)
	if err != nil {
		return err
	}
	targets, err := m.assigned(c, ruleID)
	if err != nil {
		return err
	}
	byID := map[int64]assignedTarget{}
	for _, t := range targets {
		byID[t.id] = t
	}
	var taken []string
	for _, key := range keys {
		if t, ok := byID[ids[key]]; ok {
			taken = append(taken, fmt.Sprintf("%s (%s)", key, t.action))
		}
	}
	if len(taken) > 0 {
		sort.Strings(taken)
		return fmt.Errorf("rule %d is already assigned to %s %s; remove it there or import it with ID %s",
			ruleID, m.kind, strings.Join(taken, ", "), multiAssignmentID(ruleID, action))
	}
	for _, key := range keys {
		if err := m.assign(c, ids[key], ruleID, action); err != nil {
			return fmt.Errorf("assigning rule %d to %s %s: %w", ruleID, m.kind, key, err)
		}
	}
	return nil
}

// read returns the keys from state that still have the rule with action. On
// import (no keys in state yet) it returns every target that has it.
func (m multiTarget) read(c *elves.Client, ruleID int64, action string, keys []string, importing bool) ([]string, error) {
	targets, err := m.assigned(c, ruleID)
	if err != nil {
		return nil, err
	}
	var result []string
	if importing {
		for _, t := range targets {
			if t.action == action {
				result = append(result, t.name)
			}
		}
		return result, nil
	}
	index := m.byKey(targets)
	for _, key := range keys {
		if t, ok := index[m.normalize(key)]; ok && t.action == action {
			result = append(result, key)
		}
	}
	return result, nil
}

// update moves the assignment from the old keys and action to the new ones.
func (m multiTarget) update(c *elves.Client, ruleID int64, oldAction, newAction string, oldKeys, newKeys []string) error {
	targets, err := m.assigned(c, ruleID)
	if err != nil {
		return err
	}
	index := m.byKey(targets)
	keep := map[string]bool{}
	for _, key := range newKeys {
		keep[m.normalize(key)] = true
	}
	for _, key := range oldKeys {
		if keep[m.normalize(key)] {
			continue
		}
		if t, ok := index[m.normalize(key)]; ok {
			if err := m.unassign(c, t.id, ruleID); err != nil && !isNotFound(err) {
				return fmt.Errorf("removing rule %d from %s %s: %w", ruleID, m.kind, key, err)
			}
		}
	}

	ids, err := m.resolveAll(c, newKeys)
	if err != nil {
		return err
	}
	for _, key := range newKeys {
		t, assigned := index[m.normalize(key)]
		if assigned && t.action == newAction {
			continue
		}
		if assigned && t.action != oldAction {
			// Assigned with the other action by someone else; don't take it over
			return fmt.Errorf("rule %d is already assigned to %s %s with action %s", ruleID, m.kind, key, t.action)
		}
		if err := m.assign(c, ids[key], ruleID, newAction); err != nil {
			return fmt.Errorf("assigning rule %d to %s %s: %w", ruleID, m.kind, key, err)
		}
	}
	return nil
}

// remove unassigns the rule from the keys that still have it with action.
func (m multiTarget) remove(c *elves.Client, ruleID int64, action string, keys []string) error {
	targets, err := m.assigned(c, ruleID)
	if err != nil {
		return err
	}
	index := m.byKey(targets)
	for _, key := range keys {
		t, ok := index[m.normalize(key)]
		if !ok || t.action != action {
			continue
		}
		if err := m.unassign(c, t.id, ruleID); err != nil && !isNotFound(err) {
			return fmt.Errorf("removing rule %d from %s %s: %w", ruleID, m.kind, key, err)
		}
	}
	return nil
}

var scimUserTargets = multiTarget{
	kind:      "SCIM user",
	normalize: strings.ToLower,
	resolve: func(c *elves.Client, email string) (int64, error) {
		user, err := c.ScimUserByEmail(email)
		if err != nil {
			return 0, err
		}
		if user == nil {
			return 0, fmt.Errorf("no SCIM user has this email or user name")
		}
		return user.ID, nil
	},
	assigned: func(c *elves.Client, ruleID int64) ([]assignedTarget, error) {
		users, err := c.RuleScimUsers(ruleID)
		if err != nil {
			return nil, err
		}
		targets := make([]assignedTarget, 0, len(users))
		for _, u := range users {
			t := assignedTarget{id: u.UserID, name: u.UserName, action: u.Action, keys: []string{strings.ToLower(u.UserName)}}
			if u.Email != "" {
				t.name = u.Email
				t.keys = append(t.keys, strings.ToLower(u.Email))
			}
			targets = append(targets, t)
		}
		return targets, nil
	},
	assign: func(c *elves.Client, userID, ruleID int64, action string) error {
		return c.ScimUserRuleAssign(userID, ruleID, action)
	},
	unassign: func(c *elves.Client, userID, ruleID int64) error {
		return c.ScimUserRuleUnassign(userID, ruleID)
	},
}

var configTargets = multiTarget{
	kind:      "configuration",
	normalize: func(key string) string { return key },
	resolve: func(_ *elves.Client, key string) (int64, error) {
		return strconv.ParseInt(key, 10, 64)
	},
	assigned: func(c *elves.Client, ruleID int64) ([]assignedTarget, error) {
		configs, err := c.RuleConfigs(ruleID)
		if err != nil {
			return nil, err
		}
		targets := make([]assignedTarget, 0, len(configs))
		for _, cfg := range configs {
			key := strconv.FormatInt(cfg.ConfigID, 10)
			targets = append(targets, assignedTarget{id: cfg.ConfigID, name: key, action: cfg.Action, keys: []string{key}})
		}
		return targets, nil
	},
	assign: func(c *elves.Client, configID, ruleID int64, action string) error {
		return c.ConfigRuleAssign(configID, ruleID, action)
	},
	unassign: func(c *elves.Client, configID, ruleID int64) error {
		return c.ConfigRuleUnassign(configID, ruleID)
	},
}
