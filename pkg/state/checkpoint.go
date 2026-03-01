package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// CheckpointVersion is the current version of the checkpoint format.
// Increment this when making breaking changes to the checkpoint structure.
const CheckpointVersion = 1

// CheckpointState indicates whether the last shutdown was clean or unclean.
type CheckpointState string

const (
	// CheckpointStateClean indicates a graceful shutdown.
	CheckpointStateClean CheckpointState = "clean"
	// CheckpointStateDirty indicates an unclean shutdown (crash/power loss).
	CheckpointStateDirty CheckpointState = "dirty"
)

// ToolExecution represents an in-progress tool call that needs to be resumed.
type ToolExecution struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	StartTime time.Time       `json:"start_time"`
}

// Checkpoint represents a point-in-time snapshot of agent state for recovery.
type Checkpoint struct {
	Version     int             `json:"version"`
	State       CheckpointState `json:"state"`
	Timestamp   time.Time       `json:"timestamp"`
	SessionID   string          `json:"session_id"`
	Messages    []Message       `json:"messages"`
	CurrentTool *ToolExecution  `json:"current_tool,omitempty"`
	ActiveAgent string          `json:"active_agent"`
	Channel     string          `json:"channel"`
	ChatID      string          `json:"chat_id"`
	LastAction  string          `json:"last_action"`
	RetryCount  int             `json:"retry_count"`
}

// Message is a serializable representation of providers.Message for checkpoint storage.
type Message struct {
	Role             string               `json:"role"`
	Content          string               `json:"content"`
	ReasoningContent string               `json:"reasoning_content,omitempty"`
	ToolCalls        []providers.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string               `json:"tool_call_id,omitempty"`
}

// CheckpointManager manages periodic checkpoints of agent state for crash recovery.
// It follows the same pattern as Manager, using atomic file operations for durability.
type CheckpointManager struct {
	workspace      string
	checkpointDir  string
	checkpointFile string
	historyDir     string
	mu             sync.RWMutex
	checkpoint     *Checkpoint
	maxHistory     int
}

// NewCheckpointManager creates a new checkpoint manager for the given workspace.
// It loads any existing checkpoint and prepares the checkpoint directory structure.
func NewCheckpointManager(workspace string) *CheckpointManager {
	checkpointDir := filepath.Join(workspace, "state")
	checkpointFile := filepath.Join(checkpointDir, "checkpoint.json")
	historyDir := filepath.Join(checkpointDir, "history")

	// Create directories if they do not exist
	os.MkdirAll(checkpointDir, 0o755)
	os.MkdirAll(historyDir, 0o755)

	cm := &CheckpointManager{
		workspace:      workspace,
		checkpointDir:  checkpointDir,
		checkpointFile: checkpointFile,
		historyDir:     historyDir,
		checkpoint: &Checkpoint{
			Version: CheckpointVersion,
			State:   CheckpointStateClean,
		},
		maxHistory: 5,
	}

	// Attempt to load existing checkpoint
	cm.load()

	return cm
}

// IsRecoveryNeeded returns true if the last shutdown was unclean and recovery is needed.
func (cm *CheckpointManager) IsRecoveryNeeded() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.checkpoint.State == CheckpointStateDirty
}

// GetCheckpoint returns a copy of the current checkpoint.
// Returns nil if no checkpoint exists.
func (cm *CheckpointManager) GetCheckpoint() *Checkpoint {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.checkpoint == nil {
		return nil
	}

	// Return a deep copy to prevent external modification
	return cm.copyCheckpoint(cm.checkpoint)
}

// GetMessages returns the stored messages from the checkpoint.
func (cm *CheckpointManager) GetMessages() []providers.Message {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]providers.Message, len(cm.checkpoint.Messages))
	for i, msg := range cm.checkpoint.Messages {
		result[i] = providers.Message{
			Role:             msg.Role,
			Content:          msg.Content,
			ReasoningContent: msg.ReasoningContent,
			ToolCalls:        msg.ToolCalls,
			ToolCallID:       msg.ToolCallID,
		}
	}
	return result
}

// GetCurrentTool returns the in-progress tool execution, if any.
func (cm *CheckpointManager) GetCurrentTool() *ToolExecution {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.checkpoint.CurrentTool == nil {
		return nil
	}

	// Return a copy
	tool := *cm.checkpoint.CurrentTool
	return &tool
}

// SaveDirty saves the checkpoint with dirty state (indicates active processing).
// This should be called at the start of message processing or before tool execution.
func (cm *CheckpointManager) SaveDirty(
	sessionID string,
	messages []providers.Message,
	activeAgent string,
	channel string,
	chatID string,
	lastAction string,
	currentTool *ToolExecution,
) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Archive existing checkpoint if present
	if err := cm.archiveIfExists(); err != nil {
		return fmt.Errorf("failed to archive existing checkpoint: %w", err)
	}

	// Convert messages to serializable format
	checkpointMsgs := make([]Message, len(messages))
	for i, msg := range messages {
		checkpointMsgs[i] = Message{
			Role:             msg.Role,
			Content:          msg.Content,
			ReasoningContent: msg.ReasoningContent,
			ToolCalls:        msg.ToolCalls,
			ToolCallID:       msg.ToolCallID,
		}
	}

	cm.checkpoint = &Checkpoint{
		Version:     CheckpointVersion,
		State:       CheckpointStateDirty,
		Timestamp:   time.Now(),
		SessionID:   sessionID,
		Messages:    checkpointMsgs,
		CurrentTool: currentTool,
		ActiveAgent: activeAgent,
		Channel:     channel,
		ChatID:      chatID,
		LastAction:  lastAction,
		RetryCount:  cm.checkpoint.RetryCount + 1,
	}

	return cm.saveAtomic()
}

// SaveClean saves the checkpoint with clean state (indicates completed processing).
// This should be called after successful message processing.
func (cm *CheckpointManager) SaveClean(
	sessionID string,
	messages []providers.Message,
	activeAgent string,
	channel string,
	chatID string,
) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Archive existing checkpoint if present
	if err := cm.archiveIfExists(); err != nil {
		return fmt.Errorf("failed to archive existing checkpoint: %w", err)
	}

	// Convert messages to serializable format
	checkpointMsgs := make([]Message, len(messages))
	for i, msg := range messages {
		checkpointMsgs[i] = Message{
			Role:             msg.Role,
			Content:          msg.Content,
			ReasoningContent: msg.ReasoningContent,
			ToolCalls:        msg.ToolCalls,
			ToolCallID:       msg.ToolCallID,
		}
	}

	cm.checkpoint = &Checkpoint{
		Version:     CheckpointVersion,
		State:       CheckpointStateClean,
		Timestamp:   time.Now(),
		SessionID:   sessionID,
		Messages:    checkpointMsgs,
		ActiveAgent: activeAgent,
		Channel:     channel,
		ChatID:      chatID,
		RetryCount:  0,
	}

	return cm.saveAtomic()
}

// MarkClean marks the current checkpoint as clean without modifying other fields.
// Use this for graceful shutdown.
func (cm *CheckpointManager) MarkClean() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.checkpoint.State = CheckpointStateClean
	cm.checkpoint.Timestamp = time.Now()
	return cm.saveAtomic()
}

// Clear removes the current checkpoint and archives it.
// Use this after successful recovery.
func (cm *CheckpointManager) Clear() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Archive before clearing
	if err := cm.archiveIfExists(); err != nil {
		return fmt.Errorf("failed to archive checkpoint before clear: %w", err)
	}

	cm.checkpoint = &Checkpoint{
		Version: CheckpointVersion,
		State:   CheckpointStateClean,
	}

	// Remove the checkpoint file
	if err := os.Remove(cm.checkpointFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to remove checkpoint file: %w", err)
	}

	return nil
}

// SetMaxHistory sets the maximum number of historical checkpoints to keep.
// Default is 5. Must be called before operations that create archives.
func (cm *CheckpointManager) SetMaxHistory(max int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if max > 0 {
		cm.maxHistory = max
	}
}

// copyCheckpoint creates a deep copy of a checkpoint.
func (cm *CheckpointManager) copyCheckpoint(cp *Checkpoint) *Checkpoint {
	if cp == nil {
		return nil
	}

	copy := &Checkpoint{
		Version:     cp.Version,
		State:       cp.State,
		Timestamp:   cp.Timestamp,
		SessionID:   cp.SessionID,
		ActiveAgent: cp.ActiveAgent,
		Channel:     cp.Channel,
		ChatID:      cp.ChatID,
		LastAction:  cp.LastAction,
		RetryCount:  cp.RetryCount,
	}

	if len(cp.Messages) > 0 {
		copy.Messages = make([]Message, len(cp.Messages))
		for i, msg := range cp.Messages {
			copy.Messages[i] = msg
			if len(msg.ToolCalls) > 0 {
				copy.Messages[i].ToolCalls = make([]providers.ToolCall, len(msg.ToolCalls))
				copy.Messages[i].ToolCalls = append([]providers.ToolCall(nil), msg.ToolCalls...)
			}
		}
	}

	if cp.CurrentTool != nil {
		toolCopy := *cp.CurrentTool
		copy.CurrentTool = &toolCopy
	}

	return copy
}

// archiveIfExists moves the current checkpoint to the history directory.
// Uses millisecond precision to avoid collisions during rapid saves.
func (cm *CheckpointManager) archiveIfExists() error {
	if _, err := os.Stat(cm.checkpointFile); os.IsNotExist(err) {
		return nil
	}

	timestamp := time.Now().Format("20060102_150405.000")
	archiveFile := filepath.Join(cm.historyDir, fmt.Sprintf("checkpoint_%s.json", timestamp))

	if err := os.Rename(cm.checkpointFile, archiveFile); err != nil {
		return fmt.Errorf("failed to archive checkpoint: %w", err)
	}

	// Clean up old archives
	cm.cleanupOldArchives()

	return nil
}

// cleanupOldArchives removes old checkpoint archives keeping only maxHistory.
func (cm *CheckpointManager) cleanupOldArchives() {
	entries, err := os.ReadDir(cm.historyDir)
	if err != nil {
		return
	}

	if len(entries) <= cm.maxHistory {
		return
	}

	// Sort by name (which includes timestamp) and remove oldest
	// Since we use timestamp in filename, lexical sort works
	for i := 0; i < len(entries)-cm.maxHistory; i++ {
		path := filepath.Join(cm.historyDir, entries[i].Name())
		os.Remove(path)
	}
}

// saveAtomic performs an atomic save of the checkpoint.
// Must be called with the lock held.
func (cm *CheckpointManager) saveAtomic() error {
	data, err := json.MarshalIndent(cm.checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	return fileutil.WriteFileAtomic(cm.checkpointFile, data, 0o600)
}

// load loads the checkpoint from disk.
func (cm *CheckpointManager) load() error {
	data, err := os.ReadFile(cm.checkpointFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to read checkpoint file: %w", err)
	}

	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	// Validate version
	if checkpoint.Version > CheckpointVersion {
		return fmt.Errorf("checkpoint version %d is newer than supported version %d",
			checkpoint.Version, CheckpointVersion)
	}

	cm.checkpoint = &checkpoint
	return nil
}
