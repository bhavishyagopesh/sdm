package main

import (
	"container/list"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const defaultAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.103 Safari/537.36"

type cookie struct {
	Name           string `json:"name"`
	Value          string `json:"value"`
	Path           string `json:"path,omitempty"`
	Domain         string `json:"domain,omitempty"`
	ExpirationDate int    `json:"expirationDate,omitempty"`
	Secure         bool   `json:"secure,omitempty"`
	HttpOnly       bool   `json:"httpOnly,omitempty"`
	HostOnly       bool   `json:"hostOnly,omitempty"`
	Session        bool   `json:"session,omitempty"`
	StoreID        string `json:"storeId,omitempty"`
}

type Download struct {
	State

	URL      string   `json:"url"`
	Cookies  []cookie `json:"cookies"`
	Agent    string   `json:"agent"`
	MaxThrds int      `json:"maxThrds"`
}

type State struct {
	FileSize  int64
	StartTime time.Time
	FileName  string
	Counter   int64

	Chunks    list.List
	Resumable ResumableT
	wg        sync.WaitGroup
}

type ResumableT int

const (
	DEF ResumableT = iota
	YES ResumableT = iota
	NO  ResumableT = iota
)

func isChar(ch byte) bool {
	if ch >= 'a' && ch <= 'z' {
		return true
	} else if ch >= 'A' && ch <= 'Z' {
		return true
	} else if ch >= '0' && ch <= '9' {
		return true
	}
	return false
}

func convertCookie(c cookie) *http.Cookie {
	return &http.Cookie{
		Name:     c.Name,
		Value:    c.Value,
		Path:     c.Path,
		Secure:   c.Secure,
		HttpOnly: c.HttpOnly,
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func removePrefix(a, b string) string {
	if strings.HasPrefix(a, b) {
		return a[len(b):]
	}
	return a
}

func process(a string) (string, error) {
	a = removePrefix(a, "web+sdm:")
	a, err := url.PathUnescape(a)
	if err != nil {
		return "", fmt.Errorf("unescape failed: %v", err)
	}
	return a, nil
}

func HumanReadable(num int64) string {
	return humanReadable(float64(num))
}

func humanReadable(num float64) string {
	str := ""
	sizes := []string{"B", "KB", "MB", "GB"}

	for _, unit := range sizes {
		str = fmt.Sprintf("%.2f%s", num, unit)
		if num < 1000 {
			break
		}
		num /= 1000
	}
	return str
}

func termClear() {
	fmt.Printf("\033[2J")
}

func termPos(i, j int) {
	fmt.Printf("\033[0;0f") // Go to (0, 0) first.
	if i > 0 {
		fmt.Printf("\033[%dB", i)
	} else if i < 0 {
		fmt.Printf("\033[2000B")   // Go to bottom.
		fmt.Printf("\033[%dA", -i) // Go back i positions.
	}

	if j > 0 {
		fmt.Printf("\033[%dC", j)
	} else if j < 0 {
		fmt.Printf("\033[2000C")   // Go far to the right end.
		fmt.Printf("\033[%dD", -j) // Go back left j positions.
	}
}
