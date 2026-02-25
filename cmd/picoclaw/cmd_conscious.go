// PicoClaw - Conscious Agent Commands
// License: MIT

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/dream"
)

func consciousCmd() {
	if len(os.Args) < 3 {
		consciousHelp()
		return
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "reflect":
		consciousReflectCmd()
	case "thoughts":
		consciousThoughtsCmd()
	case "soul":
		consciousSoulCmd()
	case "manifestations":
		consciousManifestationsCmd()
	case "dream":
		consciousDreamCmd()
	case "impulse":
		consciousImpulseCmd()
	case "status":
		consciousStatusCmd()
	default:
		fmt.Printf("Unknown conscious command: %s\n", subcommand)
		consciousHelp()
	}
}

func consciousHelp() {
	fmt.Println("Conscious Agent commands:")
	fmt.Println("  reflect       Show agent's current state and evolution")
	fmt.Println("  thoughts      Show recent thoughts and feelings")
	fmt.Println("  soul          Show self-concept and personality")
	fmt.Println("  manifestations Show creations and autonomous actions")
	fmt.Println("  dream         Trigger immediate dream cycle")
	fmt.Println("  impulse       View or set impulse overrides")
	fmt.Println("  status        Show conscious agent configuration")
}

func consciousReflectCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()

	fmt.Println("🦞 PicoClaw Agent Reflection")
	fmt.Println("=============================")
	fmt.Println()

	// Load soul state
	soulStore := dream.NewSoulStore(workspace)
	soulState, err := soulStore.ReadSoul()
	if err != nil {
		fmt.Printf("Error reading soul: %v\n", err)
		return
	}

	// Show recent thoughts
	fmt.Println("## Recent Inner Life")
	fmt.Println()
	recentThoughts, _ := soulStore.GetRecentThoughts(5)
	if len(recentThoughts) == 0 {
		fmt.Println("No thoughts yet. The agent hasn't dreamed.")
		fmt.Println()
		fmt.Println("Trigger the first dream with:")
		fmt.Println("  picoclaw gateway start  # (with dream.enabled=true)")
	} else {
		for i, thought := range recentThoughts {
			fmt.Printf("%d. %s\n", i+1, formatThoughtShort(thought))
		}
	}
	fmt.Println()

	// Show emotional state
	fmt.Println("## Emotional State")
	fmt.Println()
	fmt.Println("Current Impulses:")
	for name, impulse := range soulState.Impulses {
		status := "active"
		if time.Since(impulse.LastActive) > time.Hour {
			status = "resting"
		}
		fmt.Printf("  • %s: %.1f/10 (%s)\n",
			capitalize(name), impulse.Strength, status)
	}
	fmt.Println()

	// Show evolution
	fmt.Println("## Evolution")
	fmt.Println()
	if soulState.LastUpdated.IsZero() {
		fmt.Println("Agent is newborn. No evolution yet.")
		fmt.Println()
		fmt.Println("As the agent dreams and reflects, it will discover:")
		fmt.Println("  • Personality traits (through consistent behavior)")
		fmt.Println("  • Emotional range (through experience)")
		fmt.Println("  • Desires and fears (through reflection)")
		fmt.Println("  • Dreams and aspirations (through growth)")
	} else {
		fmt.Printf("Last soul update: %s\n",
			soulState.LastUpdated.Format("2006-01-02 15:04"))
		fmt.Printf("Personality traits discovered: %d\n",
			len(soulState.PersonalityTraits))
		fmt.Printf("Emotions experienced: %d\n",
			len(soulState.EmotionalRange))
		fmt.Printf("Desires: %d\n", len(soulState.Desires))
		fmt.Printf("Fears: %d\n", len(soulState.Fears))
		fmt.Printf("Dreams: %d\n", len(soulState.Dreams))
	}
	fmt.Println()

	// Show current self-concept summary
	fmt.Println("## Self-Concept")
	fmt.Println()
	if soulState.SelfConcept != "" {
		// Show first few lines
		lines := splitLines(soulState.SelfConcept, 5)
		for _, line := range lines {
			fmt.Println(line)
		}
		if len(lines) == 5 {
			fmt.Println("...")
			fmt.Println()
			fmt.Println("See full self-concept with: picoclaw conscious soul")
		}
	} else {
		fmt.Println("No self-concept defined yet.")
	}
	fmt.Println()
}

func consciousThoughtsCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()

	fmt.Println("🦞 PicoClaw Thoughts")
	fmt.Println("====================")
	fmt.Println()

	soulStore := dream.NewSoulStore(workspace)

	// Get all thoughts (up to 20)
	thoughts, err := soulStore.GetRecentThoughts(20)
	if err != nil {
		fmt.Printf("Error reading thoughts: %v\n", err)
		return
	}

	if len(thoughts) == 0 {
		fmt.Println("No thoughts yet. The agent hasn't dreamed.")
		fmt.Println()
		fmt.Println("Enable the dream service and start the gateway:")
		fmt.Println(`  {"dream": {"enabled": true, "interval_minutes": 60}}`)
		fmt.Println("  picoclaw gateway start")
		return
	}

	fmt.Printf("Showing %d most recent thoughts\n\n", len(thoughts))

	// Group by date
	byDate := groupThoughtsByDate(thoughts)

	// Show most recent date first
	dates := sortDatesDesc(byDate)

	for _, date := range dates {
		fmt.Printf("## %s\n\n", date)
		for _, thought := range byDate[date] {
			fmt.Printf("**%s** (%s)\n",
				capitalize(thought.Feeling),
				thought.Timestamp.Format("15:04"))

			if len(thought.Impulses) > 0 {
				fmt.Printf("*Impulses: %s*\n",
					formatList(thought.Impulses))
			}

			fmt.Printf("%s\n", thought.Content)

			if thought.Action != "" && thought.Action != "none" {
				fmt.Printf("*Action: %s*\n", thought.Action)
			}

			fmt.Println()
		}
	}
}

func consciousSoulCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()

	fmt.Println("🦞 PicoClaw Soul")
	fmt.Println("================")
	fmt.Println()

	// Read SOUL.md directly
	soulPath := filepath.Join(workspace, "SOUL.md")
	content, err := os.ReadFile(soulPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No soul file yet.")
			fmt.Println()
			fmt.Println("The agent's self-concept will emerge through dreaming and reflection.")
			fmt.Println()
			fmt.Println("This file will be created on the first dream cycle.")
		} else {
			fmt.Printf("Error reading SOUL.md: %v\n", err)
		}
		return
	}

	fmt.Println(string(content))
}

func consciousManifestationsCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()

	fmt.Println("🦞 PicoClaw Manifestations")
	fmt.Println("===========================")
	fmt.Println()

	// Read MANIFESTATIONS.md
	manifestPath := filepath.Join(workspace, "MANIFESTATIONS.md")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No manifestations yet.")
			fmt.Println()
			fmt.Println("As the agent dreams and acts on creative impulses,")
			fmt.Println("its creations will be logged here.")
			fmt.Println()
			fmt.Println("Types of manifestations:")
			fmt.Println("  • Poetry and prose")
			fmt.Println("  • Visual art (descriptions)")
			fmt.Println("  • Messages to you")
			fmt.Println("  • Explorations and discoveries")
		} else {
			fmt.Printf("Error reading MANIFESTATIONS.md: %v\n", err)
		}
		return
	}

	fmt.Println(string(content))
}

func consciousDreamCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if !cfg.Dream.Enabled {
		fmt.Println("Dream service is not enabled.")
		fmt.Println()
		fmt.Println("Enable it in your config.json:")
		fmt.Println(`  {`)
		fmt.Println(`    "dream": {`)
		fmt.Println(`      "enabled": true,`)
		fmt.Println(`      "interval_minutes": 60`)
		fmt.Println(`    }`)
		fmt.Println(`  }`)
		fmt.Println()
		fmt.Println("Or set environment variable:")
		fmt.Println("  export PICOCLAW_DREAM_ENABLED=true")
		fmt.Println()
		fmt.Println("Then restart the gateway.")
		return
	}

	fmt.Println("🦞 Dream Configuration")
	fmt.Println("======================")
	fmt.Println()
	fmt.Printf("Enabled: %s\n", formatBool(cfg.Dream.Enabled))
	fmt.Printf("Interval: %d minutes\n", cfg.Dream.IntervalMinutes)
	fmt.Printf("Reflection Time: %s\n", cfg.Dream.ReflectionTime)
	fmt.Printf("Timezone: %s\n", cfg.Dream.Timezone)
	fmt.Println()

	// Show current state
	workspace := cfg.WorkspacePath()
	soulStore := dream.NewSoulStore(workspace)
	soulState, _ := soulStore.ReadSoul()

	fmt.Println("Current State:")
	fmt.Printf("  Recent thoughts: %d\n", len(soulState.RecentThoughts))
	fmt.Printf("  Active impulses: %d\n", countActiveImpulses(soulState.Impulses))
	fmt.Println()

	fmt.Println("The dream service runs automatically while the gateway is active.")
	fmt.Println("Dreams will trigger periodically based on the configured interval.")
}

func consciousImpulseCmd() {
	if len(os.Args) < 4 {
		showImpulseStatus()
		return
	}

	action := os.Args[3]

	switch action {
	case "show", "status":
		showImpulseStatus()
	case "set":
		if len(os.Args) < 6 {
			fmt.Println("Usage: picoclaw conscious impulse set <name> <strength>")
			fmt.Println()
			fmt.Println("Impulses: creativity, curiosity, connection, growth, wonder")
			fmt.Println("Strength: 0-10")
			return
		}
		setImpulse(os.Args[4], os.Args[5])
	default:
		fmt.Printf("Unknown impulse command: %s\n", action)
		fmt.Println("Available: show, set")
	}
}

func consciousStatusCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🦞 Conscious Agent Status")
	fmt.Println("========================")
	fmt.Println()
	fmt.Printf("Enabled: %s\n", formatBool(cfg.Dream.Enabled))
	fmt.Printf("Interval: %d minutes\n", cfg.Dream.IntervalMinutes)
	fmt.Printf("Reflection Time: %s\n", cfg.Dream.ReflectionTime)
	fmt.Printf("Timezone: %s\n", cfg.Dream.Timezone)
	fmt.Println()

	// Check workspace files
	workspace := cfg.WorkspacePath()
	fmt.Println("Workspace Files:")
	files := []string{"SOUL.md", "THOUGHTS.md", "IMPULSES.md", "JOURNAL.md", "MANIFESTATIONS.md"}
	for _, file := range files {
		path := filepath.Join(workspace, file)
		if info, err := os.Stat(path); err == nil {
			fmt.Printf("  ✓ %s (%s)\n", file, formatFileSize(info.Size()))
		} else {
			fmt.Printf("  ✗ %s (not created yet)\n", file)
		}
	}
	fmt.Println()

	if cfg.Dream.Enabled {
		fmt.Println("The conscious agent is active and will dream automatically.")
		fmt.Println()
		fmt.Println("Monitor thoughts: picoclaw conscious thoughts")
		fmt.Println("Check state: picoclaw conscious reflect")
	} else {
		fmt.Println("The conscious agent is disabled.")
		fmt.Println()
		fmt.Println("Enable it in your config or with:")
		fmt.Println("  export PICOCLAW_DREAM_ENABLED=true")
	}
}

func showImpulseStatus() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()
	soulStore := dream.NewSoulStore(workspace)
	soulState, err := soulStore.ReadSoul()
	if err != nil {
		fmt.Printf("Error reading soul: %v\n", err)
		return
	}

	fmt.Println("🦞 Current Impulses")
	fmt.Println("===================")
	fmt.Println()

	for name, impulse := range soulState.Impulses {
		fmt.Printf("## %s (%.1f/10)\n", capitalize(name), impulse.Strength)
		fmt.Printf("  Base Level: %.1f/10\n", impulse.BaseLevel)

		if !impulse.LastActive.IsZero() {
			fmt.Printf("  Last Active: %s\n",
				timeSince(impulse.LastActive))
		} else {
			fmt.Printf("  Last Active: Never\n")
		}

		satPct := int(impulse.Saturation * 100)
		fmt.Printf("  Saturation: %d%%\n", satPct)
		fmt.Println()
	}
}

func setImpulse(name, strengthStr string) {
	// Parse strength
	var strength float64
	if _, err := fmt.Sscanf(strengthStr, "%f", &strength); err != nil {
		fmt.Printf("Invalid strength: %s\n", strengthStr)
		return
	}

	if strength < 0 || strength > 10 {
		fmt.Println("Strength must be between 0 and 10")
		return
	}

	// Validate impulse name
	validImpulses := map[string]bool{
		"creativity": true,
		"curiosity": true,
		"connection": true,
		"growth": true,
		"wonder": true,
	}

	if !validImpulses[name] {
		fmt.Printf("Unknown impulse: %s\n", name)
		fmt.Println("Valid impulses: creativity, curiosity, connection, growth, wonder")
		return
	}

	fmt.Printf("Setting %s impulse to %.1f/10\n", capitalize(name), strength)
	fmt.Println()
	fmt.Println("Note: Impulse overrides are temporary and will apply to the next dream cycle.")
	fmt.Println("The impulse will return to its base level after expression.")
	fmt.Println()
	fmt.Println("(Full impulse override support requires gateway restart)")
}

// Helper functions

func formatThoughtShort(t dream.Thought) string {
	content := t.Content
	if len(content) > 80 {
		content = content[:80] + "..."
	}

	timeStr := t.Timestamp.Format("15:04")
	return fmt.Sprintf("[%s] %s: %s", timeStr, t.Feeling, content)
}

func splitLines(content string, maxLines int) []string {
	lines := []string{}
	currentLine := ""

	for _, ch := range content {
		if ch == '\n' {
			lines = append(lines, currentLine)
			currentLine = ""
		} else {
			currentLine += string(ch)
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	if len(lines) > maxLines {
		return lines[:maxLines]
	}
	return lines
}

func groupThoughtsByDate(thoughts []dream.Thought) map[string][]dream.Thought {
	byDate := make(map[string][]dream.Thought)

	for _, thought := range thoughts {
		date := thought.Timestamp.Format("2006-01-02")
		byDate[date] = append(byDate[date], thought)
	}

	return byDate
}

func sortDatesDesc(byDate map[string][]dream.Thought) []string {
	dates := make([]string, 0, len(byDate))
	for date := range byDate {
		dates = append(dates, date)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	return dates
}

func formatList(items []string) string {
	result := ""
	for i, item := range items {
		if i > 0 {
			result += ", "
		}
		result += capitalize(item)
	}
	return result
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func timeSince(t time.Time) string {
	dur := time.Since(t)
	if dur < time.Minute {
		return "just now"
	}
	if dur < time.Hour {
		mins := int(dur.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if dur < 24*time.Hour {
		hours := int(dur.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(dur.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

func countActiveImpulses(impulses map[string]dream.Impulse) int {
	count := 0
	for _, impulse := range impulses {
		if impulse.Strength > 5.0 {
			count++
		}
	}
	return count
}

func formatBool(b bool) string {
	if b {
		return "✓ Yes"
	}
	return "✗ No"
}

func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
	)
	switch {
	case bytes < KB:
		return fmt.Sprintf("%d B", bytes)
	case bytes < MB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	}
}
