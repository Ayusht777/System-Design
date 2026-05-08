package main

// create a single hello world http server
// listen on port 8081

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// time.Sleep(15 * time.Second)
		fmt.Fprint(w, "Hello World from Server 1")
	})

	fmt.Println("Server 1 starting on :8081...")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		fmt.Println(err)
	}
}
