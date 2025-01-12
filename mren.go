package main

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

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
	if len(os.Args) > 2 || len(os.Args) < 2 {
		fmt.Println("Usage: mren <directory>")
		os.Exit(0)
	}

	files, err := os.ReadDir(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	m := model{}

	for _, file := range files {
		if !file.IsDir() {
			m.paths = append(m.paths, file.Name())
		}
	}

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
	paths []string
	//pics []byte
	loc int
	//res  int // 0, 1, 2 for different sizes
}

func (m model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m model) View() string {
	var sb strings.Builder

	for _, f := range m.paths {
		sb.WriteString(f + "\n")
	}
	return sb.String()
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
