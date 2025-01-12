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
	for i, file := range files {
		ext := path.Ext(file.Name())

		if !file.IsDir() && (slices.Contains(extList, ext)) {
			m.paths = append(m.paths, file.Name())
			m.getImage(file.Name(), i)
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
	pics  [][]byte
	loc   int
	//res  int // 0, 1, 2 for different sizes
}

func (m model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m model) View() string {
	var sb strings.Builder

	sb.Write((m.pics[m.loc]))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("img: %s | %d/%d", m.paths[m.loc], m.loc+1, len(m.pics)))

	return sb.String()
}

func (m *model) getImage(filename string, loc int) {
	fmt.Println("doing stuff")
	folder := strings.TrimSuffix(os.Args[1], "/")
	fmt.Println("opening image", loc, filename, folder)
	file, err := os.Open(fmt.Sprintf("%s/%s", folder, filename))
	if err != nil {
		m.pics = append(m.pics, []byte("opening file failed"))
		return
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

	fmt.Println("decoding image")
	var buf bytes.Buffer
	img, _, _ := image.Decode(file)
	img = resize.Resize(0, pxh, img, resize.Lanczos3)
	fmt.Println("encoding image")
	encoder := sixel.NewEncoder(&buf)
	err = encoder.Encode(img)
	if err != nil {
		m.pics = append(m.pics, []byte("encoding failed"))
		return
	}

	fmt.Println("setting image")
	m.pics = append(m.pics, buf.Bytes())
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
