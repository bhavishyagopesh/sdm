package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("    ", os.Args[0], "URL")
		os.Exit(1)
	}

	data := os.Args[1]
	data, err := process(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	dl := &Download{}
	if err := json.Unmarshal([]byte(data), dl); err != nil {
		dl = &Download{
			URL:     data,
			Cookies: nil,
			Agent:   defaultAgent,
		}
	}
	dl.MaxThrds = 8

	done := make(chan error)
	go dl.Start(done)
	if err := <-done; err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := dl.SaveFinalFile(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
