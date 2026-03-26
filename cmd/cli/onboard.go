package main

import (
	onboardingfeature "tango/cmd/cli/features/onboarding"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Launch an interactive onboarding wizard",
	RunE: func(cmd *cobra.Command, args []string) error {
		program := tea.NewProgram(onboardingfeature.NewModel(), tea.WithAltScreen())
		if _, err := program.Run(); err != nil {
			return err
		}
		return nil
	},
}
