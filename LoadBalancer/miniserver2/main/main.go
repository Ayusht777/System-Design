package main

// create a single hello world http server
// listen on port 8082

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello World from Server 2")
	})

	fmt.Println("Server 2 starting on :8082...")
	if err := http.ListenAndServe(":8082", nil); err != nil {
		fmt.Println(err)
	}
}
