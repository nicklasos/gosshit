package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicklasos/gosshit/internal/sshconfig"
	"github.com/nicklasos/gosshit/internal/ui"
)

const version = "1.1.1"

func main() {
	// Define flags
	showVersion := flag.Bool("version", false, "Show version information")
	showCredits := flag.Bool("credits", false, "Show credits")
	flag.Parse()

	// Handle --version flag
	if *showVersion {
		fmt.Printf("gosshit version %s\n", version)
		os.Exit(0)
	}

	// Handle --credits flag
	if *showCredits {
		fmt.Println("Named by: Stas Muzhyk")
		fmt.Println("Everything else: Mykyta Olkhovyk")
		os.Exit(0)
	}

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
