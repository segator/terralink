package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Build information variables - will be populated by ldflags
var (
	Version    = "dev"
	Commit     = "none"
	BuildDate  = "unknown"
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print terralink version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Terralink Version: %s\n", Version)
			fmt.Printf("Git Commit: %s\n", Commit)
			fmt.Printf("Build Date: %s\n", BuildDate)
			fmt.Printf("Go Version: %s\n", runtime.Version())
			fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
}
