package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"ats-verify/internal/config"
	"ats-verify/internal/handler"
	"ats-verify/internal/middleware"
	"ats-verify/internal/repository"
	"ats-verify/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// --- Database ---
	db, err := repository.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("connected to PostgreSQL")

	// --- Seed ---
	if err := repository.Seed(context.Background(), db, service.HashPassword); err != nil {
		log.Printf("warning: seed failed: %v", err)
	}

	// --- Repositories ---
	userRepo := repository.NewUserRepository(db)
	parcelRepo := repository.NewParcelRepository(db)
	riskRepo := repository.NewRiskRepository(db)
	riskRawRepo := repository.NewRiskRawDataRepository(db)
	ticketRepo := repository.NewTicketRepository(db)

	// --- Services ---
	authService := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.Expiration)
	parcelService := service.NewParcelService(parcelRepo)
	riskService := service.NewRiskService(riskRepo)
	imeiService := service.NewIMEIService()
	ticketService := service.NewTicketService(ticketRepo)
	trackingService := service.NewTrackingService(
		service.NewCDEKTracker(),
		service.NewKazpostTracker(),
	)
	pdfExtractor := service.NewPDFExtractor()
	riskAnalysisService := service.NewRiskAnalysisService(riskRepo, riskRawRepo)

	// --- Handlers ---
	authHandler := handler.NewAuthHandler(authService)
	parcelHandler := handler.NewParcelHandler(parcelService)
	trackHandler := handler.NewTrackHandler(parcelService, trackingService)
	riskHandler := handler.NewRiskHandler(riskService)
	imeiHandler := handler.NewIMEIHandler(imeiService, pdfExtractor)
	ticketHandler := handler.NewTicketHandler(ticketService)
	riskAnalysisHandler := handler.NewRiskAnalysisHandler(riskAnalysisService)

	// --- Router ---
	mux := http.NewServeMux()

	// Health check (no auth)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Auth routes (some require auth info)
	authMw := middleware.RequireAuth(cfg.JWT.Secret)
	authHandler.RegisterRoutes(mux, authMw)

	// Protected routes (with JWT middleware)
	parcelHandler.RegisterRoutes(mux, authMw)
	trackHandler.RegisterRoutes(mux, authMw)
	riskHandler.RegisterRoutes(mux, authMw)
	imeiHandler.RegisterRoutes(mux, authMw)
	ticketHandler.RegisterRoutes(mux, authMw)
	riskAnalysisHandler.RegisterRoutes(mux, authMw)

	// --- Attachments (Static serving) ---
	// Note: In a real app this would be under authMw or signed URLs. Serving publicly for MVP.
	mux.Handle("GET /api/v1/attachments/", http.StripPrefix("/api/v1/attachments/", http.FileServer(http.Dir("uploads"))))

	// --- SPA Static Files (production) ---
	// Serve frontend from web/dist if it exists.
	// In dev mode (Vite proxy), this directory won't exist.
	staticDir := "web/dist"
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		fs := http.FileServer(http.Dir(staticDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Try to serve static file first.
			path := staticDir + r.URL.Path
			if _, err := os.Stat(path); err == nil {
				fs.ServeHTTP(w, r)
				return
			}
			// Fallback to index.html for SPA client-side routing.
			http.ServeFile(w, r, staticDir+"/index.html")
		})
		log.Println("serving SPA from", staticDir)
	}

	// --- Server ---
	addr := ":" + cfg.Server.Port
	log.Printf("ATS-Verify server starting on %s", addr)

	wrappedMux := middleware.CORS(mux)
	if err := http.ListenAndServe(addr, wrappedMux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
