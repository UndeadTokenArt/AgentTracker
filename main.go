package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

//go:embed templates/* static/**
var embeddedFS embed.FS

func loadTemplates() *template.Template {
	t := template.New("").Funcs(template.FuncMap{})
	tpl, err := t.ParseFS(embeddedFS, "templates/*.tmpl")
	if err != nil {
		log.Fatalf("parse templates: %v", err)
	}
	return tpl
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Templates
	r.SetHTMLTemplate(loadTemplates())

	// Static
	sub, err := fs.Sub(embeddedFS, "static")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}
	r.StaticFS("/static", http.FS(sub))

	registerRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
