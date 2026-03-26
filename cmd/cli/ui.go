package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	rootBannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("117"))

	rootTagStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("31")).
			Padding(0, 1)

	rootMutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("109"))

	rootSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("159"))

	rootCommandStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("24")).
				Padding(0, 1)

	rootAccentStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("123"))
)

const tangoBanner = `
████████╗ █████╗ ███╗   ██╗ ██████╗  ██████╗
╚══██╔══╝██╔══██╗████╗  ██║██╔════╝ ██╔═══██╗
   ██║   ███████║██╔██╗ ██║██║  ███╗██║   ██║
   ██║   ██╔══██║██║╚██╗██║██║   ██║██║   ██║
   ██║   ██║  ██║██║ ╚████║╚██████╔╝╚██████╔╝
   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝  ╚═════╝
`

func renderRootScreen(cmd *cobra.Command) string {
	commands := make([]string, 0, len(cmd.Commands()))
	for _, sub := range cmd.Commands() {
		if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
			continue
		}
		commands = append(commands, fmt.Sprintf(
			"%s  %s",
			rootCommandStyle.Render(sub.Name()),
			rootMutedStyle.Render(sub.Short),
		))
	}

	usage := rootMutedStyle.Render("Run `demo <command> --help` for command details.")
	examples := strings.Join([]string{
		rootAccentStyle.Render("Examples"),
		"  demo onboard",
		"  demo chat --stream",
		"  demo discord status",
	}, "\n")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		rootBannerStyle.Render(strings.TrimPrefix(tangoBanner, "\n")),
		"",
		rootTagStyle.Render("TANGO CLI"),
		rootMutedStyle.Render("Terminal control surface for local setup, chat, and runtime operations."),
		"",
		rootSectionStyle.Render("Commands"),
		strings.Join(commands, "\n"),
		"",
		usage,
		"",
		examples,
	)
}
