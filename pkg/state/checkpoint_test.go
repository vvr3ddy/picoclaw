package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestNewCheckpointManager(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(string) error
		wantDirty bool
	}{
		{
			name:      "creates fresh manager with no existing checkpoint",
			setupFunc: nil,
			wantDirty: false,
		},
		{
			name: "loads existing clean checkpoint",
			setupFunc: func(dir string) error {
				cp := &Checkpoint{
					Version:   CheckpointVersion,
					State:     CheckpointStateClean,
					Timestamp: time.Now(),
					SessionID: "test-session",
				}
				data, _ := json.Marshal(cp)
				stateDir := filepath.Join(dir, "state")
				os.MkdirAll(stateDir, 0o755)
				return os.WriteFile(filepath.Join(stateDir, "checkpoint.json"), data, 0o600)
			},
			wantDirty: false,
		},
		{
			name: "loads existing dirty checkpoint",
			setupFunc: func(dir string) error {
				cp := &Checkpoint{
					Version:   CheckpointVersion,
					State:     CheckpointStateDirty,
					Timestamp: time.Now(),
					SessionID: "test-session",
				}
				data, _ := json.Marshal(cp)
				stateDir := filepath.Join(dir, "state")
				os.MkdirAll(stateDir, 0o755)
				return os.WriteFile(filepath.Join(stateDir, "checkpoint.json"), data, 0o600)
			},
			wantDirty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tt.setupFunc != nil {
				if err := tt.setupFunc(tmpDir); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			cm := NewCheckpointManager(tmpDir)
			if cm == nil {
				t.Fatal("NewCheckpointManager returned nil")
			}

			if got := cm.IsRecoveryNeeded(); got != tt.wantDirty {
				t.Errorf("IsRecoveryNeeded() = %v, want %v", got, tt.wantDirty)
			}
		})
	}
}

func TestCheckpointManager_SaveDirty(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCheckpointManager(tmpDir)

	messages := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}

	tool := &ToolExecution{
		ID:        "tool-1",
		Name:      "test_tool",
		Arguments: json.RawMessage(`{"arg": "value"}`),
		StartTime: time.Now(),
	}

	err := cm.SaveDirty("session-123", messages, "agent-1", "telegram", "chat-456", "processing", tool)
	if err != nil {
		t.Fatalf("SaveDirty failed: %v", err)
	}

	// Verify checkpoint was saved
	if !cm.IsRecoveryNeeded() {
		t.Error("expected IsRecoveryNeeded() to be true after SaveDirty")
	}

	cp := cm.GetCheckpoint()
	if cp == nil {
		t.Fatal("GetCheckpoint returned nil")
	}

	if cp.SessionID != "session-123" {
		t.Errorf("SessionID = %q, want %q", cp.SessionID, "session-123")
	}

	if cp.State != CheckpointStateDirty {
		t.Errorf("State = %q, want %q", cp.State, CheckpointStateDirty)
	}

	if cp.CurrentTool == nil {
		t.Fatal("CurrentTool is nil")
	}

	if cp.CurrentTool.ID != "tool-1" {
		t.Errorf("CurrentTool.ID = %q, want %q", cp.CurrentTool.ID, "tool-1")
	}

	if len(cp.Messages) != 2 {
		t.Errorf("len(Messages) = %d, want 2", len(cp.Messages))
	}

	// Verify file exists
	checkpointFile := filepath.Join(tmpDir, "state", "checkpoint.json")
	if _, err := os.Stat(checkpointFile); os.IsNotExist(err) {
		t.Error("checkpoint file does not exist")
	}
}

func TestCheckpointManager_SaveClean(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCheckpointManager(tmpDir)

	// First save dirty
	messages := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}

	err := cm.SaveDirty("session-123", messages, "agent-1", "telegram", "chat-456", "processing", nil)
	if err != nil {
		t.Fatalf("SaveDirty failed: %v", err)
	}

	// Now save clean
	err = cm.SaveClean("session-123", messages, "agent-1", "telegram", "chat-456")
	if err != nil {
		t.Fatalf("SaveClean failed: %v", err)
	}

	if cm.IsRecoveryNeeded() {
		t.Error("expected IsRecoveryNeeded() to be false after SaveClean")
	}

	cp := cm.GetCheckpoint()
	if cp.State != CheckpointStateClean {
		t.Errorf("State = %q, want %q", cp.State, CheckpointStateClean)
	}

	if cp.CurrentTool != nil {
		t.Error("CurrentTool should be nil after SaveClean")
	}

	if cp.RetryCount != 0 {
		t.Errorf("RetryCount = %d, want 0", cp.RetryCount)
	}
}

func TestCheckpointManager_MarkClean(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCheckpointManager(tmpDir)

	// Save dirty first
	messages := []providers.Message{{Role: "user", Content: "test"}}
	err := cm.SaveDirty("session-1", messages, "agent-1", "cli", "direct", "test", nil)
	if err != nil {
		t.Fatalf("SaveDirty failed: %v", err)
	}

	// Mark clean
	err = cm.MarkClean()
	if err != nil {
		t.Fatalf("MarkClean failed: %v", err)
	}

	if cm.IsRecoveryNeeded() {
		t.Error("expected IsRecoveryNeeded() to be false after MarkClean")
	}
}

func TestCheckpointManager_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCheckpointManager(tmpDir)

	// Save a checkpoint
	messages := []providers.Message{{Role: "user", Content: "test"}}
	err := cm.SaveClean("session-1", messages, "agent-1", "cli", "direct")
	if err != nil {
		t.Fatalf("SaveClean failed: %v", err)
	}

	// Clear it
	err = cm.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// File should be gone
	checkpointFile := filepath.Join(tmpDir, "state", "checkpoint.json")
	if _, err := os.Stat(checkpointFile); !os.IsNotExist(err) {
		t.Error("checkpoint file should not exist after Clear")
	}

	// Checkpoint should be reset
	cp := cm.GetCheckpoint()
	if cp.State != CheckpointStateClean {
		t.Errorf("State = %q, want %q", cp.State, CheckpointStateClean)
	}
}

func TestCheckpointManager_GetMessages(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCheckpointManager(tmpDir)

	messages := []providers.Message{
		{
			Role:             "user",
			Content:          "hello",
			ReasoningContent: "user reasoning",
		},
		{
			Role:       "assistant",
			Content:    "hi",
			ToolCallID: "call-1",
		},
	}

	err := cm.SaveClean("session-1", messages, "agent-1", "cli", "direct")
	if err != nil {
		t.Fatalf("SaveClean failed: %v", err)
	}

	got := cm.GetMessages()
	if len(got) != 2 {
		t.Fatalf("len(GetMessages()) = %d, want 2", len(got))
	}

	if got[0].Role != "user" || got[0].Content != "hello" {
		t.Errorf("first message = %+v, want role=user content=hello", got[0])
	}

	if got[1].ToolCallID != "call-1" {
		t.Errorf("second message ToolCallID = %q, want call-1", got[1].ToolCallID)
	}
}

func TestCheckpointManager_GetCurrentTool(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCheckpointManager(tmpDir)

	// No tool initially
	if tool := cm.GetCurrentTool(); tool != nil {
		t.Error("GetCurrentTool should return nil when no tool is set")
	}

	tool := &ToolExecution{
		ID:        "tool-1",
		Name:      "test_tool",
		Arguments: json.RawMessage(`{"key": "value"}`),
		StartTime: time.Now(),
	}

	err := cm.SaveDirty("session-1", nil, "agent-1", "cli", "direct", "test", tool)
	if err != nil {
		t.Fatalf("SaveDirty failed: %v", err)
	}

	got := cm.GetCurrentTool()
	if got == nil {
		t.Fatal("GetCurrentTool returned nil")
	}

	if got.ID != "tool-1" {
		t.Errorf("ID = %q, want tool-1", got.ID)
	}

	if string(got.Arguments) != `{"key": "value"}` {
		t.Errorf("Arguments = %s, want {\"key\": \"value\"}", string(got.Arguments))
	}
}

func TestCheckpointManager_Archive(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCheckpointManager(tmpDir)
	cm.SetMaxHistory(3)

	// Save multiple checkpoints
	for i := 0; i < 5; i++ {
		messages := []providers.Message{{Role: "user", Content: "test"}}
		err := cm.SaveClean("session", messages, "agent", "cli", "direct")
		if err != nil {
			t.Fatalf("SaveClean iteration %d failed: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Check that archives exist
	historyDir := filepath.Join(tmpDir, "state", "history")
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		t.Fatalf("failed to read history dir: %v", err)
	}

	// Should have maxHistory archives (3)
	if len(entries) != 3 {
		t.Errorf("len(history entries) = %d, want 3", len(entries))
	}
}

func TestCheckpointManager_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCheckpointManager(tmpDir)

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 50

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				messages := []providers.Message{{Role: "user", Content: "test"}}
				err := cm.SaveClean("session", messages, "agent", "cli", "direct")
				if err != nil {
					t.Errorf("goroutine %d: SaveClean failed: %v", id, err)
				}
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = cm.IsRecoveryNeeded()
				_ = cm.GetCheckpoint()
				_ = cm.GetMessages()
			}
		}(i)
	}

	wg.Wait()
}

func TestCheckpointManager_VersionValidation(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")
	os.MkdirAll(stateDir, 0o755)

	// Create a checkpoint with future version
	cp := &Checkpoint{
		Version:   CheckpointVersion + 1,
		State:     CheckpointStateClean,
		Timestamp: time.Now(),
		SessionID: "test",
	}
	data, _ := json.Marshal(cp)
	checkpointFile := filepath.Join(stateDir, "checkpoint.json")
	os.WriteFile(checkpointFile, data, 0o600)

	// Loading should fail due to version mismatch
	cm := NewCheckpointManager(tmpDir)

	// The checkpoint should be empty/default since load failed
	cpLoaded := cm.GetCheckpoint()
	if cpLoaded.Version != CheckpointVersion {
		t.Errorf("Version = %d, want %d (default)", cpLoaded.Version, CheckpointVersion)
	}
}

func BenchmarkCheckpointManager_SaveClean(b *testing.B) {
	tmpDir := b.TempDir()
	cm := NewCheckpointManager(tmpDir)

	messages := make([]providers.Message, 10)
	for i := 0; i < 10; i++ {
		messages[i] = providers.Message{
			Role:    "user",
			Content: "This is a test message with some content",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := cm.SaveClean("session", messages, "agent", "cli", "direct")
		if err != nil {
			b.Fatalf("SaveClean failed: %v", err)
		}
	}
}

func BenchmarkCheckpointManager_GetMessages(b *testing.B) {
	tmpDir := b.TempDir()
	cm := NewCheckpointManager(tmpDir)

	messages := make([]providers.Message, 100)
	for i := 0; i < 100; i++ {
		messages[i] = providers.Message{
			Role:    "user",
			Content: "Test message content",
		}
	}

	err := cm.SaveClean("session", messages, "agent", "cli", "direct")
	if err != nil {
		b.Fatalf("SaveClean failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cm.GetMessages()
	}
}

func BenchmarkCheckpointManager_ConcurrentSave(b *testing.B) {
	tmpDir := b.TempDir()
	cm := NewCheckpointManager(tmpDir)

	messages := []providers.Message{{Role: "user", Content: "test"}}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := cm.SaveClean("session", messages, "agent", "cli", "direct")
			if err != nil {
				b.Fatalf("SaveClean failed: %v", err)
			}
		}
	})
}
