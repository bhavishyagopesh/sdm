package main

import (
	"sync"
)

type chunk struct {
	start  int64
	downld int64
	active bool
	suffix string
	mu     *sync.Mutex
}

func (c *chunk) Last() bool {
	return c.suffix == ""
}

//
// func ungrab(chunks []chunk, i int) {
// 	chunks[i].mu.Lock()
// 	chunks[i].grabbed = false
// 	chunks[i].mu.Unlock()
// }
//
// func grab(chunks []chunk, i int) int {
// 	chunks[i].mu.Lock()
// 	if !chunks[i].grabbed {
// 		chunks[i].grabbed = true
// 		chunks[i].mu.Unlock()
// 		return i
// 	}
// 	chunks[i].mu.Unlock()
// 	return -1
// }
//
// func grabChunk(chunks []chunk) int {
// 	for i, _ := range chunks {
// 		if ret := grab(chunks, i); ret != -1 {
// 			return ret
// 		}
// 	}
// 	return -1
// }
//
// func getChunks(length int64) []chunk {
// 	chunks := make([]chunk, 0, 8)
// 	chunksize := max(int64(math.Ceil(float64(length)/8)), 1024*1024)
// 	offset := int64(0)
// 	for length > 0 {
// 		chunks = append(chunks, chunk{
// 			start: offset,
// 			size:  min(length, chunksize),
// 			mu:    &sync.Mutex{},
// 		})
// 		length -= chunks[len(chunks)-1].size
// 		offset += chunks[len(chunks)-1].size
// 	}
// 	fmt.Println("Chunks are:", chunks)
// 	return chunks
// }
