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
			defer wg.Done()
			fmt.Println(i)
		}(i)
	}
	for i := 'a'; i < 'e'; i++ {
		wg.Add(1)
		go func(i rune) {
			defer wg.Done()
			fmt.Printf("%c\n", i)
		}(i)
	}
	wg.Wait()
}
