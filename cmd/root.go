package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "autocommit",
	Short: "A CLI tool to help you write better git commits",
	Long:  `autocommit walks you through your changes and helps you write a properly formatted conventional commit message.`,
	Run: func(cmd *cobra.Command, args []string) {
		files, fileMap, err := getFiles()
		if err != nil {
			fmt.Println("Error: are you inside a git repository?")
			return
		}
		if len(files) == 0 {
			fmt.Println("No file changes found.")
			return
		}

		m := setInitialModel(files, fileMap)

		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Println("Error starting UI:", err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
