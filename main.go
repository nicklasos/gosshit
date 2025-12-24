package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicklasos/gosshit/internal/sshconfig"
	"github.com/nicklasos/gosshit/internal/ui"
)

func main() {
	configPath := sshconfig.GetSSHConfigPath()

	model, err := ui.InitialModel(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing application: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
