package server

import (
	"context"
	"crypto/tls"
	"forum/cmd/config"
	repository "forum/internal/database"
	database "forum/internal/database/migration"
	"forum/internal/service"
	handlers "forum/internal/web/handlers"
	"log"
	"net/http"
	"os"
)

type Server struct {
	httpServer *http.Server
}

func InitServer(conf *config.Config, ctx context.Context) *Server {
	db, err := database.CreateDb(conf.DbDriver, conf.DbPath, ctx)
	if err != nil {
		log.Fatal(err)
	}
	repository := repository.NewRepository(db) // stores the db in the repository
	service := service.NewService(repository)
	handler := handlers.NewHandler(service)

	port := os.Getenv("PORT")
	if port == "" {
		port = "localhost" + conf.Address // Explicitly bind to localhost
	}

	cert, err := tls.LoadX509KeyPair("./tls/cert.pem", "./tls/key.pem")
	if err != nil {
		log.Fatalf("Failed to load TLS certificate and key: %v", err)
	}

	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		CurvePreferences:         []tls.CurveID{tls.CurveP384, tls.CurveP256},
		Certificates:             []tls.Certificate{cert}, // Add the certificate pair
	}

	ServerObj := Server{
		httpServer: &http.Server{
			Addr:      port,
			Handler:   handler.InitRouter(),
			TLSConfig: tlsConfig,
		},
	}

	return &ServerObj
}

func (server *Server) Start() error {
	log.Println("Starting API server at https://" + server.httpServer.Addr)
	return server.httpServer.ListenAndServeTLS("", "")
}
