package main

import (
	"bufio"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// MilliaVersion software version for print
const MilliaVersion string = "0.0.1"

func ctrlKey(k byte) int {
	return int(k & 0x1F)
}

const (
	// ArrowLeft  representation of arrow left key
	ArrowLeft = iota + 1000
	// ArrowRight  representation of arrow right key
	ArrowRight
	// ArrowUp  representation of arrow up key
	ArrowUp
	// ArrowDown  representation of arrow down key
	ArrowDown
	// DelKey  representation of delete key
	DelKey
	// HomeKey  representation of home key
	HomeKey
	// EndKey  representation of end key
	EndKey
	// PageUp  representation of pageup key
	PageUp
	// PageDown  representation of pagedown key
	PageDown
)

// data

type editorConfig struct {
	cx, cy      int
	rowOff      int
	colOff      int
	screenRows  uint16
	screeenCols uint16
	rows        []string
	origTermios *unix.Termios
}

var e editorConfig

// terminal
func die() {
	syscall.Write(unix.Stdout, []byte("\x1b[2J"))
	syscall.Write(unix.Stdout, []byte("\x1b[H"))
	disableRawMode()
}

func enableRawMode() {
	termios, err := unix.IoctlGetTermios(unix.Stdin, unix.TCGETS)
	if err != nil {
		panic(err)
	}

	mask := ^(unix.BRKINT | unix.ICRNL | unix.INPCK | unix.ISTRIP | unix.IXON)
	termios.Iflag &= uint32(mask)
	mask = ^(unix.OPOST)
	termios.Oflag &= uint32(mask)
	termios.Cflag |= uint32(unix.CS8)
	mask = ^(unix.ECHO | unix.ICANON | unix.IEXTEN | unix.ISIG)
	termios.Lflag &= uint32(mask)

	termios.Cc[unix.VMIN] = 0
	termios.Cc[unix.VTIME] = 1

	if err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, termios); err != nil {
		panic(err)
	}
}

func disableRawMode() {
	if err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, e.origTermios); err != nil {
		panic(err)
	}
}

// input
func editorMoveCursor(key int) {
	switch key {
	case ArrowLeft:
		if e.cx != 0 {
			e.cx--
		}
	case ArrowRight:
		e.cx++
	case ArrowUp:
		if e.cy != 0 {
			e.cy--
		}
	case ArrowDown:
		if e.cy != len(e.rows) {
			e.cy++
		}
	}
}

func editorReadKey() int {
	buf := make([]byte, 1)

	for size, err := syscall.Read(unix.Stdin, buf); size != 1; {
		if err != nil {
			panic(err)
		}
	}

	if buf[0] == '\x1b' {
		seq := make([]byte, 3)
		_, err := syscall.Read(unix.Stdin, seq)
		if err != nil {
			panic(err)
		}

		if seq[0] == '[' {
			if seq[1] >= '0' && seq[1] <= '9' {
				if seq[2] == '~' {
					switch seq[1] {
					case '1':
						return HomeKey
					case '3':
						return DelKey
					case '4':
						return EndKey
					case '5':
						return PageUp
					case '6':
						return PageDown
					case '7':
						return HomeKey
					case '8':
						return EndKey
					}
				}
			} else {
				switch seq[1] {
				case 'A':
					return ArrowUp
				case 'B':
					return ArrowDown
				case 'C':
					return ArrowRight
				case 'D':
					return ArrowLeft
				case 'H':
					return HomeKey
				case 'F':
					return EndKey
				}
			}
		} else if seq[0] == 'O' {
			switch seq[1] {
			case 'H':
				return HomeKey
			case 'F':
				return EndKey
			}
		}

		return '\x1b'
	}

	return int(buf[0])
}

func editorProcessKeypress() bool {
	switch c := editorReadKey(); c {
	case ctrlKey('q'):
		return false
	case HomeKey:
		e.cx = 0
	case EndKey:
		e.cx = int(e.screeenCols) - 1
	case PageUp, PageDown:
		times := int(e.screenRows)
		for ; times != 0; times-- {
			if c == PageUp {
				editorMoveCursor(ArrowUp)
			} else {
				editorMoveCursor(ArrowDown)
			}
		}
	case ArrowUp, ArrowDown, ArrowLeft, ArrowRight:
		editorMoveCursor(c)
	}

	return true
}

func getWindowSize(rows *uint16, cols *uint16) int {
	ws, err := unix.IoctlGetWinsize(unix.Stdout, unix.TIOCGWINSZ)
	if err != nil {
		panic(err)
	}
	*cols = ws.Col
	*rows = ws.Row

	return 0
}

// row operations

func editorAppendRow(s string) {
	e.rows = append(e.rows, s)
}

// file I/O

func editorOpen(fileName string) {
	file, err := os.Open(fileName)
	defer file.Close()

	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		editorAppendRow(scanner.Text())
	}
}

// append buffer
type abuf struct {
	b string
}

func abAppend(ab *abuf, s string) {
	ab.b += s
}

// output

func editorScroll() {
	if e.cy < e.rowOff {
		e.rowOff = e.cy
	}
	if e.cy >= e.rowOff+int(e.screenRows) {
		e.rowOff = e.cy - int(e.screenRows) + 1
	}
	if e.cx < e.colOff {
		e.colOff = e.cx
	}
	if e.cx >= e.colOff+int(e.screeenCols) {
		e.colOff = e.cx - int(e.screeenCols) + 1
	}
}

func editorDrawRows(ab *abuf) {
	for y := 0; y < int(e.screenRows); y++ {
		fileRow := y + e.rowOff
		if fileRow >= len(e.rows) {
			if len(e.rows) == 0 && y == int(e.screenRows)/3 {
				welcome := "Millia editor -- version " + MilliaVersion
				padding := (int(e.screeenCols) - len(welcome)) / 2
				if padding != 0 {
					abAppend(ab, "~")
					padding--
				}
				for ; padding != 0; padding-- {
					abAppend(ab, " ")
				}
				abAppend(ab, welcome)
			} else {
				abAppend(ab, "~")
			}
		} else {
			len := len(e.rows[fileRow]) - e.colOff
			if len > 0 {
				if len > int(e.screeenCols) {
					len = int(e.screeenCols)
				}
				abAppend(ab, e.rows[fileRow][e.colOff:e.colOff+len])
			}
		}

		abAppend(ab, "\x1b[K")
		if y < int(e.screenRows)-1 {
			abAppend(ab, "\r\n")
		}
	}
}

func editorRefreshScreen() {
	editorScroll()

	ab := new(abuf)

	abAppend(ab, "\x1b[?25l")
	abAppend(ab, "\x1b[H")

	editorDrawRows(ab)

	abAppend(ab, fmt.Sprintf("\x1b[%d;%dH",
		(e.cy-e.rowOff)+1, (e.cx-e.colOff)+1))

	abAppend(ab, "\x1b[?25h")

	syscall.Write(unix.Stdout, []byte(ab.b))
}

// init

func initEditor() {
	e.cx = 0
	e.cy = 0
	e.rowOff = 0
	e.colOff = 0
	e.rows = make([]string, 0)
	getWindowSize(&e.screenRows, &e.screeenCols)
}

func main() {
	var err error
	e.origTermios, err = unix.IoctlGetTermios(unix.Stdin, unix.TCGETS)
	if err != nil {
		panic(err)
	}

	enableRawMode()
	initEditor()
	defer die()

	if len(os.Args) >= 2 {
		editorOpen(os.Args[1])
	}

	for persist := true; persist; {
		editorRefreshScreen()
		persist = editorProcessKeypress()
	}
}
