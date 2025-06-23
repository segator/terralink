package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var (
	rootCmd = &cobra.Command{
		Use:   "terralink",
		Short: "A CLI tool to seamlessly link local Terraform modules for development.",
		Long: `terralink improves the Terraform development workflow by allowing you
to quickly swap remote module registry sources with local file paths.

It uses a simple annotation in your .tf files to manage the state,
keeping your configuration clean and readable.`,
	}

	scanDir    string
	ignoreFile string
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, err = fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		if err != nil {
			log.Panic(err)
		}

		os.Exit(1)
	}
}

func commonFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&scanDir, "dir", ".", "Directory to scan for .tf files")
	cmd.Flags().StringVar(&ignoreFile, "terralinkignore", ".", ".terralinkignore dir path")
}
