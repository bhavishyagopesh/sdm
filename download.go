package main

import (
	"container/list"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var counter = 15

func (dl *Download) CreateRequest(start int64) (*http.Request, error) {
	req, err := http.NewRequest("GET", dl.URL, nil)
	if err != nil {
		return nil, err
	}
	for _, c := range dl.Cookies {
		req.AddCookie(convertCookie(c))
	}
	req.Header.Set("User-Agent", dl.Agent)
	if start != 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", start))
	}
	return req, nil
}

func (dl *Download) SetFileName(resp *http.Response) {
	finalURL := resp.Request.URL.String()
	pieces := strings.Split(finalURL, "/")
	for pieces[len(pieces)-1] == "" {
		pieces = pieces[:len(pieces)-1]
	}
	name := []byte(pieces[len(pieces)-1])
	for i, ch := range name {
		if isChar(ch) || ch == '.' {
			continue
		}
		name[i] = '_'
	}
	dl.FileName = string(name)
}

func (dl *Download) Start(done chan error) {
	termClear()
	dl.StartTime = time.Now()

	req, err := dl.CreateRequest(0)
	if err != nil {
		done <- err
		return
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		done <- err
		return
	}
	dl.SetFileName(resp)
	dl.FileSize = resp.ContentLength

	if dl.Resumable = DEF; dl.FileSize == -1 {
		dl.Resumable = NO
	} else if resp.Header.Get("Accept-Ranges") == "bytes" {
		dl.Resumable = YES
	}
	dl.CreateChunks()
	go dl.PrintInformation()
	go dl.ContinueDownload(dl.Chunks.Front(), resp.Body)
	done <- dl.ThreadSpawner()
}

func (dl *Download) PrintInformation() {
	for {
		termPos(0, 0)
		for i := 0; i < 10; i++ {
			fmt.Println(
				"									   ",
				"									   ",
				"									   ")
		}
		termPos(0, 0)
		fmt.Printf("FileSize: %6s\n", HumanReadable(dl.FileSize))
		fmt.Printf("Speed: %-12s\n", humanReadable(dl.AvgSpeed())+"/s")
		for e := dl.Chunks.Front(); !e.Value.(*chunk).Last(); e = e.Next() {
			curr := e.Value.(*chunk)
			fmt.Printf(
				"Start: %10s  Downld: %10s  Suffix: %10s  Active: %5v\n",
				HumanReadable(curr.start),
				HumanReadable(curr.downld),
				curr.suffix, curr.active,
			)
		}
		time.Sleep(400 * time.Millisecond)
	}
}

func (dl *Download) AvgSpeed() float64 {
	dld := 0.0
	for e := dl.Chunks.Front(); !e.Value.(*chunk).Last(); e = e.Next() {
		dld += float64(e.Value.(*chunk).downld)
	}
	tDiff := float64(time.Now().Sub(dl.StartTime).Seconds())
	return dld / tDiff
}

func (dl *Download) SaveFinalFile() error {
	var out, f *os.File
	var err error
	filename := fmt.Sprintf("/home/pallav/%s", dl.FileName)
	if out, err = os.Create(filename); err != nil {
		return err
	}
	for e := dl.Chunks.Front(); !e.Value.(*chunk).Last(); e = e.Next() {
		file := dl.FileName + e.Value.(*chunk).suffix
		file = fmt.Sprintf("/home/pallav/%s", file)
		diff := e.Next().Value.(*chunk).start - e.Value.(*chunk).start
		if f, err = os.Open(file); err != nil {
			return err
		}
		if _, err = io.CopyN(out, f, diff); err != nil {
			return err
		}
		if err = os.Remove(file); err != nil {
			return err
		}
	}
	return nil
}

func (dl *Download) ThreadSpawner() error {
OUTER:
	for {
		if dl.Resumable == NO {
			for dl.Chunks.Front().Value.(*chunk).active {
				time.Sleep(100 * time.Millisecond)
			}
			return nil
		}
		if dl.GetActiveCount() >= dl.MaxThrds {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// We should spawn another thread.
		var maxDiff int64
		var maxElem *list.Element
		var isActive bool
		for e := dl.Chunks.Front(); !e.Value.(*chunk).Last(); e = e.Next() {
			curr := e.Value.(*chunk)
			next := e.Next().Value.(*chunk)
			diff := next.start - (curr.start + curr.downld)

			if !curr.active && curr.downld == 0 {
				curr.active = true
				go dl.Resume(e)
				continue OUTER
			}
			if diff > maxDiff {
				isActive = curr.active
				maxDiff = diff
				maxElem = e
			}
		}

		if maxDiff <= 0 {
			return nil // The download has been completed.
		}
		toInsert := &chunk{
			start:  maxElem.Next().Value.(*chunk).start - maxDiff,
			active: false,
			suffix: fmt.Sprintf(".part%d", dl.Counter),
			mu:     &sync.Mutex{},
		}
		if isActive {
			toInsert.start = maxElem.Next().Value.(*chunk).start - maxDiff/2
			if maxDiff <= 128*1024 {
				continue // Don't split if less than 128KB is left
			}
		}

		dl.Chunks.InsertAfter(toInsert, maxElem)
		dl.Counter += 1
	}
	return nil
}

func (dl *Download) Resume(e *list.Element) {
	ch := e.Value.(*chunk)
	defer func() { ch.active = false }()
	req, err := dl.CreateRequest(ch.start)
	if err != nil {
		return
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return
	}
	if resp.ContentLength != dl.FileSize-ch.start {
		dl.Resumable = NO
		return
	}
	termPos(counter, 0)
	counter += 1
	fmt.Println("Here for: ", ch.suffix)
	dl.ContinueDownload(e, resp.Body)
}

func (dl *Download) GetActiveCount() int {
	count := 0
	for e := dl.Chunks.Front(); e != nil; e = e.Next() {
		if e.Value.(*chunk).active {
			count += 1
		}
	}
	return count
}

func (dl *Download) CreateChunks() {
	dl.Chunks.Init()
	dl.Chunks.PushBack(&chunk{
		start:  0,
		active: true,
		suffix: ".part0",
		mu:     &sync.Mutex{},
	})
	dl.Chunks.PushBack(&chunk{
		start: dl.FileSize,
		mu:    &sync.Mutex{},
	})
	dl.Counter += 1
}

//func start(c chan struct{}) {
//go printSpeed()
//url := info.URL
//chunks := []chunk{}
//chunkable := false
//wg := &sync.WaitGroup{}

//client := &http.Client{}
//req, err := http.NewRequest("GET", url, nil)
//check(err)
//for _, c := range info.Cookies {
//req.AddCookie(convertCookie(c))
//}
//req.Header.Set("User-Agent", info.Agent)
//resp, err := client.Do(req)
//check(err)
//defer resp.Body.Close()
//finalURL := resp.Request.URL.String()
//fmt.Println("Final url is:", finalURL)

//length := resp.ContentLength
//sizeGlobal = length
//fmt.Println("ContentLength is:", length, "bytes")

//if length == -1 || resp.Header.Get("Accept-Ranges") != "bytes" {
//chunkable = false
//chunks = append(chunks, chunk{
//start: 0,
//size:  length,
//mu:    &sync.Mutex{},
//})
//} else {
//chunkable = true
//chunks = getChunks(length)
//}

//out, err := os.Create("/home/pallav/output.part0")
//check(err)
//defer out.Close()

//chunks[0].grabbed = true
//if chunkable {
//for i := 0; i < 8; i++ {
//go startDownload(chunks, url, wg)
//wg.Add(1)
//}
//}
//contDownload(chunks, resp.Body, 0)
//wg.Wait()
//c <- struct{}{}
//}

//func startDownload(chunks []chunk, url string, wg *sync.WaitGroup) {
//defer wg.Done()

//i := grabChunk(chunks)
//if i == -1 {
//return
//}
//client := &http.Client{}
//req, err := http.NewRequest("GET", url, nil)
//check(err)
//for _, c := range info.Cookies {
//req.AddCookie(convertCookie(c))
//}
//req.Header.Set("User-Agent", info.Agent)

//req.Header.Set("Range", fmt.Sprintf("bytes=%d-", chunks[i].start))

//resp, err := client.Do(req)
//if err != nil || resp.ContentLength != sizeGlobal-chunks[i].start {
//ungrab(chunks, i)
//return
//}

//contDownload(chunks, resp.Body, i)
//}

func (dl *Download) ContinueDownload(elem *list.Element, body io.ReadCloser) {
	filename := dl.FileName + elem.Value.(*chunk).suffix
	filename = fmt.Sprintf("/home/pallav/%s", filename)
	curChunk := elem.Value.(*chunk)
	out, err := os.Create(filename)
	check(err) // FIXME: Handle error gracefully.

	defer func() {
		body.Close()
		out.Close()
		curChunk.active = false
	}()

	for {
		n, err := io.CopyN(out, body, 4096)
		curChunk.downld += n
		if err != nil {
			break
		}
		if n := elem.Next().Value.(*chunk).start; n == -1 {
			continue
		} else if curChunk.start+curChunk.downld >= n {
			break
		}
	}
}

//func contDownload(chunks []chunk, in io.Reader, index int) {
//fmt.Println("Started for index", index)
//size := chunks[index].size
//filename := fmt.Sprintf("/home/pallav/output.part%d", index)
//out, err := os.Create(filename)
//check(err)
//defer out.Close()

//if index == -1 || index > len(chunks) {
//return
//}

//for {
//bytes := int64(4096)
//if size > -1 {
//bytes = min(bytes, size)
//}

//n, err := io.CopyN(out, in, bytes)
//size -= n
//atomic.AddInt64(&totalDown, n)
//if err != nil || size == 0 {
//break
//}
//println(index, "index", chunks[index].size-size, "bytes downloaded")
//}
//println(index, "index", chunks[index].size-size, "bytes downloaded")

//if index == len(chunks)-1 {
//return
//}

//Now try 3 seconds to grab the next index, or timeout.
//for i := 0; i < 10; i++ {
//if ret := grab(chunks, index+1); ret != -1 {
//contDownload(chunks, in, ret)
//break
//}
//time.Sleep(300 * time.Millisecond)
//}
//}
