package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, i int) {
			fmt.Println(i)
			wg.Done()
		}(&wg, i)
	}
	for i := 'a'; i < 'e'; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, i rune) {
			fmt.Printf("%c\n", i)
			defer wg.Done()
		}(&wg, i)
	}
	wg.Wait()
}
