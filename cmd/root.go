package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "terralink",
	Short: "A CLI tool to seamlessly link local Terraform modules for development.",
	Long: `terralink improves the Terraform development workflow by allowing you
to quickly swap remote module registry sources with local file paths.

It uses a simple annotation in your .tf files to manage the state,
keeping your configuration clean and readable.`,
}
var scanDir string

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	rootCmd.Flags().StringVar(&scanDir, "dir", ".", "Directory to scan for .tf files")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
