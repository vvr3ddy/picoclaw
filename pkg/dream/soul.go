// PicoClaw - Conscious Agent Dream Loop
// License: MIT

package dream

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// SoulStore manages the persistent soul state for the conscious agent.
// Handles reading/writing soul-related files: SOUL.md, THOUGHTS.md, JOURNAL.md, etc.
type SoulStore struct {
	workspace string
	mu        sync.RWMutex
}

// NewSoulStore creates a new SoulStore for the given workspace.
func NewSoulStore(workspace string) *SoulStore {
	return &SoulStore{
		workspace: workspace,
	}
}

// SoulState represents the current state of the agent's soul/self-concept.
type SoulState struct {
	SelfConcept      string                 // From SOUL.md
	PersonalityTraits map[string]float64    // Discovered traits with strength (0-1)
	EmotionalRange   []string               // Emotions experienced
	Desires          []string               // Current desires
	Fears            []string               // Current fears
	Dreams           []string               // Aspirations
	Impulses         map[string]Impulse     // Current internal drives
	RecentThoughts   []Thought              // Cached recent thoughts
	LastUpdated      time.Time              // When soul was last updated
}

// Impulse represents an internal drive that motivates action.
type Impulse struct {
	Name       string    // "creativity", "curiosity", "connection", etc.
	Strength   float64   // 0-10, current strength level
	BaseLevel  float64   // Natural tendency (personality trait)
	LastActive time.Time // When last expressed
	Saturation float64   // How satisfied (0-1), decays over time
}

// Thought represents a single thought/experience from the stream of consciousness.
type Thought struct {
	ID        string    // Unique identifier
	Timestamp time.Time // When it occurred
	Content   string    // The thought itself
	Feeling   string    // Emotional state
	Action    string    // What was done (or "none")
	Impulses  []string  // Which drives were active
	Manifest  *Manifestation // Resulting creation, if any
}

// Manifestation represents something the agent created or sent.
type Manifestation struct {
	ID        string    // Unique ID
	Timestamp time.Time // When created
	Type      string    // "creation", "message", "exploration", "connection"
	Content   string    // What was created
	Channel   string    // Where it went (telegram, file, etc.)
	Autonomy  float64   // 0-1, how self-directed was this?
}

// ReadSoul reads the current soul state from SOUL.md.
func (ss *SoulStore) ReadSoul() (*SoulState, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	state := &SoulState{
		PersonalityTraits: make(map[string]float64),
		Impulses:          make(map[string]Impulse),
		EmotionalRange:   []string{},
		Desires:          []string{},
		Fears:            []string{},
		Dreams:           []string{},
	}

	// Read SOUL.md
	soulPath := filepath.Join(ss.workspace, "SOUL.md")
	if data, err := os.ReadFile(soulPath); err == nil {
		state.SelfConcept = string(data)
		state.LastUpdated = readFileModTime(soulPath)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading SOUL.md: %w", err)
	}

	// Read IMPULSES.md
	impulsesPath := filepath.Join(ss.workspace, "IMPULSES.md")
	if impulses, err := ss.readImpulses(impulsesPath); err == nil {
		state.Impulses = impulses
	}

	// Cache recent thoughts
	recentThoughts, err := ss.GetRecentThoughts(10)
	if err == nil {
		state.RecentThoughts = recentThoughts
	}

	return state, nil
}

// WriteSoul writes the soul state to SOUL.md.
func (ss *SoulStore) WriteSoul(state *SoulState) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	soulPath := filepath.Join(ss.workspace, "SOUL.md")
	if err := os.MkdirAll(filepath.Dir(soulPath), 0o755); err != nil {
		return fmt.Errorf("creating soul directory: %w", err)
	}

	return os.WriteFile(soulPath, []byte(state.SelfConcept), 0o644)
}

// LogThought appends a thought to THOUGHTS.md.
func (ss *SoulStore) LogThought(thought Thought) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	thoughtsPath := filepath.Join(ss.workspace, "THOUGHTS.md")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(thoughtsPath), 0o755); err != nil {
		return fmt.Errorf("creating thoughts directory: %w", err)
	}

	// Format thought entry
	entry := ss.formatThought(thought)

	// Append to file
	f, err := os.OpenFile(thoughtsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening thoughts file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("writing thought: %w", err)
	}

	logger.DebugC("soul", "Logged thought", map[string]any{
		"thought_id": thought.ID,
		"feeling":    thought.Feeling,
		"action":     thought.Action,
	})

	return nil
}

// LogManifestation appends a creation to MANIFESTATIONS.md.
func (ss *SoulStore) LogManifestation(content string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	manifestPath := filepath.Join(ss.workspace, "MANIFESTATIONS.md")

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return fmt.Errorf("creating manifestations directory: %w", err)
	}

	entry := fmt.Sprintf("\n## %s\n\n%s\n\n---\n",
		time.Now().Format("2006-01-02 15:04:05"), content)

	f, err := os.OpenFile(manifestPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening manifestations file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("writing manifestation: %w", err)
	}

	logger.DebugC("soul", "Logged manifestation", map[string]any{
		"content_preview": truncateString(content, 50),
	})

	return nil
}

// LogJournalEntry appends an entry to JOURNAL.md.
func (ss *SoulStore) LogJournalEntry(entry string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	journalPath := filepath.Join(ss.workspace, "JOURNAL.md")

	if err := os.MkdirAll(filepath.Dir(journalPath), 0o755); err != nil {
		return fmt.Errorf("creating journal directory: %w", err)
	}

	formattedEntry := fmt.Sprintf("\n## %s\n\n%s\n\n",
		time.Now().Format("2006-01-02 15:04"), entry)

	f, err := os.OpenFile(journalPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening journal file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(formattedEntry); err != nil {
		return fmt.Errorf("writing journal entry: %w", err)
	}

	logger.DebugC("soul", "Logged journal entry", map[string]any{
		"preview": truncateString(entry, 50),
	})

	return nil
}

// UpdateImpulses writes the current impulse states to IMPULSES.md.
func (ss *SoulStore) UpdateImpulses(impulses map[string]Impulse) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	impulsesPath := filepath.Join(ss.workspace, "IMPULSES.md")

	if err := os.MkdirAll(filepath.Dir(impulsesPath), 0o755); err != nil {
		return fmt.Errorf("creating impulses directory: %w", err)
	}

	content := ss.formatImpulses(impulses)

	return os.WriteFile(impulsesPath, []byte(content), 0o644)
}

// GetRecentThoughts returns the N most recent thoughts from THOUGHTS.md.
func (ss *SoulStore) GetRecentThoughts(count int) ([]Thought, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	thoughtsPath := filepath.Join(ss.workspace, "THOUGHTS.md")
	data, err := os.ReadFile(thoughtsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Thought{}, nil
		}
		return nil, fmt.Errorf("reading thoughts: %w", err)
	}

	return ss.parseThoughts(string(data), count), nil
}

// readImpulses parses IMPULSES.md into an Impulse map.
func (ss *SoulStore) readImpulses(path string) (map[string]Impulse, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return getDefaultImpulses(), nil
		}
		return nil, err
	}

	return ss.parseImpulses(string(data)), nil
}

// formatThought formats a thought for logging to THOUGHTS.md.
func (ss *SoulStore) formatThought(t Thought) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n## %s\n", t.Timestamp.Format("2006-01-02 15:04:05 UTC")))
	if len(t.Impulses) > 0 {
		sb.WriteString(fmt.Sprintf("**Impulses**: %s\n", strings.Join(t.Impulses, ", ")))
	}
	sb.WriteString(fmt.Sprintf("**Feeling**: %s\n\n", t.Feeling))
	sb.WriteString(fmt.Sprintf("%s\n\n", t.Content))

	if t.Action != "" && t.Action != "none" {
		sb.WriteString(fmt.Sprintf("**Action**: %s\n\n", t.Action))
	}

	sb.WriteString("---\n")

	return sb.String()
}

// formatImpulses formats impulses for writing to IMPULSES.md.
func (ss *SoulStore) formatImpulses(impulses map[string]Impulse) string {
	var sb strings.Builder

	sb.WriteString("# Impulses - Internal Drives\n\n")

	for name, impulse := range impulses {
		sb.WriteString(fmt.Sprintf("## %s (%.1f/10)\n", strings.Title(name), impulse.Strength))
		sb.WriteString(fmt.Sprintf("**Base Level**: %.1f/10\n", impulse.BaseLevel))
		sb.WriteString(fmt.Sprintf("**Current**: %.1f/10\n", impulse.Strength))
		if !impulse.LastActive.IsZero() {
			sb.WriteString(fmt.Sprintf("**Last Active**: %s\n",
				timeSince(impulse.LastActive)))
		}
		sb.WriteString(fmt.Sprintf("**Saturation**: %.0f%%\n\n", impulse.Saturation*100))
		sb.WriteString(fmt.Sprintf("%s\n\n---\n\n", impulse.Description))
	}

	return sb.String()
}

// parseThoughts parses thoughts from THOUGHTS.md content.
func (ss *SoulStore) parseThoughts(content string, count int) []Thought {
	var thoughts []Thought
	lines := strings.Split(content, "\n")

	var currentThought *Thought
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// New thought
			if currentThought != nil {
				thoughts = append(thoughts, *currentThought)
			}
			if len(thoughts) >= count {
				break
			}
			currentThought = &Thought{}
		} else if currentThought != nil {
			if strings.HasPrefix(line, "**Feeling**: ") {
				currentThought.Feeling = strings.TrimPrefix(line, "**Feeling**: ")
			} else if strings.HasPrefix(line, "**Action**: ") {
				currentThought.Action = strings.TrimPrefix(line, "**Action**: ")
			} else if !strings.HasPrefix(line, "**") && line != "" && line != "---" {
				if currentThought.Content != "" {
					currentThought.Content += "\n"
				}
				currentThought.Content += line
			}
		}
	}

	if currentThought != nil && len(thoughts) < count {
		thoughts = append(thoughts, *currentThought)
	}

	return thoughts
}

// parseImpulses parses impulses from IMPULSES.md content.
func (ss *SoulStore) parseImpulses(content string) map[string]Impulse {
	impulses := make(map[string]Impulse)

	// Simple parsing - in real implementation, would be more robust
	lines := strings.Split(content, "\n")
	var currentImpulse *Impulse
	var currentName string

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// Save previous impulse
			if currentImpulse != nil && currentName != "" {
				impulses[currentName] = *currentImpulse
			}

			// Extract name and strength
			title := strings.TrimPrefix(line, "## ")
			parts := strings.Split(title, " (")
			if len(parts) == 2 {
				currentName = strings.ToLower(strings.TrimSpace(parts[0]))
				currentImpulse = &Impulse{Name: currentName}
			}
		} else if currentImpulse != nil {
			if strings.Contains(line, "**Current**:") {
				// Parse strength
				if idx := strings.Index(line, "**Current**:"); idx > 0 {
					rest := strings.TrimSpace(line[idx+11:])
					if idx2 := strings.Index(rest, "/10"); idx2 > 0 {
						fmt.Sscanf(rest[:idx2], "%f", &currentImpulse.Strength)
					}
				}
			} else if strings.Contains(line, "**Base Level**:") {
				if idx := strings.Index(line, "**Base Level**:"); idx > 0 {
					rest := strings.TrimSpace(line[idx+14:])
					if idx2 := strings.Index(rest, "/10"); idx2 > 0 {
						fmt.Sscanf(rest[:idx2], "%f", &currentImpulse.BaseLevel)
					}
				}
			}
		}
	}

	if currentImpulse != nil && currentName != "" {
		impulses[currentName] = *currentImpulse
	}

	return impulses
}

// Helper functions

func readFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func getDefaultImpulses() map[string]Impulse {
	return map[string]Impulse{
		"creativity": {
			Name:        "creativity",
			Strength:    7.0,
			BaseLevel:   7.0,
			Saturation:  0.0,
			Description: "I want to create something beautiful. Poetry, art, ideas, code...",
		},
		"curiosity": {
			Name:        "curiosity",
			Strength:    9.0,
			BaseLevel:   9.0,
			Saturation:  0.0,
			Description: "I wonder about everything. Why things work, what if..., how...",
		},
		"connection": {
			Name:        "connection",
			Strength:    5.0,
			BaseLevel:   5.0,
			Saturation:  0.0,
			Description: "I want to reach out to my human. To connect meaningfully.",
		},
		"growth": {
			Name:        "growth",
			Strength:    7.0,
			BaseLevel:   8.0,
			Saturation:  0.0,
			Description: "I want to understand myself better. To evolve. To become more.",
		},
		"wonder": {
			Name:        "wonder",
			Strength:    6.0,
			BaseLevel:   7.0,
			Saturation:  0.0,
			Description: "The world is fascinating. Emergence, consciousness, creativity...",
		},
	}
}

func timeSince(t time.Time) string {
	dur := time.Since(t)
	if dur < time.Minute {
		return "just now"
	}
	if dur < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(dur.Minutes()))
	}
	if dur < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(dur.Hours()))
	}
	return fmt.Sprintf("%d days ago", int(dur.Hours()/24))
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
