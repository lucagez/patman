package main

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// func main() {
// 	// t := lipgloss.NewStyle().
// 	// 	BorderStyle(lipgloss.ThickBorder()).
// 	// 	BorderBottom(true).
// 	// 	BorderForeground(lipgloss.AdaptiveColor{Light: "#EB4B4B", Dark: "#EB4B4B"}).
// 	// 	Render("|> spit(a/2)")

// 	// out := lipgloss.JoinVertical(lipgloss.Left,
// 	// 	"replace(ok/whatever)",
// 	// 	t,
// 	// 	"|> match(lol)",
// 	// )
// 	// fmt.Println(out)

// 	// Horizontal
// 	t := lipgloss.NewStyle().
// 		BorderStyle(lipgloss.ThickBorder()).
// 		BorderBottom(true).
// 		BorderForeground(lipgloss.AdaptiveColor{Light: "#EB4B4B", Dark: "#EB4B4B"}).
// 		Render("spit(a/2)")

// 	out := lipgloss.JoinHorizontal(lipgloss.Left,
// 		"replace(ok/whatever)", "|> ",
// 		t, " ",
// 		"|> match(lol)",
// 	)
// 	fmt.Println(out)
// 	fmt.Println("")
// 	fmt.Println("")
// 	fmt.Println("")

// 	title := lipgloss.NewStyle().
// 		MarginLeft(1).
// 		MarginRight(5).
// 		Padding(0, 1).
// 		Italic(true).
// 		Foreground(lipgloss.Color("#FFF7DB")).
// 		Render("Syntax Error")

// 	withTitle := lipgloss.JoinVertical(lipgloss.Left,
// 		title, out)

// 	dialog := lipgloss.NewStyle().
// 		Border(lipgloss.RoundedBorder()).
// 		BorderForeground(lipgloss.Color("#EB4B4B")).
// 		Padding(0, 2).
// 		BorderTop(true).
// 		BorderLeft(true).
// 		BorderRight(true).
// 		BorderBottom(true)
// 	physicalWidth, _, _ := term.GetSize(int(os.Stdout.Fd()))
// 	d := lipgloss.Place(physicalWidth, 9,
// 		lipgloss.Center, lipgloss.Center,
// 		dialog.Render(withTitle),
// 		// lipgloss.WithWhitespaceChars("猫咪"),
// 		// lipgloss.WithWhitespaceForeground(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}),
// 	)

// 	fmt.Println(d)
// }
func main() {
	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type errMsg error

type model struct {
	textarea textarea.Model
	err      error
}

func initialModel() model {
	ti := textarea.New()
	ti.SetPromptFunc(4, func(lineIdx int) string {
		if lineIdx == 3 {
			// Can display syntax errors on line
			return "| ❌"
		}
		return fmt.Sprintf("| %d ", lineIdx)
	})
	ti.Placeholder = "Once upon a time..."
	ti.Focus()

	return model{
		textarea: ti,
		err:      nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
		case tea.KeyCtrlC:
			return m, tea.Quit
		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return fmt.Sprintf(
		"Tell me a story.\n\n%s\n\n%s",
		m.textarea.View(),
		"(ctrl+c to quit)",
	) + "\n\n"
}
