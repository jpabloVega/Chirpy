package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	"github.com/jpabloVega/Chirpy/internal/database"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Print(err)
	}
	dbQueries := database.New(db)

	const port = ":8080"
	rootFilePath := http.Dir(".")
	ServeMux := http.NewServeMux()
	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		dbQueries:      dbQueries,
	}

	fileServerHandler := http.FileServer(rootFilePath)
	ServeMux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(fileServerHandler)))
	ServeMux.HandleFunc("GET /api/healthz", handlerReadiness)
	ServeMux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	ServeMux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	ServeMux.HandleFunc("POST /api/validate_chirp", apiCfg.createChirp)
	ServeMux.HandleFunc("POST /api/users", apiCfg.addUser)

	server := &http.Server{
		Handler: ServeMux,
		Addr:    port,
	}
	log.Fatal(server.ListenAndServe())
}

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		w.WriteHeader(200)
		next.ServeHTTP(w, req)
	})
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, req *http.Request) {
	res := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %v times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(200)
	w.Write([]byte(res))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)
	err := cfg.dbQueries.ResetUsers(req.Context())
	if err != nil {
		log.Printf("Error deleting users: %v", err)
		return
	}
	w.WriteHeader(200)
	w.Write([]byte("File server hits reset to 0"))
}
