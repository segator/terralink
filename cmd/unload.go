package cmd

import (
	"log"
	"terralink/internal/ignore"
	"terralink/internal/linker"

	"github.com/spf13/cobra"
)

// unloadCmd represents the prod command
var unloadCmd = &cobra.Command{
	Use:   "unload",
	Short: "unload local modules and restore remote sources",
	Long: `The 'unload' command restores modules to their original remote source.
It reads the state from the 'terralink-state' comment, reverts the changes,
and removes the temporary state comment, cleaning the file for production.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Unloading dev mode...")
		matcher, err := ignore.NewMatcher(ignoreFile)
		if err != nil {
			log.Fatalf("Error creating ignore matcher: %v", err)
		}
		l := linker.NewLinker(matcher)
		_, err = l.DevUnload(scanDir)
		if err != nil {
			log.Fatalf("Error running in reset mode: %v", err)
		}
	},
}

func init() {
	commonFlags(unloadCmd)
	rootCmd.AddCommand(unloadCmd)
}
