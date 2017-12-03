package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var sizeGlobal int64
var startTime time.Time = time.Now()
var totalDown int64

func printSpeed() {
	for {
		time.Sleep(1 * time.Second)
		t := time.Now().Sub(startTime).Seconds()
		speed := float64(totalDown) / t
		if speed < 1000 {
			fmt.Printf("Speed: %.2fB/s\n", speed)
			continue
		}
		speed /= 1000
		if speed < 1000 {
			fmt.Printf("Speed: %.2fKB/s\n", speed)
			continue
		}
		speed /= 1000
		if speed < 1000 {
			fmt.Printf("Speed: %.2fMB/s\n", speed)
			continue
		}
		speed /= 1000
		fmt.Printf("Speed: %.2fGB/s\n", speed)
	}
}

func start(c chan struct{}) {
	go printSpeed()
	url := info.URL
	chunks := []chunk{}
	chunkable := false
	wg := &sync.WaitGroup{}

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	check(err)
	for _, c := range info.Cookies {
		req.AddCookie(convertCookie(c))
	}
	req.Header.Set("User-Agent", info.Agent)
	resp, err := client.Do(req)
	check(err)
	defer resp.Body.Close()
	finalURL := resp.Request.URL.String()
	fmt.Println("Final url is:", finalURL)

	length := resp.ContentLength
	sizeGlobal = length
	fmt.Println("ContentLength is:", length, "bytes")

	if length == -1 || resp.Header.Get("Accept-Ranges") != "bytes" {
		chunkable = false
		chunks = append(chunks, chunk{
			start: 0,
			size:  length,
			mu:    &sync.Mutex{},
		})
	} else {
		chunkable = true
		chunks = getChunks(length)
	}

	out, err := os.Create("/home/pallav/output.part0")
	check(err)
	defer out.Close()

	chunks[0].grabbed = true
	if chunkable {
		for i := 0; i < 8; i++ {
			go startDownload(chunks, url, wg)
			wg.Add(1)
		}
	}
	contDownload(chunks, resp.Body, 0)
	wg.Wait()
	c <- struct{}{}
}

func startDownload(chunks []chunk, url string, wg *sync.WaitGroup) {
	defer wg.Done()

	i := grabChunk(chunks)
	if i == -1 {
		return
	}
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	check(err)
	for _, c := range info.Cookies {
		req.AddCookie(convertCookie(c))
	}
	req.Header.Set("User-Agent", info.Agent)

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", chunks[i].start))

	resp, err := client.Do(req)
	if err != nil || resp.ContentLength != sizeGlobal-chunks[i].start {
		ungrab(chunks, i)
		return
	}

	contDownload(chunks, resp.Body, i)
}

func contDownload(chunks []chunk, in io.Reader, index int) {
	fmt.Println("Started for index", index)
	size := chunks[index].size
	filename := fmt.Sprintf("/home/pallav/output.part%d", index)
	out, err := os.Create(filename)
	check(err)
	defer out.Close()

	if index == -1 || index > len(chunks) {
		return
	}

	for {
		bytes := int64(4096)
		if size > -1 {
			bytes = min(bytes, size)
		}

		n, err := io.CopyN(out, in, bytes)
		size -= n
		atomic.AddInt64(&totalDown, n)
		if err != nil || size == 0 {
			break
		}
		println(index, "index", chunks[index].size-size, "bytes downloaded")
	}
	println(index, "index", chunks[index].size-size, "bytes downloaded")

	if index == len(chunks)-1 {
		return
	}

	// Now try 3 seconds to grab the next index, or timeout.
	for i := 0; i < 10; i++ {
		if ret := grab(chunks, index+1); ret != -1 {
			contDownload(chunks, in, ret)
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
}
