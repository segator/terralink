package cmd

import (
	"fmt"
	"log"
	"os"
	"terralink/internal/linker"

	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify that no modules are in dev mode.",
	Long: `The 'check' command scans for any active 'terralink-state' annotations.
If any are found, it lists the linked modules and exits with a non-zero status code.
This is useful in pre-commit hooks to prevent committing dev configurations.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Checking for active dev links...")
		activeDevLoadModules, err := linker.Check(scanDir)
		if err != nil {
			log.Fatalf("Error during check: %v", err)
		}

		if len(activeDevLoadModules) > 0 {
			fmt.Fprintln(os.Stderr, "\n❌ Error: Found Loaded Dev modules")
			for _, loadedModules := range activeDevLoadModules {
				fmt.Fprintf(os.Stderr, "  - Module '%s' is loaded.\n", loadedModules)
			}
			fmt.Fprintln(os.Stderr, "\nRun 'terralink prod' to fix this.")
			os.Exit(1)
		}

		log.Println("✅ Success! All modules are configured for production.")
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
