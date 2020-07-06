package main

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// MilliaVersion software version for print
const MilliaVersion string = "0.0.1"

func ctrlKey(k byte) byte {
	return k & 0x1F
}

// data
type editorConfig struct {
	screenRows  uint16
	screeenCols uint16
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
func editorReadKey() byte {
	buf := make([]byte, 1)

	for size, err := syscall.Read(unix.Stdin, buf); size != 1; {
		if err != nil {
			panic(err)
		}
	}

	return buf[0]
}

func editorProcessKeypress() bool {
	switch c := editorReadKey(); c {
	case ctrlKey('q'):
		return false
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

// append buffer
type abuf struct {
	b string
}

func abAppend(ab *abuf, s string) {
	ab.b += s
}

// output
func editorDrawRows(ab *abuf) {
	for y := 0; y < int(e.screenRows); y++ {
		if y == int(e.screenRows)/3 {
			welcome := "Millia editor " + MilliaVersion
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

		abAppend(ab, "\x1b[K")
		if y < int(e.screenRows)-1 {
			abAppend(ab, "\r\n")
		}
	}
}

func editorRefreshScreen() {
	ab := new(abuf)

	abAppend(ab, "\x1b[?25l")
	abAppend(ab, "\x1b[H")

	editorDrawRows(ab)

	abAppend(ab, "\x1b[H")
	abAppend(ab, "\x1b[?25h")

	syscall.Write(unix.Stdout, []byte(ab.b))
}

// init

func initEditor() {
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

	for persist := true; persist; {
		editorRefreshScreen()
		persist = editorProcessKeypress()
	}
}
