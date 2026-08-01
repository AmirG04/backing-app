package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"banking-app/backend/internal/config"
	"banking-app/backend/internal/db"
	"banking-app/backend/internal/handlers"
	appmiddleware "banking-app/backend/internal/middleware"
)

func main() {
	cfg := config.Load()

	pgPool, err := db.NewPostgresPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("fatal: %v", err)
	}
	defer pgPool.Close()

	if err := db.Migrate(pgPool); err != nil {
		log.Fatalf("fatal: %v", err)
	}

	clusterID, err := strconv.ParseUint(cfg.TigerBeetleCluster, 10, 64)
	if err != nil {
		log.Fatalf("fatal: TIGERBEETLE_CLUSTER_ID invalido: %v", err)
	}

	tbClient, err := db.NewTigerBeetleClient(clusterID, []string{cfg.TigerBeetleAddress})
	if err != nil {
		log.Fatalf("fatal: %v", err)
	}
	defer tbClient.Close()

	if err := tbClient.EnsureBankAccount(); err != nil {
		log.Fatalf("fatal: no se pudo inicializar la cuenta del banco: %v", err)
	}

	h := handlers.New(pgPool, tbClient, cfg.JWTSecret)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // en produccion: restringir al dominio del frontend
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/logout", h.Logout)
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(appmiddleware.Auth(cfg.JWTSecret))

		r.Get("/accounts/me", h.GetAccountInfo)
		r.Get("/accounts/balance", h.GetBalance)

		r.Post("/transactions/deposit", h.Deposit)
		r.Post("/transactions/withdraw", h.Withdraw)
		r.Post("/transactions/transfer", h.Transfer)
		r.Get("/transactions/history", h.History)

		r.Post("/chat", h.Chat) // chat con IA (MCP) - ver internal/handlers/chat.go
	})

	log.Printf("servidor escuchando en :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
