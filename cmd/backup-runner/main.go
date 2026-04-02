package main

import (
	"context"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tango/internal/config"
	runnerhttp "tango/internal/runner/http"
	runnerservice "tango/internal/runner/service"
	runnertools "tango/internal/runner/tools"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	logger.Info("backup runner config loaded",
		"port", cfg.BackupRunnerPort,
		"postgresInstallDir", cfg.PostgresInstallDir,
		"mysqlInstallDir", cfg.MySQLInstallDir,
		"mariadbInstallDir", cfg.MariaDBInstallDir,
		"mongoToolsDir", cfg.MongoToolsDir,
	)
	if err := runnertools.VerifyPostgresInstallation(cfg.PostgresInstallDir); err != nil {
		logger.Warn("backup runner postgres client tools verification failed", "installDir", cfg.PostgresInstallDir, "err", err)
	} else {
		logger.Info("backup runner postgres client tools verified", "installDir", cfg.PostgresInstallDir)
	}
	if err := runnertools.VerifyMySQLInstallation(cfg.MySQLInstallDir); err != nil {
		logger.Warn("backup runner mysql client tools verification failed", "installDir", cfg.MySQLInstallDir, "err", err)
	} else {
		logger.Info("backup runner mysql client tools verified", "installDir", cfg.MySQLInstallDir)
	}
	if err := runnertools.VerifyMariaDBInstallation(cfg.MariaDBInstallDir); err != nil {
		logger.Warn("backup runner mariadb client tools verification failed", "installDir", cfg.MariaDBInstallDir, "err", err)
	} else {
		logger.Info("backup runner mariadb client tools verified", "installDir", cfg.MariaDBInstallDir)
	}
	if err := runnertools.VerifyMongoInstallation(cfg.MongoToolsDir); err != nil {
		logger.Warn("backup runner mongo tools verification failed", "toolsDir", cfg.MongoToolsDir, "err", err)
	} else {
		logger.Info("backup runner mongo tools verified", "toolsDir", cfg.MongoToolsDir)
	}

	postgresRunner := runnerservice.NewPostgresRunner(cfg.PostgresInstallDir)
	mysqlRunner := runnerservice.NewMySQLRunner(cfg.MySQLInstallDir)
	mariadbRunner := runnerservice.NewMariaDBRunner(cfg.MariaDBInstallDir)
	mongoRunner := runnerservice.NewMongoRunner(cfg.MongoToolsDir)
	handler := runnerhttp.NewHandler(cfg.BackupRunnerToken, mysqlRunner, mariadbRunner, postgresRunner, mongoRunner)
	server := &nethttp.Server{
		Addr:              ":" + cfg.BackupRunnerPort,
		Handler:           runnerhttp.NewRouter(handler),
		ReadHeaderTimeout: 15 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	logger.Info("backup runner listening", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
		logger.Error("backup runner stopped", "err", err)
		os.Exit(1)
	}
	logger.Info("backup runner stopped", "reason", fmt.Sprintf("%v", ctx.Err()))
}
