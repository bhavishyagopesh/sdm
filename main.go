package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

var chunkable bool

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

type chunk struct {
	start   int64
	size    int64
	grabbed bool
	mu      *sync.Mutex
}

var chunks []chunk

func getChunks(length int64) []chunk {
	chunks := make([]chunk, 0, 8)
	chunksize := max(length/8, 2*1024*1024)
	offset := int64(0)
	for length > 0 {
		chunks = append(chunks, chunk{
			start: offset,
			size:  min(length, chunksize),
			mu:    &sync.Mutex{},
		})
		length -= chunks[len(chunks)-1].size
		offset += chunks[len(chunks)-1].size
	}
	return chunks
}

func start(url string, c chan struct{}) {
	wg := &sync.WaitGroup{}
	resp, err := http.Get(url)
	check(err)
	defer resp.Body.Close()

	length := resp.ContentLength
	fmt.Println(length)

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
			go startDownload(url, wg)
			wg.Add(1)
		}
	}
	contDownload(resp.Body, 0)
	wg.Wait()
	c <- struct{}{}
}

func startDownload(url string, wg *sync.WaitGroup) {
	defer wg.Done()

	i := grabChunk()
	if i == -1 {
		return
	}
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	check(err)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", chunks[i].start,
		chunks[i].start+chunks[i].size-1))

	res, err := client.Do(req)
	check(err)

	if res.ContentLength != chunks[i].size {
		ungrab(i)
		return
	}

	contDownload(res.Body, i)
}

func ungrab(i int) {
	chunks[i].mu.Lock()
	chunks[i].grabbed = false
	chunks[i].mu.Unlock()
}

func grab(i int) int {
	chunks[i].mu.Lock()
	if !chunks[i].grabbed {
		chunks[i].grabbed = true
		chunks[i].mu.Unlock()
		return i
	}
	chunks[i].mu.Unlock()
	return -1
}

func grabChunk() int {
	for i, _ := range chunks {
		if ret := grab(i); ret != -1 {
			return ret
		}
	}
	return -1
}

func contDownload(in io.Reader, index int) {
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
		if size != -1 {
			bytes = min(bytes, size)
		}

		n, err := io.CopyN(out, in, bytes)
		size -= n
		if err != nil || size == 0 {
			break
		}
		println(index, "index", chunks[index].size-size, "bytes downloaded")
	}

	if index == len(chunks)-1 {
		return
	}

	// Now try 3 seconds to grab the next index, or timeout.
	for i := 0; i < 10; i++ {
		if ret := grab(index + 1); ret != -1 {
			contDownload(in, ret)
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func main() {
	url := os.Args[1]
	c := make(chan struct{})
	go start(url, c)
	<-c
}
