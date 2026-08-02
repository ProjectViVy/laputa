// Package registry provides a persistent entity registry with learning capabilities.
// It tracks people, projects, and their relationships with confidence scores.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Entity represents a known person, project, or concept.
type Entity struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"` // "person", "project", "topic"
	Aliases       []string          `json:"aliases,omitempty"`
	Relationships map[string]string `json:"relationships,omitempty"` // name -> relationship
	Confidence    float64           `json:"confidence"`
	Source        string            `json:"source"` // "onboarding", "learned", "wiki"
	Frequency     int               `json:"frequency"`
	Notes         string            `json:"notes,omitempty"`
}

// Registry manages persistent entity storage and learning.
type Registry struct {
	path     string
	entities map[string]*Entity
	mu       sync.RWMutex
}

// New loads or creates an entity registry from the given palace path.
func New(palacePath string) (*Registry, error) {
	path := filepath.Join(palacePath, "entity_registry.json")
	r := &Registry{
		path:     path,
		entities: make(map[string]*Entity),
	}

	if err := r.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load registry: %w", err)
	}

	return r, nil
}

// Load reads the registry from disk.
func (r *Registry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}

	var entities []*Entity
	if err := json.Unmarshal(data, &entities); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	r.entities = make(map[string]*Entity)
	for _, e := range entities {
		r.entities[strings.ToLower(e.Name)] = e
	}
	return nil
}

// Save writes the registry to disk.
func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entities := make([]*Entity, 0, len(r.entities))
	for _, e := range r.entities {
		entities = append(entities, e)
	}

	data, err := json.MarshalIndent(entities, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(r.path), 0755); err != nil {
		return err
	}

	return os.WriteFile(r.path, data, 0644)
}

// Lookup finds an entity by name or alias.
func (r *Registry) Lookup(name string) *Entity {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := strings.ToLower(name)
	if e, ok := r.entities[key]; ok {
		return e
	}

	// Check aliases
	for _, e := range r.entities {
		for _, alias := range e.Aliases {
			if strings.ToLower(alias) == key {
				return e
			}
		}
	}
	return nil
}

// Add registers a new entity or updates an existing one.
func (r *Registry) Add(name, entityType string, confidence float64, source string) *Entity {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := strings.ToLower(name)
	if existing, ok := r.entities[key]; ok {
		existing.Frequency++
		if confidence > existing.Confidence {
			existing.Confidence = confidence
		}
		return existing
	}

	e := &Entity{
		Name:       name,
		Type:       entityType,
		Confidence: confidence,
		Source:     source,
		Frequency:  1,
	}
	r.entities[key] = e
	return e
}

// AddAlias adds an alias to an existing entity.
func (r *Registry) AddAlias(name, alias string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := strings.ToLower(name)
	e, ok := r.entities[key]
	if !ok {
		return fmt.Errorf("entity not found: %s", name)
	}

	e.Aliases = append(e.Aliases, alias)
	return nil
}

// AddRelationship adds a relationship between two entities.
func (r *Registry) AddRelationship(name, target, relationship string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := strings.ToLower(name)
	e, ok := r.entities[key]
	if !ok {
		return fmt.Errorf("entity not found: %s", name)
	}

	if e.Relationships == nil {
		e.Relationships = make(map[string]string)
	}
	e.Relationships[target] = relationship
	return nil
}

// List returns all entities, optionally filtered by type.
func (r *Registry) List(entityType string) []*Entity {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Entity
	for _, e := range r.entities {
		if entityType == "" || e.Type == entityType {
			result = append(result, e)
		}
	}
	return result
}

// LearnFromText extracts potential entities from text and adds them with low confidence.
func (r *Registry) LearnFromText(text string, detector interface{ Detect(text string) []Entity }) []*Entity {
	type Detector interface {
		Detect(text string) []Entity
	}

	d, ok := detector.(Detector)
	if !ok {
		return nil
	}

	var learned []*Entity
	entities := d.Detect(text)

	for _, e := range entities {
		if e.Confidence < 0.5 {
			continue
		}

		key := strings.ToLower(e.Name)
		r.mu.Lock()
		if existing, ok := r.entities[key]; ok {
			existing.Frequency++
			learned = append(learned, existing)
		} else {
			newEntity := &Entity{
				Name:       e.Name,
				Type:       e.Type,
				Confidence: e.Confidence * 0.7, // Lower confidence for learned entities
				Source:     "learned",
				Frequency:  e.Frequency,
			}
			r.entities[key] = newEntity
			learned = append(learned, newEntity)
		}
		r.mu.Unlock()
	}

	return learned
}

// Count returns the number of registered entities.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entities)
}
