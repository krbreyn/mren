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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-sixel"
	"github.com/nfnt/resize"
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

	extList := []string{".jpg", ".jpeg", ".png", ".webp"}
	for _, file := range files {
		ext := path.Ext(file.Name())

		if !file.IsDir() && (slices.Contains(extList, ext)) {
			m.paths = append(m.paths, file.Name())
		}
	}

	if len(m.paths) == 0 {
		fmt.Println("no images found")
		os.Exit(0)
	}

	m.outChan = make(chan []byte, len(m.paths))

	m.currImage = getImage(m.paths[0])

	// TODO should be a status notif to show if images are still being converted and how many
	go backgroundDownloader(m.paths[1:], m.outChan)

	p := tea.NewProgram(m)

	if m, err := p.Run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	} else {
		s, ok := m.(model)
		if ok {
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
	paths     []string
	currImage []byte
	loc       int
	exitMsg   string
	outChan   chan []byte
	//res  int // 0, 1, 2 for different sizes
}

func (m model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m model) View() string {
	var sb strings.Builder

	sb.Write((m.currImage))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("img: %s\n%d/%d", m.paths[m.loc], m.loc+1, len(m.paths)))
	sb.WriteString("\nEnter New Name: ")
	sb.WriteString("\n[not yet implemented]")
	sb.WriteString("\n[keybindings to be shown]")

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

		// TODO ask if you're all done before quitting
		case "enter":
			if m.loc < len(m.paths)-1 {
				m.loc++
			} else {
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
		cmds = append(cmds, tea.ClearScreen)
	}

	return m, tea.Batch(cmds...)
}

// util

func getImage(filename string) []byte {

	folder := strings.TrimSuffix(os.Args[1], "/")
	file, err := os.Open(fmt.Sprintf("%s/%s", folder, filename))
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
	pxh := uint(temp * 9)

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
