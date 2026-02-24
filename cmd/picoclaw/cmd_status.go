// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"fmt"
	"os"

	"github.com/sipeed/picoclaw/pkg/auth"
)

func statusCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	configPath := getConfigPath()

	fmt.Printf("%s picoclaw Status\n", logo)
	fmt.Printf("Version: %s\n", formatVersion())
	build, _ := formatBuildInfo()
	if build != "" {
		fmt.Printf("Build: %s\n", build)
	}
	fmt.Println()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("Config:", configPath, "✓")
	} else {
		fmt.Println("Config:", configPath, "✗")
	}

	workspace := cfg.WorkspacePath()
	if _, err := os.Stat(workspace); err == nil {
		fmt.Println("Workspace:", workspace, "✓")
	} else {
		fmt.Println("Workspace:", workspace, "✗")
	}

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Model: %s\n", cfg.Agents.Defaults.GetModelName())

		// Check if model_list is configured
		hasModelList := len(cfg.ModelList) > 0

		if hasModelList {
			// Show configured models from model_list
			fmt.Println("\nConfigured Models:")
			for _, model := range cfg.ModelList {
				hasKey := model.APIKey != ""
				hasOAuth := model.AuthMethod == "oauth"
				isLocal := model.Model == "ollama/" || model.Model == "vllm/" ||
				          model.APIBase == "http://localhost" || model.APIBase == "http://127.0.0.1"

				authStatus := "no auth"
				if hasKey {
					authStatus = "api key"
				} else if hasOAuth {
					authStatus = "oauth"
				} else if isLocal {
					authStatus = "local"
				}

				// Extract protocol from model field (e.g., "openrouter/free" -> "openrouter")
				protocol := "unknown"
				if idx := indexOfSlash(model.Model); idx != -1 {
					protocol = model.Model[:idx]
				}

				fmt.Printf("  • %s (%s) [%s]\n", model.ModelName, protocol, authStatus)
			}
		}

		// Check legacy providers (for backward compatibility)
		hasLegacyProviders := cfg.HasProvidersConfig()

		if hasLegacyProviders {
			fmt.Println("\nLegacy Provider APIs:")

			hasOpenRouter := cfg.Providers.OpenRouter.APIKey != ""
			hasAnthropic := cfg.Providers.Anthropic.APIKey != ""
			hasOpenAI := cfg.Providers.OpenAI.APIKey != ""
			hasGemini := cfg.Providers.Gemini.APIKey != ""
			hasZhipu := cfg.Providers.Zhipu.APIKey != ""
			hasQwen := cfg.Providers.Qwen.APIKey != ""
			hasGroq := cfg.Providers.Groq.APIKey != ""
			hasVLLM := cfg.Providers.VLLM.APIBase != ""
			hasMoonshot := cfg.Providers.Moonshot.APIKey != ""
			hasDeepSeek := cfg.Providers.DeepSeek.APIKey != ""
			hasVolcEngine := cfg.Providers.VolcEngine.APIKey != ""
			hasNvidia := cfg.Providers.Nvidia.APIKey != ""
			hasOllama := cfg.Providers.Ollama.APIBase != ""

			status := func(enabled bool) string {
				if enabled {
					return "✓"
				}
				return "not set"
			}
			fmt.Println("  OpenRouter API:", status(hasOpenRouter))
			fmt.Println("  Anthropic API:", status(hasAnthropic))
			fmt.Println("  OpenAI API:", status(hasOpenAI))
			fmt.Println("  Gemini API:", status(hasGemini))
			fmt.Println("  Zhipu API:", status(hasZhipu))
			fmt.Println("  Qwen API:", status(hasQwen))
			fmt.Println("  Groq API:", status(hasGroq))
			fmt.Println("  Moonshot API:", status(hasMoonshot))
			fmt.Println("  DeepSeek API:", status(hasDeepSeek))
			fmt.Println("  VolcEngine API:", status(hasVolcEngine))
			fmt.Println("  Nvidia API:", status(hasNvidia))
			if hasVLLM {
				fmt.Printf("  vLLM/Local: ✓ %s\n", cfg.Providers.VLLM.APIBase)
			} else {
				fmt.Println("  vLLM/Local: not set")
			}
			if hasOllama {
				fmt.Printf("  Ollama: ✓ %s\n", cfg.Providers.Ollama.APIBase)
			} else {
				fmt.Println("  Ollama: not set")
			}
		}

		// Show OAuth/Token auth status
		store, _ := auth.LoadStore()
		if store != nil && len(store.Credentials) > 0 {
			fmt.Println("\nOAuth/Token Auth:")
			for provider, cred := range store.Credentials {
				status := "authenticated"
				if cred.IsExpired() {
					status = "expired"
				} else if cred.NeedsRefresh() {
					status = "needs refresh"
				}
				fmt.Printf("  %s (%s): %s\n", provider, cred.AuthMethod, status)
			}
		}
	}
}

// indexOfSlash finds the last '/' in a string.
// Used to extract protocol from model identifiers like "openrouter/free".
func indexOfSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}
