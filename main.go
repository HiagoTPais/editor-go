package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
)

var offsetY int
var searchQuery string
var searchResults [][2]int
var currentSearchIndex int

// Pilhas de histórico para Undo/Redo
var undoStack [][][]rune
var redoStack [][][]rune

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run main.go <nome_do_arquivo>")
		return
	}
	filename := os.Args[1]

	screen, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := screen.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	defer screen.Fini()

	text, err := loadFromFile(filename)
	if err != nil {
		text = [][]rune{make([]rune, 0)}
	}

	cursorX := 0
	cursorY := 0

	for {
		screen.Clear()
		width, height := screen.Size()

		if cursorY < offsetY {
			offsetY = cursorY
		}
		if cursorY >= offsetY+height-1 {
			offsetY = cursorY - height + 2
		}

		// Exibe o texto na tela com scroll
		for y := offsetY; y < offsetY+height-1 && y < len(text); y++ {
			for x, r := range text[y] {
				if x < width {
					screen.SetContent(x, y-offsetY, r, nil, tcell.StyleDefault)
				}
			}
		}

		// Barra de status
		status := fmt.Sprintf("[Arquivo: %s] Linhas:%d Pos:%d,%d Ctrl+S:Salvar Ctrl+Z:Undo Ctrl+Y:Redo ESC:Sair",
			filename, len(text), cursorY+1, cursorX+1)
		for x, r := range status {
			if x < width {
				screen.SetContent(x, height-1, r, nil, tcell.StyleDefault.Foreground(tcell.ColorGreen))
			}
		}

		screen.ShowCursor(cursorX, cursorY-offsetY)
		screen.Show()

		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				return
			case tcell.KeyEnter:
				saveUndo(text)
				currentLine := text[cursorY]
				newLine := append([]rune{}, currentLine[cursorX:]...)
				text[cursorY] = currentLine[:cursorX]
				cursorY++
				text = append(text[:cursorY], append([][]rune{newLine}, text[cursorY:]...)...)
				cursorX = 0
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if cursorX > 0 {
					saveUndo(text)
					line := text[cursorY]
					text[cursorY] = append(line[:cursorX-1], line[cursorX:]...)
					cursorX--
				} else if cursorY > 0 {
					saveUndo(text)
					prevLineLen := len(text[cursorY-1])
					text[cursorY-1] = append(text[cursorY-1], text[cursorY]...)
					text = append(text[:cursorY], text[cursorY+1:]...)
					cursorY--
					cursorX = prevLineLen
				}
			case tcell.KeyLeft:
				if cursorX > 0 {
					cursorX--
				} else if cursorY > 0 {
					cursorY--
					cursorX = len(text[cursorY])
				}
			case tcell.KeyRight:
				if cursorX < len(text[cursorY]) {
					cursorX++
				} else if cursorY+1 < len(text) {
					cursorY++
					cursorX = 0
				}
			case tcell.KeyUp:
				if cursorY > 0 {
					cursorY--
					if cursorX > len(text[cursorY]) {
						cursorX = len(text[cursorY])
					}
				}
			case tcell.KeyDown:
				if cursorY+1 < len(text) {
					cursorY++
					if cursorX > len(text[cursorY]) {
						cursorX = len(text[cursorY])
					}
				}
			case tcell.KeyCtrlS:
				saveToFile(text, filename)
			case tcell.KeyCtrlO:
				loadedText, err := loadFromFile(filename)
				if err == nil {
					saveUndo(text)
					text = loadedText
					cursorX = 0
					cursorY = 0
					offsetY = 0
				} else {
					log.Printf("Erro ao abrir o arquivo: %v", err)
				}
			case tcell.KeyCtrlZ:
				// Undo
				if len(undoStack) > 0 {
					redoStack = append(redoStack, cloneText(text))
					text = undoStack[len(undoStack)-1]
					undoStack = undoStack[:len(undoStack)-1]
					cursorX, cursorY = 0, 0
					offsetY = 0
				}
			case tcell.KeyCtrlY:
				// Redo
				if len(redoStack) > 0 {
					undoStack = append(undoStack, cloneText(text))
					text = redoStack[len(redoStack)-1]
					redoStack = redoStack[:len(redoStack)-1]
					cursorX, cursorY = 0, 0
					offsetY = 0
				}
			case tcell.KeyCtrlF:
				query := []rune{}
				if query != nil && len(query) > 0 {
					searchQuery = string(query)
					searchResults = searchInText(text, searchQuery)
					currentSearchIndex = 0
					if len(searchResults) > 0 {
						cursorX = searchResults[0][0]
						cursorY = searchResults[0][1]
						offsetY = 0
					}
				}
			default:
				if ev.Rune() != 0 {
					saveUndo(text)
					r := ev.Rune()
					line := text[cursorY]
					text[cursorY] = append(line[:cursorX], append([]rune{r}, line[cursorX:]...)...)
					cursorX++
				}
			}
		case *tcell.EventResize:
			screen.Sync()
		}
	}
}

// Salva o estado atual do texto na pilha de Undo
func saveUndo(text [][]rune) {
	snapshot := cloneText(text)
	undoStack = append(undoStack, snapshot)
	redoStack = nil // Limpa o redo ao fazer uma nova alteração
}

// Faz uma cópia profunda do texto
func cloneText(text [][]rune) [][]rune {
	copyText := make([][]rune, len(text))
	for i, line := range text {
		copyText[i] = append([]rune{}, line...)
	}
	return copyText
}

// Salva o texto no arquivo
func saveToFile(text [][]rune, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Printf("Erro ao salvar o arquivo: %v", err)
		return
	}
	defer file.Close()

	for _, line := range text {
		_, err := file.WriteString(string(line) + "\n")
		if err != nil {
			log.Printf("Erro ao escrever no arquivo: %v", err)
			return
		}
	}
	log.Printf("Arquivo salvo como %s", filename)
}

// Carrega o texto de um arquivo
func loadFromFile(filename string) ([][]rune, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var loadedText [][]rune
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		loadedText = append(loadedText, []rune(scanner.Text()))
	}
	if len(loadedText) == 0 {
		loadedText = [][]rune{make([]rune, 0)}
	}
	return loadedText, scanner.Err()
}

func searchInText(text [][]rune, query string) [][2]int {
	var results [][2]int
	for y, line := range text {
		lineStr := string(line)
		x := 0
		for {
			idx := strings.Index(lineStr[x:], query)
			if idx == -1 {
				break
			}
			results = append(results, [2]int{x + idx, y})
			x += idx + 1
		}
	}
	return results
}

func IndexOf(s, substr string) int {
	return len([]rune(s[:len(s)])) - len([]rune(s[len([]rune(s)):])) + len([]rune(substr)) - len([]rune(substr)) + len([]rune(substr))
}
