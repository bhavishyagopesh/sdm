package main

import "net/http"

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

type Info struct {
	URL     string   `json:"url"`
	Cookies []cookie `json:"cookies"`
	Agent   string   `json:"agent"`
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
