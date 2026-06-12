// Command vpanel is the single-binary VPN panel: HTTP API, embedded SPA,
// core supervision, traffic accounting and the subscription service.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/egorbr4z/panel/internal/api"
	"github.com/egorbr4z/panel/internal/api/handler"
	"github.com/egorbr4z/panel/internal/config"
	"github.com/egorbr4z/panel/internal/core"
	"github.com/egorbr4z/panel/internal/database"
	"github.com/egorbr4z/panel/internal/models"
	"github.com/egorbr4z/panel/internal/service/auth"
)

func main() {
	cfg := config.Load()

	// Subcommands: `vpanel admin create -u NAME -p PASS [-role sudo]`
	if len(os.Args) > 1 && os.Args[1] == "admin" {
		runAdminCmd(cfg, os.Args[2:])
		return
	}

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	jm := auth.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)

	// Core supervisor: launches/reconciles Xray + sing-box from the DB.
	mgr := core.NewManager(db, cfg)
	if err := mgr.Start(); err != nil {
		log.Printf("core manager start: %v", err)
	}

	router := api.NewRouter(cfg, db, jm, mgr)

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("vpanel %s listening on %s", handler.Version, srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	mgr.Stop()
}

// runAdminCmd handles the `admin create` subcommand used by install.sh and ops.
func runAdminCmd(cfg *config.Config, args []string) {
	if len(args) < 1 || args[0] != "create" {
		fmt.Println("usage: vpanel admin create -u <username> -p <password> [-role sudo|admin]")
		os.Exit(2)
	}

	fs := flag.NewFlagSet("create", flag.ExitOnError)
	username := fs.String("u", "", "admin username")
	password := fs.String("p", "", "admin password")
	role := fs.String("role", "sudo", "admin role: sudo or admin")
	_ = fs.Parse(args[1:])

	if *username == "" || *password == "" {
		fmt.Println("error: -u and -p are required")
		os.Exit(2)
	}

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	hash, err := auth.HashPassword(*password)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	admin := models.Admin{
		Username:     *username,
		PasswordHash: hash,
		Role:         models.AdminRole(*role),
		IsActive:     true,
	}
	// Upsert by username so re-running resets the password (handy for ops).
	var existing models.Admin
	if err := db.Where("username = ?", *username).First(&existing).Error; err == nil {
		existing.PasswordHash = hash
		existing.Role = admin.Role
		existing.IsActive = true
		if err := db.Save(&existing).Error; err != nil {
			log.Fatalf("update admin: %v", err)
		}
		fmt.Printf("admin %q updated\n", *username)
		return
	}
	if err := db.Create(&admin).Error; err != nil {
		log.Fatalf("create admin: %v", err)
	}
	fmt.Printf("admin %q created with role %s\n", *username, *role)
}
