package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// START OMIT
func main() {
	runtime.GOMAXPROCS(4)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			time.Sleep(time.Second * 2)
			fmt.Println(i)
		}(i)
	}
	for i := 'a'; i < 'e'; i++ {
		wg.Add(1)
		go func(i rune) {
			defer wg.Done()
			time.Sleep(time.Second * 2)
			fmt.Printf("%c\n", i)
		}(i)
	}
	wg.Wait()
}

//END OMIT
