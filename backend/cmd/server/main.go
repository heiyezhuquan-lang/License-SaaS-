package main

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"license-saas/backend/internal/api"
	"license-saas/backend/internal/db"
	"license-saas/backend/internal/util"

	"github.com/gin-gonic/gin"
)

//go:embed webdist/*
var embeddedFrontend embed.FS

func main() {
	addr := util.Env("APP_ADDR", ":8080")
	dbPath := util.Env("APP_DB", "./license-saas.db")
	jwtSecret := util.Env("APP_JWT_SECRET", "dev-secret-change-me")
	adminUser := os.Getenv("APP_ADMIN_USER")
	adminPass := os.Getenv("APP_ADMIN_PASS")

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	if err := db.Migrate(database, adminUser, adminPass); err != nil {
		log.Fatal(err)
	}

	router := api.New(database, jwtSecret, util.Env("APP_CLIENT_SIGN_SECRET", "demo-client-secret-change-me")).Router()
	mountFrontend(router, os.Getenv("APP_FRONTEND_DIST"))
	log.Printf("License SaaS listening on %s, db=%s", addr, dbPath)
	if err := router.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func mountFrontend(router *gin.Engine, dist string) {
	if dist != "" {
		index := filepath.Join(dist, "index.html")
		if _, err := os.Stat(index); err == nil {
			router.StaticFS("/assets", http.Dir(filepath.Join(dist, "assets")))
			if _, err := os.Stat(filepath.Join(dist, "qq-group-465663266.png")); err == nil {
				router.StaticFile("/qq-group-465663266.png", filepath.Join(dist, "qq-group-465663266.png"))
			}
			router.GET("/", func(c *gin.Context) { c.File(index) })
			router.NoRoute(func(c *gin.Context) {
				if strings.HasPrefix(c.Request.URL.Path, "/api/") {
					c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "api not found"})
					return
				}
				c.File(index)
			})
			log.Printf("frontend mounted from %s", dist)
			return
		}
		log.Printf("frontend dist not found at %s, falling back to embedded frontend", dist)
	}

	frontend, err := fs.Sub(embeddedFrontend, "webdist")
	if err != nil {
		log.Printf("embedded frontend unavailable: %v", err)
		return
	}
	assets, err := fs.Sub(frontend, "assets")
	if err == nil {
		router.StaticFS("/assets", http.FS(assets))
	}
	router.GET("/qq-group-465663266.png", func(c *gin.Context) {
		c.FileFromFS("qq-group-465663266.png", http.FS(frontend))
	})
	renderIndex := func(c *gin.Context) {
		f, err := frontend.Open("index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "index.html not found")
			return
		}
		defer f.Close()
		c.Header("Content-Type", "text/html; charset=utf-8")
		_, _ = io.Copy(c.Writer, f)
	}
	router.GET("/", renderIndex)
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "api not found"})
			return
		}
		renderIndex(c)
	})
	log.Printf("frontend mounted from embedded webdist")
}
