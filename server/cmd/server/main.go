package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/waylontech/backup/pkg/crypto"
	"github.com/waylontech/backup/server/internal/api"
	"github.com/waylontech/backup/server/internal/config"
	"github.com/waylontech/backup/server/internal/db"
	"github.com/waylontech/backup/server/internal/storage"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config", zap.Error(err))
	}

	ctx := context.Background()

	database, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("db connect", zap.Error(err))
	}
	defer database.Close()

	if err := database.Migrate(cfg.DatabaseURL); err != nil {
		log.Fatal("migrate", zap.Error(err))
	}
	log.Info("migrations applied")

	if err := bootstrap(ctx, database, cfg, log); err != nil {
		log.Fatal("bootstrap", zap.Error(err))
	}

	srv := api.NewServer(database, cfg.JWTSecret, cfg.MasterKey, log, cfg.CORSOrigins)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("listening", zap.String("addr", cfg.HTTPAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen", zap.Error(err))
		}
	}()

	<-quit
	log.Info("shutting down")

	shutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutCtx); err != nil {
		log.Error("shutdown", zap.Error(err))
	}
}

func bootstrap(ctx context.Context, database *db.DB, cfg *config.Config, log *zap.Logger) error {
	_, err := database.GetUserByEmail(ctx, cfg.AdminEmail)
	if err == nil {
		return nil
	}

	log.Info("bootstrapping vendor tenant and admin user")

	dek, err := crypto.GenerateDEK()
	if err != nil {
		return err
	}
	dekEnc, err := crypto.EncryptDEK(cfg.MasterKey, dek)
	if err != nil {
		return err
	}

	tenant, err := database.CreateTenant(ctx, "Waylon Technologies", "waylon", dekEnc)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = database.CreateUser(ctx, tenant.ID, cfg.AdminEmail, string(hash), "vendor_admin", "Admin")
	if err != nil {
		return err
	}

	if os.Getenv("DEFAULT_STORAGE_ENDPOINT") != "" {
		creds := storage.Credentials{
			AccessKeyID:     os.Getenv("DEFAULT_STORAGE_ACCESS_KEY"),
			SecretAccessKey: os.Getenv("DEFAULT_STORAGE_SECRET_KEY"),
		}
		credsEnc, err := storage.EncryptCredentials(cfg.MasterKey, creds)
		if err != nil {
			return err
		}
		_, err = database.CreateStorageTarget(ctx,
			tenant.ID, "primary", "s3",
			os.Getenv("DEFAULT_STORAGE_ENDPOINT"),
			os.Getenv("DEFAULT_STORAGE_BUCKET"),
			os.Getenv("DEFAULT_STORAGE_REGION"),
			credsEnc, true,
			os.Getenv("DEFAULT_STORAGE_PATH_STYLE") == "true",
		)
		if err != nil {
			return err
		}

		sc := storage.New(
			os.Getenv("DEFAULT_STORAGE_ENDPOINT"),
			os.Getenv("DEFAULT_STORAGE_REGION"),
			os.Getenv("DEFAULT_STORAGE_BUCKET"),
			creds.AccessKeyID, creds.SecretAccessKey,
			os.Getenv("DEFAULT_STORAGE_PATH_STYLE") == "true",
		)
		if err := sc.EnsureBucket(ctx); err != nil {
			log.Warn("ensure bucket", zap.Error(err))
		}
	}

	log.Info("bootstrap complete",
		zap.String("tenant_id", tenant.ID),
		zap.String("admin_email", cfg.AdminEmail),
	)
	return nil
}
