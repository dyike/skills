package genimage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/crosszan/modu/pkg/env"
	genimagerepo "github.com/crosszan/modu/repos/gen_image_repo"
	genimagevo "github.com/crosszan/modu/vo/gen_image_vo"
	"github.com/spf13/cobra"
)

var cfg struct {
	baseURL    string
	apiKey     string
	output     string
	system     string
	promptFile string
	envFile    string
}

// NewCmd creates the generate-image command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-image [prompt]",
		Short: "Generate images using Gemini API",
		Long:  "Generate images using Gemini API. Prompt can be text or @filepath to read from file.",
		RunE:  runGenerate,
	}

	cmd.Flags().StringVarP(&cfg.baseURL, "base-url", "b", "", "API base URL (or set GEMINI_BASE_URL env)")
	cmd.Flags().StringVarP(&cfg.apiKey, "api-key", "k", "", "API key (or set GEMINI_API_KEY env)")
	cmd.Flags().StringVarP(&cfg.output, "output", "o", "generated_image.jpg", "Output file path")
	cmd.Flags().StringVarP(&cfg.system, "system", "s", "", "System prompt (optional)")
	cmd.Flags().StringVarP(&cfg.promptFile, "prompt-file", "p", "", "Read prompt from file")
	cmd.Flags().StringVarP(&cfg.envFile, "env-file", "e", ".env", "Environment file to load")

	return cmd
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Load env file using modu/pkg/env
	env.Load(env.WithFile(cfg.envFile))

	// Get prompt from flag, file, or args
	var prompt string
	if cfg.promptFile != "" {
		data, err := os.ReadFile(cfg.promptFile)
		if err != nil {
			return fmt.Errorf("failed to read prompt file: %w", err)
		}
		prompt = string(data)
	} else if len(args) > 0 {
		prompt = args[0]
		// Support @filepath syntax
		if len(prompt) > 1 && prompt[0] == '@' {
			data, err := os.ReadFile(prompt[1:])
			if err != nil {
				return fmt.Errorf("failed to read prompt file: %w", err)
			}
			prompt = string(data)
		}
	} else {
		return fmt.Errorf("prompt required: provide as argument, use @filepath, or --prompt-file")
	}

	// Get API key from flag or env
	apiKey := cfg.apiKey
	if apiKey == "" {
		apiKey = env.Get("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("API key required: use --api-key or set GEMINI_API_KEY in .env")
	}

	// Get base URL from flag or env
	baseURL := cfg.baseURL
	if baseURL == "" {
		baseURL = env.GetDefault("GEMINI_BASE_URL", "http://127.0.0.1:8045")
	}

	fmt.Fprintf(os.Stderr, "Generating image...\n")
	fmt.Fprintf(os.Stderr, "Prompt: %s\n", prompt)
	if cfg.system != "" {
		fmt.Fprintf(os.Stderr, "System: %s\n", cfg.system)
	}

	// Use modu's gen_image_repo
	repo := genimagerepo.NewGeminiImageImpl(baseURL, apiKey)

	resp, err := repo.Generate(context.Background(), &genimagevo.GenImageRequest{
		UserPrompt:   prompt,
		SystemPrompt: cfg.system,
	})
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	// Print usage info
	fmt.Fprintf(os.Stderr, "\n✓ Generated successfully!\n")
	fmt.Fprintf(os.Stderr, "Provider: %s\n", resp.ProviderName)
	fmt.Fprintf(os.Stderr, "Model: %s\n", resp.Model)
	if resp.Usage != nil {
		fmt.Fprintf(os.Stderr, "Tokens: %d prompt, %d total\n", resp.Usage.PromptTokens, resp.Usage.TotalTokens)
	}

	// Save images
	if len(resp.Images) == 0 {
		return fmt.Errorf("no images generated")
	}

	for i, img := range resp.Images {
		filename := cfg.output
		if len(resp.Images) > 1 {
			ext := filepath.Ext(cfg.output)
			base := cfg.output[:len(cfg.output)-len(ext)]
			filename = fmt.Sprintf("%s_%d%s", base, i+1, ext)
		}

		if err := os.WriteFile(filename, img.Data, 0644); err != nil {
			return fmt.Errorf("failed to save image: %w", err)
		}

		fileInfo, _ := os.Stat(filename)
		fmt.Fprintf(os.Stderr, "✓ Saved: %s (%.2f KB)\n", filename, float64(fileInfo.Size())/1024)
	}

	return nil
}
