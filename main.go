package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/LouisRemes-95/chirpy.git/internal/auth"
	"github.com/LouisRemes-95/chirpy.git/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type user struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
	secret         string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, req)
	})
}

// Helper functions

func respondWithError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	_, err := w.Write([]byte(`{"error":"` + msg + `"}`))
	if err != nil {
		log.Printf("failed to write response %s", err)
	}
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("failed to marshal JSON: %s", err)
		respondWithError(w, 500, "Internal server error")
		return
	}
	w.WriteHeader(code)
	w.Write(data)
}

// Handler functions

func (cfg *apiConfig) handlerGetMetrics(w http.ResponseWriter, req *http.Request) {
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

func (cfg *apiConfig) handlerPostReset(w http.ResponseWriter, req *http.Request) {

	if cfg.platform != "dev" {
		respondWithError(w, 403, "No dev authorisation")
		return
	}

	err := cfg.dbQueries.DeleteUsers(req.Context())
	if err != nil {
		log.Printf("failed to delete users %s", err)
		respondWithError(w, 500, "Internal server error")
		return
	}

	w.WriteHeader(200)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err = w.Write([]byte("Reset"))
	if err != nil {
		log.Printf("failed to write response %s", err)
		return
	}

	cfg.fileserverHits.Store(0)
}

func handlerGetHealthz(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err := w.Write([]byte("OK"))
	if err != nil {
		fmt.Println("failed to write the response body: %w", err)
	}
}

func (cfg *apiConfig) handlerPostUser(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("failed to decode parameters: %s", err)
		respondWithError(w, 500, "Internal server error")
		return
	}

	HashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("failed to hash password: %s", err)
		respondWithError(w, 500, "Internal server error")
		return
	}

	myParams := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: HashedPassword,
	}

	createdUser, err := cfg.dbQueries.CreateUser(req.Context(), myParams)
	if err != nil {
		log.Printf("failed to create user: %s", err)
		respondWithError(w, 500, "Internal server error")
		return
	}

	respBody := user{
		ID:        createdUser.ID,
		CreatedAt: createdUser.CreatedAt,
		UpdatedAt: createdUser.UpdatedAt,
		Email:     createdUser.Email,
	}

	respondWithJSON(w, 201, respBody)
}

func (cfg *apiConfig) handlerPostChirp(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	tokenString, err := auth.GetBearerToken(req.Header)
	if err != nil {
		log.Printf("failed to get bearer token: %v", err)
		respondWithError(w, 401, "Unauthorized")
		return
	}

	userID, err := auth.ValidateJWT(tokenString, cfg.secret)
	if err != nil {
		log.Printf("failed to validate token string: %v", err)
		respondWithError(w, 401, "Unauthorized")
		return
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("failed to decode parameters: %s", err)
		respondWithError(w, 500, "Internal server error")
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	chirpParams := database.CreateChirpParams{
		Body:   cleanMessage(params.Body),
		UserID: userID,
	}

	createdChirp, err := cfg.dbQueries.CreateChirp(req.Context(), chirpParams)
	if err != nil {
		log.Printf("failed to create chirp: %s", err)
		respondWithError(w, 500, "Internal server error")
		return
	}

	respBody := chirp{
		ID:        createdChirp.ID,
		CreatedAt: createdChirp.CreatedAt,
		UpdatedAt: createdChirp.UpdatedAt,
		Body:      createdChirp.Body,
		UserID:    createdChirp.UserID,
	}

	respondWithJSON(w, 201, respBody)
}

func cleanMessage(s string) string {
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Split(s, " ")
	for i, word := range words {
		if slices.Contains(badWords, strings.ToLower(word)) {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.dbQueries.GetChirps(req.Context())
	if err != nil {
		log.Printf("failed to get chirps: %s", err)
		respondWithError(w, 500, "Internal server error")
		return
	}

	respBody := make([]chirp, len(chirps))
	for i, currentChirp := range chirps {
		respBody[i] = chirp{
			ID:        currentChirp.ID,
			CreatedAt: currentChirp.CreatedAt,
			UpdatedAt: currentChirp.UpdatedAt,
			Body:      currentChirp.Body,
			UserID:    currentChirp.UserID,
		}
	}

	respondWithJSON(w, 200, respBody)
}

func (cfg *apiConfig) handlerGetChirpsByID(w http.ResponseWriter, req *http.Request) {
	chirpID, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil {
		log.Printf("failed to parse chirpID string to uuid: %s", err)
		respondWithError(w, 400, "Invalid chirp ID")
		return
	}

	chirpByID, err := cfg.dbQueries.GetChirpByID(req.Context(), chirpID)
	switch err {
	case nil:
	case sql.ErrNoRows:
		log.Printf("failed to get chirp, Id not found: %s", err)
		respondWithError(w, 404, "Chirp not found")
		return
	default:
		log.Printf("failed to get chirp: %s", err)
		respondWithError(w, 500, "Internal server error")
		return
	}

	respBody := chirp{
		ID:        chirpByID.ID,
		CreatedAt: chirpByID.CreatedAt,
		UpdatedAt: chirpByID.UpdatedAt,
		Body:      chirpByID.Body,
		UserID:    chirpByID.UserID,
	}

	respondWithJSON(w, 200, respBody)
}

func (cfg *apiConfig) handlerPostLogin(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Password         string `json:"password"`
		Email            string `json:"email"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("failed to decode parameters: %v", err)
		respondWithError(w, 500, "Internal server error")
		return
	}

	if s := params.ExpiresInSeconds; s == 0 || s > 3600 {
		params.ExpiresInSeconds = 3600
	}

	returnedUser, err := cfg.dbQueries.GetUsetByEmail(req.Context(), params.Email)
	if err != nil {
		log.Printf("failed to get the user by email: %v", err)
		respondWithError(w, 401, "Unauthorized")
		return
	}

	match, err := auth.CheckPasswordHash(params.Password, returnedUser.HashedPassword)
	if !match || err != nil {
		log.Printf("failed to check password: %v", err)
		respondWithError(w, 401, "Unauthorized")
		return
	}

	print()
	token, err := auth.MakeJWT(returnedUser.ID, cfg.secret, time.Duration(params.ExpiresInSeconds)*time.Second)
	if err != nil {
		log.Printf("failed to make JWT token: %v", err)
		respondWithError(w, 500, "Internal server error")
		return
	}

	respBody := struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
		Token     string    `json:"token"`
	}{
		ID:        returnedUser.ID,
		CreatedAt: returnedUser.CreatedAt,
		UpdatedAt: returnedUser.UpdatedAt,
		Email:     returnedUser.Email,
		Token:     token,
	}

	respondWithJSON(w, 200, respBody)
}

// MAIN

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("failed to load the environment: %w", err)
	}

	dbUrl := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatalln("failed to open database: %w", err)
	}

	dbQueries := database.New(db)

	apiCfg := apiConfig{}
	apiCfg.dbQueries = dbQueries
	apiCfg.platform = os.Getenv("PLATFORM")
	apiCfg.secret = os.Getenv("secret")

	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerGetMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerPostReset)
	mux.HandleFunc("GET /api/healthz", handlerGetHealthz)
	mux.HandleFunc("POST /api/users", apiCfg.handlerPostUser)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerPostChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirpsByID)
	mux.HandleFunc("POST /api/login", apiCfg.handlerPostLogin)

	svr := &http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	err = svr.ListenAndServe()
	if err != nil {
		log.Fatalln("failed to listen and serve: %w", err)
	}

}
