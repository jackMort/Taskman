package main

import (
	"fmt"
	"os"
	"taskman/components/calendar"
	"taskman/components/config"
	"taskman/components/footer"
	"taskman/components/results"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	boxer "github.com/treilik/bubbleboxer"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "taskman",
	Short: "A CLI tool for Task Management",
	Long: `
░▀█▀░█▀█░█▀▀░█░█░█▄█░█▀█░█▀█
░░█░░█▀█░▀▀█░█▀▄░█░█░█▀█░█░█
░░▀░░▀░▀░▀▀▀░▀░▀░▀░▀░▀░▀░▀░▀

Taskman is a CLI tool for Task API.`,
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		config.SetVersion(version)
		viper.SetConfigName("config")         // name of config file (without extension)
		viper.SetConfigType("json")           // REQUIRED if the config file does not have the extension in the name
		viper.AddConfigPath("/etc/taskman/")  // path to look for the config file in
		viper.AddConfigPath("$HOME/.taskman") // call multiple times to add many search paths
		err := viper.ReadInConfig()           // Find and read the config file
		if err != nil {                       // Handle errors reading the config file
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// NOTE: ignore if config file is not found
			} else {
				panic(fmt.Errorf("fatal error config file: %w", err))
			}
		}

		// ----
		zone.NewGlobal()

		footerBox := footer.New()
		resultsBox := results.New()
		calendarBox := calendar.New()

		// layout-tree defintion
		m := Model{tui: boxer.Boxer{}}

		rootNode := boxer.CreateNoBorderNode()
		rootNode.VerticalStacked = true
		rootNode.SizeFunc = func(node boxer.Node, widthOrHeight int) []int {
			return []int{
				widthOrHeight - 1,
				1,
			}
		}

		centerNode := boxer.CreateNoBorderNode()
		centerNode.VerticalStacked = false
		centerNode.SizeFunc = func(node boxer.Node, widthOrHeight int) []int {
			return []int{
				widthOrHeight - 30,
				30,
			}
		}

		centerNode.Children = []boxer.Node{
			stripErr(m.tui.CreateLeaf("results", resultsBox)),
			stripErr(m.tui.CreateLeaf("calendar", calendarBox)),
		}

		rootNode.Children = []boxer.Node{
			centerNode,
			stripErr(m.tui.CreateLeaf("footer", footerBox)),
		}

		m.tui.LayoutTree = rootNode

		if f, err := tea.LogToFile("debug.log", "debug"); err != nil {
			fmt.Println("Couldn't open a file for logging:", err)
			os.Exit(1)
		} else {
			defer f.Close()
		}

		p := tea.NewProgram(
			m,
			tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
			tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
		)

		if _, err := p.Run(); err != nil {
			fmt.Println("could not run program:", err)
			os.Exit(1)
		}
	},
}
