package groupscheduling

import (
	"fmt"
	"time"
)

type Source string
type LocalStatus string
type Lifecycle string

const (
	SourceInherit  Source = "inherit"
	SourceOverride Source = "override"

	StatusOpen   LocalStatus = "open"
	StatusLocked LocalStatus = "locked"

	LifecycleOpen   Lifecycle = "open"
	LifecycleLocked Lifecycle = "locked"
	LifecycleClosed Lifecycle = "closed"
)

type Project struct {
	ID                  string
	Name                string
	Status              string
	AutomaticScheduling bool
	SchedulingStartDate *time.Time
}

type Node struct {
	ID                          string
	ParentID                    *string
	Name                        string
	Source                      Source
	AutomaticSchedulingOverride *bool
	SchedulingStartDateOverride *time.Time
	LocalStatus                 LocalStatus
	LockedAutomaticScheduling   *bool
	LockedSchedulingStartDate   *time.Time
}

type Effective struct {
	AutomaticScheduling bool
	SchedulingStartDate *time.Time
	AutomaticOwnerID    string
	AutomaticOwnerName  string
	StartDateOwnerID    string
	StartDateOwnerName  string
	Lifecycle           Lifecycle
	LockOwnerID         string
	LockOwnerName       string
}

type Resolver struct {
	project  Project
	nodes    map[string]Node
	children map[string]int
}

func New(project Project, nodes []Node) (*Resolver, error) {
	byID := make(map[string]Node, len(nodes))
	children := make(map[string]int)
	for _, node := range nodes {
		if node.ID == "" {
			return nil, fmt.Errorf("group scheduling node ID is required")
		}
		if node.Source == "" {
			node.Source = SourceInherit
		}
		if node.LocalStatus == "" {
			node.LocalStatus = StatusOpen
		}
		if node.Source != SourceInherit && node.Source != SourceOverride {
			return nil, fmt.Errorf("invalid group scheduling source %q", node.Source)
		}
		if node.LocalStatus != StatusOpen && node.LocalStatus != StatusLocked {
			return nil, fmt.Errorf("invalid group local status %q", node.LocalStatus)
		}
		byID[node.ID] = node
		if node.ParentID != nil {
			children[*node.ParentID]++
		}
	}
	return &Resolver{project: project, nodes: byID, children: children}, nil
}

func (resolver *Resolver) IsGrouping(id string) bool { return resolver.children[id] > 0 }

func (resolver *Resolver) ProjectID() string { return resolver.project.ID }

func (resolver *Resolver) LocalLockOwnerFor(id string) (string, string, error) {
	node, ok := resolver.nodes[id]
	if !ok {
		return "", "", fmt.Errorf("group scheduling node %s was not found", id)
	}
	chain, err := resolver.ancestorChain(node)
	if err != nil {
		return "", "", err
	}
	if resolver.IsGrouping(id) {
		chain = append(chain, node)
	}
	for index := len(chain) - 1; index >= 0; index-- {
		if chain[index].LocalStatus == StatusLocked {
			return chain[index].ID, chain[index].Name, nil
		}
	}
	return "", "", nil
}

func (resolver *Resolver) EffectiveFor(id string) (Effective, error) {
	return resolver.resolve(id, true)
}

// InheritedFor resolves the configuration a Group would receive if its local
// scheduling source were inherit. The Group's own scheduling override and
// local lock are intentionally excluded; ancestor locks remain authoritative.
func (resolver *Resolver) InheritedFor(id string) (Effective, error) {
	return resolver.resolve(id, false)
}

func (resolver *Resolver) resolve(id string, includeSelf bool) (Effective, error) {
	node, ok := resolver.nodes[id]
	if !ok {
		return Effective{}, fmt.Errorf("group scheduling node %s was not found", id)
	}
	chain, err := resolver.ancestorChain(node)
	if err != nil {
		return Effective{}, err
	}
	if includeSelf && resolver.IsGrouping(id) {
		chain = append(chain, node)
	}
	value := Effective{
		AutomaticScheduling: resolver.project.AutomaticScheduling,
		SchedulingStartDate: cloneDate(resolver.project.SchedulingStartDate),
		AutomaticOwnerID:    resolver.project.ID,
		AutomaticOwnerName:  resolver.project.Name,
		StartDateOwnerID:    resolver.project.ID,
		StartDateOwnerName:  resolver.project.Name,
		Lifecycle:           LifecycleOpen,
	}
	if resolver.project.Status == "closed" {
		value.Lifecycle = LifecycleClosed
		value.LockOwnerID, value.LockOwnerName = resolver.project.ID, resolver.project.Name
	} else if resolver.project.Status == "locked" {
		value.Lifecycle = LifecycleLocked
		value.LockOwnerID, value.LockOwnerName = resolver.project.ID, resolver.project.Name
	}
	groupConfigFrozen := false
	for _, group := range chain {
		if groupConfigFrozen {
			continue
		}
		if group.LocalStatus == StatusLocked {
			if group.LockedAutomaticScheduling == nil {
				return Effective{}, fmt.Errorf("locked group scheduling snapshot is missing for %s", group.ID)
			}
			value.AutomaticScheduling = *group.LockedAutomaticScheduling
			value.AutomaticOwnerID, value.AutomaticOwnerName = group.ID, group.Name
			// A local lock snapshots the resolved anchor even when that anchor is nil.
			// LocalStatus=locked plus the non-null automatic snapshot distinguishes an
			// intentional frozen "no anchor" from an Open Group that simply inherits.
			value.SchedulingStartDate = cloneDate(group.LockedSchedulingStartDate)
			value.StartDateOwnerID, value.StartDateOwnerName = group.ID, group.Name
			if value.Lifecycle == LifecycleOpen {
				value.Lifecycle = LifecycleLocked
				value.LockOwnerID, value.LockOwnerName = group.ID, group.Name
			}
			// The first locked Group is a hard configuration ceiling for descendants.
			// Nested overrides/locks remain persisted locally but cannot alter the
			// effective frozen configuration until this ancestor is reopened.
			groupConfigFrozen = true
			continue
		}
		if group.Source == SourceOverride {
			if group.AutomaticSchedulingOverride != nil {
				value.AutomaticScheduling = *group.AutomaticSchedulingOverride
				value.AutomaticOwnerID, value.AutomaticOwnerName = group.ID, group.Name
			}
			if group.SchedulingStartDateOverride != nil {
				value.SchedulingStartDate = cloneDate(group.SchedulingStartDateOverride)
				value.StartDateOwnerID, value.StartDateOwnerName = group.ID, group.Name
			}
		}
	}
	return value, nil
}

func (resolver *Resolver) ancestorChain(node Node) ([]Node, error) {
	chain := make([]Node, 0)
	seen := map[string]bool{node.ID: true}
	current := node.ParentID
	for current != nil {
		if seen[*current] {
			return nil, fmt.Errorf("group scheduling hierarchy contains a cycle")
		}
		seen[*current] = true
		parent, ok := resolver.nodes[*current]
		if !ok {
			return nil, fmt.Errorf("group scheduling parent %s was not found", *current)
		}
		chain = append(chain, parent)
		current = parent.ParentID
	}
	for left, right := 0, len(chain)-1; left < right; left, right = left+1, right-1 {
		chain[left], chain[right] = chain[right], chain[left]
	}
	return chain, nil
}

func cloneDate(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
