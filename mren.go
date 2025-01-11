package main

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	_ "golang.org/x/image/webp"
)

//grab all .png/.jpg/.jpeg/.webp images (gif maybe if i can figure it out)
//open first image and convert to sixel
//display old name and ask for new name
//enter with something entered -> rename
//enter with nothing entered -> keep old name
//ctrl+enter -> delete image
//shift+enter -> copy image
//grab next image and repeat

//default encoding at 640x480 for view (or smaller), button to cycle between 3 different sizes

/* possibly make it so you can move files around,
i.e. if i enter ../newfolder/myname instead of myname it will create the folder in that location
if it doesnt exist and put it there */

// have a rolling log in the corner showing the history of renames/deletions/movings etc

func main() {
	m := model{}
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	} else {
		fmt.Println("Have a nice day!")
		os.Exit(0)
	}

}

type model struct {
	pics []pic
	loc  int
	res  int // 0, 1, 2 for different sizes
}

type pic struct {
	path string
	data []byte //sixel
}

func (m model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m model) View() string {
	return "not implemented!"
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	oldLoc := m.loc
	var (
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d", "q":
			return m, tea.Quit
		}

	}

	if oldLoc != m.loc {
		cmds = append(cmds, tea.ClearScreen)
	}

	return m, tea.Batch(cmds...)
}
