package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			fmt.Println("failed to write the response body: %w", err)
		}
	})

	svr := &http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	err := svr.ListenAndServe()
	if err != nil {
		log.Fatalln("failed to listen and serve: %w", err)
	}

}
