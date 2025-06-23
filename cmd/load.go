package cmd

import (
	"log"
	"terralink/internal/linker"

	"github.com/spf13/cobra"
)

var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "Link modules to local paths for development.",
	Long: `The 'dev' command scans .tf files for modules with a 'terralink:path' annotation.
It replaces the remote 'source' with the local path and saves the original state
in a temporary 'terralink-state' comment for later restoration.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Linking local modules for DEV mode...")
		_, err := linker.DevLoad(scanDir)
		if err != nil {
			log.Fatalf("Error running in dev mode: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(loadCmd)
}
