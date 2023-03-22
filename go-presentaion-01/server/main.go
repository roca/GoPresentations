// Serving http://localhost:8080/world
package main

import (
	"fmt"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s.", r.URL.Path[1:])
}

func main() {
	err := http.ListenAndServe(":8081", http.HandlerFunc(handler))
	if err != nil {
		panic(err)
	}
}
