package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dashimaki/laputa/governance"
)

type GovernanceWriter interface {
	GetSection(context.Context, governance.SectionName) (map[string]any, error)
	SetSection(context.Context, governance.SectionName, map[string]any) error
}

type Checkpointer struct {
	Gov GovernanceWriter
	WS  *WorkingSet
}

func (c *Checkpointer) Save(ctx context.Context, scope string) error {
	if c.Gov == nil {
		return fmt.Errorf("governance unavailable")
	}
	if c.WS == nil {
		return fmt.Errorf("working set unavailable")
	}
	snap := c.WS.Snapshot(scope)
	section, err := c.Gov.GetSection(ctx, governance.SectionMemoryMD)
	if err != nil {
		return err
	}
	if section == nil {
		section = map[string]any{}
	}
	workingSets, _ := section["working_sets"].(map[string]any)
	if workingSets == nil {
		workingSets = map[string]any{}
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	var snapMap map[string]any
	if err := json.Unmarshal(raw, &snapMap); err != nil {
		return err
	}
	key := scope
	if key == "" {
		key = "_default"
	}
	workingSets[key] = snapMap
	section["working_sets"] = workingSets
	meta, _ := section["_meta"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
	}
	meta["checkpoint_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	section["_meta"] = meta
	if err := c.Gov.SetSection(ctx, governance.SectionMemoryMD, section); err != nil {
		return err
	}
	c.WS.MarkCheckpoint(scope)
	return nil
}

func (c *Checkpointer) Load(ctx context.Context, scope string) error {
	if c.Gov == nil {
		return fmt.Errorf("governance unavailable")
	}
	if c.WS == nil {
		return fmt.Errorf("working set unavailable")
	}
	section, err := c.Gov.GetSection(ctx, governance.SectionMemoryMD)
	if err != nil {
		return err
	}
	workingSets, _ := section["working_sets"].(map[string]any)
	if workingSets == nil {
		return nil
	}
	key := scope
	if key == "" {
		key = "_default"
	}
	raw, ok := workingSets[key]
	if !ok {
		return nil
	}
	rawJSON, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	var snap ScopeSnapshot
	if err := json.Unmarshal(rawJSON, &snap); err != nil {
		return err
	}
	c.WS.Restore(scope, snap)
	return nil
}
