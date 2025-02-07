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

// undo commands should be a stack of functions that you pop and call to undo the action
//but do that after more refactoring

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: mren <directory>")
		os.Exit(0)
	}

	m := initialModel()

	p := tea.NewProgram(m)

	if m, err := p.Run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	} else {
		goodbye(m)
	}

}

func goodbye(m tea.Model) {
	s, ok := m.(model)
	if ok && s.exitMsg != "" {
		fmt.Println(s.exitMsg)
	}

	fmt.Println("Have a nice day!")
	os.Exit(0)
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

	sb.WriteString(fmt.Sprintf("%d/%d | %s\n",
		m.loc+1, len(m.paths), trimPath(m.paths[m.loc], m.folder)))

	sb.WriteString("Enter New Name:\n")
	sb.WriteString(m.textInput.View())

	sb.WriteString("\nenter: empty = skip | 'asd' = rename | '../x/asd' = move & rename\n")
	sb.WriteString("alt+enter: empty = delete | with '../x' = move without rename\n")
	sb.WriteString("(todo) undo button\n")

	sb.WriteString(m.displayMsg + "\n")

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
		key := msg.String()
		switch key {
		case "ctrl+c", "ctrl+d":
			return m, tea.Quit

			// TODO ask if you're all done before quitting (after a going back mechanism)
		case "enter", "alt+enter":
			if m.loc < len(m.paths)-1 {
				input := m.textInput.Value()

				action := handleInput(key, input, m.paths[m.loc], m.folder)
				m.displayMsg = action()

				m.loc++
				m.textInput.Reset()
				m.textInput.Placeholder = trimPath(m.paths[m.loc], m.folder)
			} else {
				//m.toQuit++, displayMsg = "once more to quit"
				m.exitMsg = "All done!"
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		/* TODO change the image size to a model variable
		 *		initialized on startup and change it through here */

	}

	if oldLoc != m.loc {
		m.currImage = <-m.outChan
		cmds = append(cmds, tea.ClearScreen)
	}

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

var extList []string = []string{".jpg", ".jpeg", ".png", ".webp"}

func initialModel() model {
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

	m.outChan = make(chan []byte, 20)

	m.currImage = getImage(m.paths[0])
	//TODO trim ext
	m.textInput.Placeholder = trimPath(m.paths[0], m.folder)

	// TODO should be a status notif to show if images are still being converted and how many
	go backgroundDownloader(m.paths[1:], m.outChan)

	return m
}

// util

func handleInput(key, input, pathname, folder string) func() string {
	if input != "" {
		if err := ensureDirsExist(input, folder, key); err != nil {
			panic(err)
		}

		new_path, display_msg := getNewPath(input, key, folder, pathname)

		action := func() string {
			if err := os.Rename(pathname, new_path); err != nil {
				return fmt.Sprintf("error %v", err)
			} else {
				return display_msg
			}
		}

		return action
	} else {
		if key != "alt+enter" {
			action := func() string {
				return fmt.Sprintf("skipped %s", trimPath(pathname, folder))
			}

			return action
		}

		action := func() string {
			if err := os.Remove(pathname); err != nil {
				return fmt.Sprintf("failed to delete %s", trimPath(pathname, folder))
			} else {
				return fmt.Sprintf("deleted %s", trimPath(pathname, folder))
			}
		}

		return action
	}
}

func getNewPath(input, key, folder, pathname string) (string, string) {
	var new_path string
	var display_msg string

	switch key {
	case "enter":
		new_path = fmt.Sprintf(
			"%s/%s%s", folder, input, path.Ext(pathname))
		display_msg = fmt.Sprintf(
			"renamed %s to %s", trimPath(pathname, folder), new_path)

	case "alt+enter":
		if input[len(input)-1] != byte('/') {
			input += "/"
		}
		new_path = fmt.Sprintf(
			"%s/%s%s", folder, input, trimPath(pathname, folder))
		display_msg = fmt.Sprintf(
			"moved %s to %s", trimPath(pathname, folder), trimPath(new_path, folder))
	}
	return new_path, display_msg
}

func ensureDirsExist(input, folder, key string) error {
	fields := strings.Split(input, "/")
	path := folder + "/"
	var target int
	switch key {
	case "alt+enter":
		target = len(fields)

	case "enter":
		target = len(fields) - 1
	}
	for i := 0; i < target; i++ {
		path += fields[i] + "/"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.Mkdir(path, os.ModePerm); err != nil {
				return err
			}
		}

	}

	return nil
}

func trimPath(filename, folder string) string {
	return strings.TrimPrefix(filename, folder+"/")
}

func backgroundDownloader(paths []string, outChan chan<- []byte) {
	for _, p := range paths {
		outChan <- getImage(p)
	}
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
	img, _, err := image.Decode(file)
	if err != nil {
		return []byte("error decoding image")
	}
	img = resize.Resize(0, pxh, img, resize.Lanczos3)
	encoder := sixel.NewEncoder(&buf)
	err = encoder.Encode(img)
	if err != nil {
		return []byte("encoding failed")
	}

	return buf.Bytes()
}
