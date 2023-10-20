package ensure

import (
    "fmt"

    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "ensure",
    Aliases: []string{"en"},
    Short:  "Ensure a KubeStellar object exists",
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("ENSURE")
    },
}

func init() {
    // add flags
}