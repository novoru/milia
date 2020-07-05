package main

import (
	"syscall"

	"golang.org/x/sys/unix"
)

func ctrlKey(k byte) byte {
	return k & 0x1F
}

// terminal
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

func disableRawMode(origTermios *unix.Termios) {
	if err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, origTermios); err != nil {
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

// output
func editorRefreshScreen() {
	syscall.Write(unix.Stdout, []byte("\x1b[2J"))
	syscall.Write(unix.Stdout, []byte("\x1b[H"))
}

// init
func main() {
	origTermios, err := unix.IoctlGetTermios(unix.Stdin, unix.TCGETS)
	if err != nil {
		panic(err)
	}

	enableRawMode()
	defer disableRawMode(origTermios)

	for persist := true; persist; {
		editorRefreshScreen()
		persist = editorProcessKeypress()
	}
}
