package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-sixel"
	"github.com/nfnt/resize"
	_ "golang.org/x/image/webp"
)

// TODO figure out how to make it look pretty with lipgloss

// TODO refactor this mess and write tests

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: mren <directory>")
		os.Exit(0)
	}

	files, err := os.ReadDir(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	m := model{}

	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 128
	ti.Width = 64

	m.textInput = ti

	m.folder = strings.TrimSuffix(os.Args[1], "/")
	extList := []string{".jpg", ".jpeg", ".png", ".webp"}
	for _, file := range files {
		ext := path.Ext(file.Name())

		if !file.IsDir() && slices.Contains(extList, ext) {
			m.paths = append(m.paths, fmt.Sprintf("%s/%s", m.folder, file.Name()))
		}
	}

	if len(m.paths) == 0 {
		fmt.Println("no images found")
		os.Exit(0)
	}

	m.outChan = make(chan []byte, len(m.paths))

	m.currImage = getImage(m.paths[0])
	m.textInput.Placeholder = trimPath(m.paths[0], m.folder)

	// TODO should be a status notif to show if images are still being converted and how many
	go backgroundDownloader(m.paths[1:], m.outChan)

	p := tea.NewProgram(m)

	if m, err := p.Run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	} else {
		s, ok := m.(model)
		if ok && s.exitMsg != "" {
			fmt.Println(s.exitMsg)
		}
		fmt.Println("Have a nice day!")
		os.Exit(0)
	}

}

func backgroundDownloader(paths []string, outChan chan<- []byte) {
	for _, p := range paths {
		outChan <- getImage(p)
	}
}

type model struct {
	paths      []string
	currImage  []byte
	loc        int
	exitMsg    string
	outChan    chan []byte
	textInput  textinput.Model
	displayMsg string
	folder     string
	//res  int // 0, 1, 2 for different sizes
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, textinput.Blink)
}

func (m model) View() string {
	var sb strings.Builder

	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("img: %s\n%d/%d",
		trimPath(m.paths[0], m.folder), m.loc+1, len(m.paths)))

	sb.WriteString("\nEnter New Name: \n")
	sb.WriteString(m.textInput.View())

	sb.WriteString("\nenter: submit | empty = skip | 'asd' = rename | '../x/asd' = move\n")
	sb.WriteString("(todo) shift+enter: delete\n")
	sb.WriteString("(todo) ctrl+enter: copy\n")

	sb.WriteString(m.displayMsg)
	sb.WriteString("\n")

	sb.Write(m.currImage)

	return sb.String()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	oldLoc := m.loc
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c", "ctrl+d":
			return m, tea.Quit

		// TODO ask if you're all done before quitting
		// TODO implement saving to different folders
		// TODO remove "test/" from placeholder and img:
		case "enter":
			if m.loc < len(m.paths)-1 {
				input := m.textInput.Value()
				if input != "" {
					new_path := fmt.Sprintf("%s/%s%s", m.folder, input, path.Ext(m.paths[m.loc]))
					err := os.Rename(m.paths[m.loc], new_path)
					if err != nil {
						panic(err)
					}
					m.displayMsg = fmt.Sprintf("renamed %s to %s", trimPath(m.paths[m.loc], m.folder), new_path)
				} else {
					m.displayMsg = fmt.Sprintf("skipped %s", trimPath(m.paths[m.loc], m.folder))
				}

				m.textInput.Reset()

				m.loc++
				m.textInput.Placeholder = m.paths[m.loc]
			} else {
				//m.toQuit++
				m.exitMsg = "All done!"
				return m, tea.Quit
			}

		}

	case tea.WindowSizeMsg:
		/* TODO change the image size to a model variable
		initialized on startup and change it through here */

	}

	if oldLoc != m.loc {
		m.currImage = <-m.outChan
	}

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// util

func trimPath(filename, folder string) string {
	return strings.TrimPrefix(filename, folder+"/")
}

func getImage(filename string) []byte {

	file, err := os.Open(filename)
	if err != nil {
		return []byte("opening file failed")
	}
	defer file.Close()

	//get size for image
	// TODO handle resizing events
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	// TODO handle err
	out, _ := cmd.Output()
	th := strings.Split(strings.TrimSpace(string(out)), " ")[0]
	temp, _ := strconv.Atoi(th)
	pxh := uint(temp * 7)

	var buf bytes.Buffer
	img, _, _ := image.Decode(file)
	img = resize.Resize(0, pxh, img, resize.Lanczos3)
	encoder := sixel.NewEncoder(&buf)
	err = encoder.Encode(img)
	if err != nil {
		return []byte("encoding failed")
	}

	return buf.Bytes()
}
