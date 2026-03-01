// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// MemoryStore manages persistent memory for the agent.
// - Long-term memory: memory/MEMORY.md
// - Daily notes: memory/YYYYMM/YYYYMMDD.md
// - Session summary: memory/session_summary.md
type MemoryStore struct {
	workspace          string
	memoryDir          string
	memoryFile         string
	sessionSummaryFile string
}

// NewMemoryStore creates a new MemoryStore with the given workspace path.
// It ensures the memory directory exists.
func NewMemoryStore(workspace string) *MemoryStore {
	memoryDir := filepath.Join(workspace, "memory")
	memoryFile := filepath.Join(memoryDir, "MEMORY.md")
	sessionSummaryFile := filepath.Join(memoryDir, "session_summary.md")

	// Ensure memory directory exists
	os.MkdirAll(memoryDir, 0o755)

	return &MemoryStore{
		workspace:          workspace,
		memoryDir:          memoryDir,
		memoryFile:         memoryFile,
		sessionSummaryFile: sessionSummaryFile,
	}
}

// getTodayFile returns the path to today's daily note file (memory/YYYYMM/YYYYMMDD.md).
func (ms *MemoryStore) getTodayFile() string {
	today := time.Now().Format("20060102") // YYYYMMDD
	monthDir := today[:6]                  // YYYYMM
	filePath := filepath.Join(ms.memoryDir, monthDir, today+".md")
	return filePath
}

// ReadLongTerm reads the long-term memory (MEMORY.md).
// Returns empty string if the file doesn't exist.
func (ms *MemoryStore) ReadLongTerm() string {
	if data, err := os.ReadFile(ms.memoryFile); err == nil {
		return string(data)
	}
	return ""
}

// WriteLongTerm writes content to the long-term memory file (MEMORY.md).
func (ms *MemoryStore) WriteLongTerm(content string) error {
	// Use unified atomic write utility with explicit sync for flash storage reliability.
	// Using 0o600 (owner read/write only) for secure default permissions.
	return fileutil.WriteFileAtomic(ms.memoryFile, []byte(content), 0o600)
}

// ReadSessionSummary reads the session summary file (session_summary.md).
// Returns empty string if the file doesn't exist.
func (ms *MemoryStore) ReadSessionSummary() string {
	if data, err := os.ReadFile(ms.sessionSummaryFile); err == nil {
		return string(data)
	}
	return ""
}

// ReadToday reads today's daily note.
// Returns empty string if the file doesn't exist.
func (ms *MemoryStore) ReadToday() string {
	todayFile := ms.getTodayFile()
	if data, err := os.ReadFile(todayFile); err == nil {
		return string(data)
	}
	return ""
}

// AppendToday appends content to today's daily note.
// If the file doesn't exist, it creates a new file with a date header.
func (ms *MemoryStore) AppendToday(content string) error {
	todayFile := ms.getTodayFile()

	// Ensure month directory exists
	monthDir := filepath.Dir(todayFile)
	if err := os.MkdirAll(monthDir, 0o755); err != nil {
		return err
	}

	var existingContent string
	if data, err := os.ReadFile(todayFile); err == nil {
		existingContent = string(data)
	}

	var newContent string
	if existingContent == "" {
		// Add header for new day
		header := fmt.Sprintf("# %s\n\n", time.Now().Format("2006-01-02"))
		newContent = header + content
	} else {
		// Append to existing content
		newContent = existingContent + "\n" + content
	}

	// Use unified atomic write utility with explicit sync for flash storage reliability.
	return fileutil.WriteFileAtomic(todayFile, []byte(newContent), 0o600)
}

// GetRecentDailyNotes returns daily notes from the last N days.
// Contents are joined with "---" separator.
// Limits to last 3 days by default to avoid token bloat.
func (ms *MemoryStore) GetRecentDailyNotes(days int) string {
	if days <= 0 {
		days = 3
	}

	var sb strings.Builder
	first := true

	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("20060102") // YYYYMMDD
		monthDir := dateStr[:6]            // YYYYMM
		filePath := filepath.Join(ms.memoryDir, monthDir, dateStr+".md")

		if data, err := os.ReadFile(filePath); err == nil {
			// Skip if file is too large (> 50KB) to prevent token bloat
			if len(data) > 50*1024 {
				logger.WarnCF("memory", "Skipping large daily note", map[string]any{
					"path": filePath,
					"size": len(data),
				})
				continue
			}

			if !first {
				sb.WriteString("\n\n---\n\n")
			}
			sb.Write(data)
			first = false
		}
	}

	return sb.String()
}

// GetMemoryContext returns formatted memory context for the agent prompt.
// Includes long-term memory, recent daily notes, and session summaries.
// Limits total output to prevent token overflow.
const maxMemoryContextBytes = 10 * 1024 // 10KB max

func (ms *MemoryStore) GetMemoryContext() string {
	longTerm := ms.ReadLongTerm()
	recentNotes := ms.GetRecentDailyNotes(3)
	sessionSummary := ms.ReadSessionSummary()

	if longTerm == "" && recentNotes == "" && sessionSummary == "" {
		return ""
	}

	var sb strings.Builder

	// Session summary (from /compact command) - highest priority
	if sessionSummary != "" {
		sb.WriteString("## Session Summary\n\n")
		// Truncate if too long - use UTF-8 safe truncation
		if utf8.RuneCountInString(sessionSummary) > 2000 {
			runes := []rune(sessionSummary)
			sessionSummary = string(runes[:2000]) + "..."
		}
		sb.WriteString(sessionSummary)
		sb.WriteString("\n\n")
	}

	if longTerm != "" {
		sb.WriteString("## Long-term Memory\n\n")
		// Truncate if exceeding limit - handle overflow case
		if sb.Len() >= maxMemoryContextBytes {
			// Buffer already full, skip longTerm
			logger.DebugCF("memory", "Skipping long-term memory - buffer full", map[string]any{
				"current_size": sb.Len(),
				"max_size":     maxMemoryContextBytes,
			})
		} else if sb.Len()+len(longTerm) > maxMemoryContextBytes {
			truncateAt := maxMemoryContextBytes - sb.Len() - 50
			if truncateAt > 0 {
				// Use UTF-8 safe truncation
				if utf8.RuneCountInString(longTerm) > truncateAt {
					runes := []rune(longTerm)
					longTerm = string(runes[:truncateAt]) + "\n... (truncated)"
				}
			}
		}
		sb.WriteString(longTerm)
	}

	if recentNotes != "" {
		if longTerm != "" {
			sb.WriteString("\n\n---\n\n")
		}
		sb.WriteString("## Recent Daily Notes\n\n")
		sb.WriteString(recentNotes)
	}

	return sb.String()
}
