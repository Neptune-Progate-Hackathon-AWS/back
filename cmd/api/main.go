package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	api "github.com/Neptune-Progate-Hackathon-AWS/back/internal/api"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	server := handler.NewServer()
	h := api.HandlerFromMux(api.NewStrictHandler(server, nil), r)

	addr := ":8080"
	fmt.Printf("Server listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, h))
}
