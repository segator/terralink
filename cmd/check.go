package cmd

import (
	"fmt"
	"log"
	"os"
	"terralink/internal/ignore"
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
		matcher, err := ignore.NewMatcher(ignoreFile)
		if err != nil {
			log.Fatalf("Error creating ignore matcher: %v", err)
		}
		l := linker.NewLinker(matcher)
		activeDevLoadModules, err := l.Check(scanDir)
		if err != nil {
			log.Fatalf("Error during check: %v", err)
		}

		if len(activeDevLoadModules) > 0 {
			_, err = fmt.Fprintln(os.Stderr, "\n❌ Error: Found Loaded Dev modules")
			if err != nil {
				log.Panic(err)
			}

			for _, loadedModules := range activeDevLoadModules {
				_, err = fmt.Fprintf(os.Stderr, "  - Module '%s' is loaded.\n", loadedModules)
				if err != nil {
					log.Panic(err)
				}

			}
			_, err = fmt.Fprintln(os.Stderr, "\nRun 'terralink unload' to fix this.")
			if err != nil {
				log.Panic(err)
			}

			os.Exit(1)
		}

		log.Println("✅ Success! All modules are configured for production.")
	},
}

func init() {
	commonFlags(checkCmd)
	rootCmd.AddCommand(checkCmd)
}
