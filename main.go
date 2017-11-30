package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

var info Info

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("    ", os.Args[0], "URL")
		os.Exit(1)
	}
	data := os.Args[1]
	if strings.HasPrefix(data, "web+sdm:") {
		data = data[8:]
	}
	if !strings.ContainsAny(data, "/") {
		if d, err := url.PathUnescape(data); err == nil {
			data = d
		}
	}

	if err := json.Unmarshal([]byte(data), &info); err != nil {
		info = Info{
			URL:     data,
			Cookies: nil,
			Agent:   "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.103 Safari/537.36",
		}
	}

	c := make(chan struct{})
	go start(c)
	<-c
}
