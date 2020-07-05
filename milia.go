package main

import (
	"fmt"
	"os"
	"unicode"

	"golang.org/x/sys/unix"
)

func ctrlKey(k byte) byte {
	return k & 0x1F
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

func disableRawMode(origTermios *unix.Termios) {
	if err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, origTermios); err != nil {
		panic(err)
	}
}

func main() {
	origTermios, err := unix.IoctlGetTermios(unix.Stdin, unix.TCGETS)
	if err != nil {
		panic(err)
	}

	enableRawMode()
	defer disableRawMode(origTermios)

	for {
		buf := make([]byte, 1)
		_, err := os.Stdin.Read(buf)
		if err != nil {
			panic(err)
		}

		if unicode.IsControl(rune(buf[0])) {
			fmt.Printf("%d\r\n", buf[0])
		} else {
			fmt.Printf("%d ('%c')\r\n", buf[0], buf[0])
		}
		if buf[0] == ctrlKey('q') {
			break
		}
	}
}
