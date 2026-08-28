package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"go.uber.org/zap"

	"github.com/waylontech/backup/agent/internal/backup"
	"github.com/waylontech/backup/agent/internal/client"
	"github.com/waylontech/backup/agent/internal/config"
)

func main() {
	var (
		configPath = flag.String("config", defaultConfigPath(), "config file path")
		doEnroll   = flag.Bool("enroll", false, "enroll this device and exit")
		doBackup   = flag.Bool("backup-now", false, "run one backup cycle and exit")
	)
	flag.Parse()

	log, _ := zap.NewProduction()
	defer log.Sync()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal("load config", zap.Error(err))
	}

	if *doEnroll {
		if err := enrollDevice(cfg, log); err != nil {
			log.Fatal("enroll", zap.Error(err))
		}
		return
	}

	if !cfg.IsEnrolled() {
		log.Fatal("device not enrolled; run with --enroll first")
	}

	c := client.New(cfg.ServerURL, cfg.DeviceToken)

	if err := c.Heartbeat(); err != nil {
		log.Warn("heartbeat failed", zap.Error(err))
	}

	if *doBackup {
		if err := runBackup(context.Background(), cfg, c, log); err != nil {
			log.Fatal("backup", zap.Error(err))
		}
	}
}

func enrollDevice(cfg *config.Config, log *zap.Logger) error {
	if cfg.EnrollToken == "" {
		return fmt.Errorf("enroll_token not set in config")
	}

	c := client.New(cfg.ServerURL, "")

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	resp, err := c.Enroll(client.EnrollRequest{
		Token:        cfg.EnrollToken,
		Hostname:     hostname,
		OSType:       runtime.GOOS,
		Arch:         runtime.GOARCH,
		AgentVersion: cfg.AgentVersion,
	})
	if err != nil {
		return fmt.Errorf("enroll: %w", err)
	}

	cfg.DeviceID = resp.Device.ID
	cfg.DeviceToken = resp.DeviceToken
	cfg.EnrollToken = ""

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	log.Info("enrolled", zap.String("device_id", cfg.DeviceID))
	return nil
}

func runBackup(ctx context.Context, cfg *config.Config, c *client.Client, log *zap.Logger) error {
	engine := backup.New(c, cfg.WatchPath, log)
	result, err := engine.Run(ctx, "")
	if err != nil {
		return err
	}
	log.Info("backup complete",
		zap.String("snapshot_id", result.SnapshotID),
		zap.Int("files", result.FileCount),
		zap.Int64("bytes_scanned", result.BytesScanned),
		zap.Int("chunks_total", result.ChunksTotal),
		zap.Int("chunks_uploaded", result.ChunksUploaded),
		zap.Int("chunks_deduped", result.ChunksDeduped),
	)
	return nil
}

func defaultConfigPath() string {
	if runtime.GOOS == "windows" {
		return `C:\ProgramData\backup-agent\config.yaml`
	}
	return "/etc/backup-agent/config.yaml"
}
