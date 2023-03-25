package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			fmt.Println(i)
			wg.Done()
		}(i)
	}
	for i := 'a'; i < 'e'; i++ {
		wg.Add(1)
		go func(i rune) {
			fmt.Printf("%c\n", i)
			defer wg.Done()
		}(i)
	}
	wg.Wait()
}
