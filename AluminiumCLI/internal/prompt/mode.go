package prompt

import (
	"os"

	"github.com/PandaTwoxx/Aluminium/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// IsInteractive returns true when interactive prompts should be used.
// Priority: --no-interactive > --interactive > config.interactive (all require a TTY).
func IsInteractive(cmd *cobra.Command, cfg *config.Config) bool {
	if !isTerminal() {
		return false
	}
	root := cmd.Root()
	if root != nil {
		if noInteractive, _ := root.PersistentFlags().GetBool("no-interactive"); noInteractive {
			return false
		}
		if interactive, _ := root.PersistentFlags().GetBool("interactive"); interactive {
			return true
		}
	}
	if cfg != nil && cfg.Interactive {
		return true
	}
	return false
}

// FromConfig returns true when interactive mode is enabled in config and stdin is a TTY.
func FromConfig(cfg *config.Config) bool {
	return cfg != nil && cfg.Interactive && isTerminal()
}

func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
