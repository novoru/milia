package main

import (
	"io"
	"os"
)

func main() {
	for {
		buf := make([]byte, 1)
		_, err := os.Stdin.Read(buf)
		if err == io.EOF {
			break
		}
	}
}
