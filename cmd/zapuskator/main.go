package main

import (
	"github.com/spf13/cobra"
)

func main() {
	err := rootCmd.Execute()
	cobra.CheckErr(err)
}
