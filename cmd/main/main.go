package main

import (
	"archivist/cmd/main/commands"
	"fmt"
	"os"
)

func main() {
	rootCmd := commands.NewRootCommand()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
