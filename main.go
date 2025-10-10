package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, req)
	})
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := fmt.Sprintf(
		`<html>
  			<body>
   				<h1>Welcome, Chirpy Admin</h1>
    			<p>Chirpy has been visited %d times!</p>
  			</body>
		</html>`,
		cfg.fileserverHits.Load(),
	)

	_, err := w.Write([]byte(html))
	if err != nil {
		fmt.Printf("failed to write the response body: %v\n", err)
	}
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err := w.Write([]byte("Reset hits"))
	if err != nil {
		fmt.Println("failed to write the response body: %w", err)
	}
}

func handlerValidate_chirp(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"Internal server error"}`))
		return
	}

	if len(params.Body) > 140 {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"Chirp is too long"}`))
		return
	}

	w.WriteHeader(200)
	w.Write([]byte(`{"valid":true}`))
}

func main() {
	apiCfg := apiConfig{}

	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	mux.HandleFunc("POST /api/validate_chirp", handlerValidate_chirp)

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
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
