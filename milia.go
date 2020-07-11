package main

import (
	"bufio"
	"fmt"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// MilliaVersion software version for print
const MilliaVersion string = "0.0.1"

// MilliaTabStop length of a tab stop
const MilliaTabStop int = 8

func ctrlKey(k byte) int {
	return int(k & 0x1F)
}

const (
	// BackSpace  representation of backspace key
	BackSpace = iota + 127
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

type erow struct {
	s      string
	render string
}

type editorConfig struct {
	cx, cy        int
	rx            int
	rowOff        int
	colOff        int
	screenRows    uint16
	screeenCols   uint16
	rows          []erow
	fileName      string
	statusMsg     string
	statusMsgTime time.Time
	origTermios   *unix.Termios
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
	var row erow
	if e.cy < len(e.rows) {
		row = e.rows[e.cy]
	}

	switch key {
	case ArrowLeft:
		if e.cx != 0 {
			e.cx--
		} else if e.cy > 0 {
			e.cy--
			e.cx = len(e.rows[e.cy].s)
		}
	case ArrowRight:
		if row.s != "" && e.cx < len(row.s) {
			e.cx++
		} else if row.s != "" && e.cx == len(row.s) {
			e.cy++
			e.cx = 0
		}
	case ArrowUp:
		if e.cy != 0 {
			e.cy--
			if len(e.rows[e.cy].render) < e.cx {
				e.cx = len(e.rows[e.cy].render)
			}
		}
	case ArrowDown:
		if e.cy != len(e.rows) {
			e.cy++
			if e.cy < len(e.rows) {
				if len(e.rows[e.cy].render) < e.cx {
					e.cx = len(e.rows[e.cy].render)
				}
			}
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
	case '\r':
		// TODO
		break
	case ctrlKey('q'):
		return false
	case ctrlKey('s'):
		editorSave()
	case HomeKey:
		e.cx = 0
	case EndKey:
		if e.cy < len(e.rows) {
			e.cx = len(e.rows[e.cy].s)
		}
	case BackSpace, ctrlKey('h'), DelKey:
		// TODO
		break
	case PageUp, PageDown:
		if c == PageUp {
			e.cy = e.rowOff
		} else if c == PageDown {
			e.cy = e.rowOff + int(e.screenRows) - 1
			if e.cy > len(e.rows) {
				e.cy = len(e.rows)
			}
		}
		for times := int(e.screenRows); times != 0; times-- {
			if c == PageUp {
				editorMoveCursor(ArrowUp)
			} else {
				editorMoveCursor(ArrowDown)
			}
		}
	case ArrowUp, ArrowDown, ArrowLeft, ArrowRight:
		editorMoveCursor(c)
	case ctrlKey('l'), '\x1b':
		break
	default:
		editorInsertChar(c)
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

func editorRowCxToRx(row *erow, cx int) int {
	rx := 0
	for i := 0; i < cx && i < len(row.s); i++ {
		if row.s[i] == '\t' {
			rx += (MilliaTabStop - 1) - (rx % MilliaTabStop)
		}
		rx++
	}

	return rx
}

func editorUpdateRow(row *erow) {
	row.render = ""
	i := 0
	for j := 0; j < len(row.s); j++ {
		if row.s[j] == '\t' {
			row.render += " "
			i++
			for ; i%MilliaTabStop != 0; i++ {
				row.render += " "
			}
		} else {
			row.render += string(row.s[j])
			i++
		}
	}

}

func editorAppendRow(s string) {
	e.rows = append(e.rows, erow{s, ""})
	editorUpdateRow(&e.rows[len(e.rows)-1])
}

func editorRowInsertChar(row *erow, at int, c int) {
	if at < 0 || at > len(row.s) {
		at = len(row.s)
	}
	row.s = row.s[:at] + string(c) + row.s[at:]
	editorUpdateRow(row)
}

// editor operations

func editorInsertChar(c int) {
	if e.cy == len(e.rows) {
		editorAppendRow("")
	}
	editorRowInsertChar(&e.rows[e.cy], e.cx, c)
	e.cx++
}

// file I/O

func editorRowsToString() string {
	buf := ""

	for i := 0; i < len(e.rows); i++ {
		buf += e.rows[i].s + "\n"
	}

	return buf
}

func editorOpen(fileName string) {
	e.fileName = fileName
	file, err := os.OpenFile(e.fileName, os.O_RDWR|os.O_CREATE, 0644)
	defer file.Close()

	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		editorAppendRow(scanner.Text())
	}
}

func editorSave() {
	if e.fileName == "" {
		return
	}

	file, err := os.OpenFile(e.fileName, os.O_RDWR|os.O_CREATE, 0644)
	defer file.Close()
	if err != nil {
		panic(err)
	}

	buf := editorRowsToString()

	n, err := file.Write([]byte(buf))
	if err != nil {
		panic(err)
	}
	e.statusMsgTime = time.Now()
	editorSetStatusMessage("%d bytes written to disk", n)
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
	e.rx = 0
	if e.cy < len(e.rows) {
		e.rx = editorRowCxToRx(&e.rows[e.cy], e.cx)
	}

	if e.cy < e.rowOff {
		e.rowOff = e.cy
	}
	if e.cy >= e.rowOff+int(e.screenRows) {
		e.rowOff = e.cy - int(e.screenRows) + 1
	}
	if e.rx < e.colOff {
		e.colOff = e.rx
	}
	if e.rx >= e.colOff+int(e.screeenCols) {
		e.colOff = e.rx - int(e.screeenCols) + 1
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
			len := len(e.rows[fileRow].render) - e.colOff
			if len > 0 {
				if len > int(e.screeenCols) {
					len = int(e.screeenCols)
				}
				abAppend(ab, e.rows[fileRow].render[e.colOff:e.colOff+len])
			}
		}

		abAppend(ab, "\x1b[K")
		abAppend(ab, "\r\n")
	}
}

func editorDrawStatusBar(ab *abuf) {
	abAppend(ab, "\x1b[7m")

	fileName := "[No Name]"
	if e.fileName != "" {
		fileName = e.fileName
	}

	status := fmt.Sprintf("%.20s - %d lines", fileName, len(e.rows))
	rstatus := fmt.Sprintf("%d/%d", e.cy+1, len(e.rows))
	l := len(status)
	if l > int(e.screeenCols) {
		l = int(e.screeenCols)
	}
	abAppend(ab, status)

	for ; l < int(e.screeenCols); l++ {
		if int(e.screeenCols)-l == len(rstatus) {
			abAppend(ab, rstatus)
			break
		} else {
			abAppend(ab, " ")
		}
	}
	abAppend(ab, "\x1b[m")
	abAppend(ab, "\r\n")
}

func editorDrawMessageBar(ab *abuf) {
	abAppend(ab, "\x1b[K")
	msgLen := len(e.statusMsg)
	if msgLen > int(e.screeenCols) {
		msgLen = int(e.screeenCols)
	}

	if msgLen != 0 && time.Now().Sub(e.statusMsgTime).Seconds() < 5 {
		abAppend(ab, e.statusMsg)
	}
}

func editorRefreshScreen() {
	editorScroll()

	ab := new(abuf)

	abAppend(ab, "\x1b[?25l")
	abAppend(ab, "\x1b[H")

	editorDrawRows(ab)
	editorDrawStatusBar(ab)
	editorDrawMessageBar(ab)

	abAppend(ab, fmt.Sprintf("\x1b[%d;%dH",
		(e.cy-e.rowOff)+1, (e.rx-e.colOff)+1))

	abAppend(ab, "\x1b[?25h")

	syscall.Write(unix.Stdout, []byte(ab.b))
}

func editorSetStatusMessage(f string, a ...interface{}) {
	e.statusMsg = fmt.Sprintf(f, a...)
}

// init

func initEditor() {
	e.cx = 0
	e.cy = 0
	e.rx = 0
	e.rowOff = 0
	e.colOff = 0
	e.rows = make([]erow, 0)
	e.fileName = ""
	e.statusMsg = ""
	e.statusMsgTime = time.Now()

	getWindowSize(&e.screenRows, &e.screeenCols)
	e.screenRows -= 2
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

	editorSetStatusMessage("HELP: Ctrl-Q = quit")

	for persist := true; persist; {
		editorRefreshScreen()
		persist = editorProcessKeypress()
	}
}
