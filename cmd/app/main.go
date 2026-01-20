package main

import (
	"github.com/arsteldev/createPDF/internal/app"
	"github.com/arsteldev/createPDF/internal/config"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.MustLoad()
	log := setupLogger()

	log.Info("Запускаем приложение",
		slog.Int("port", cfg.GRPC.Port),
		slog.Any("config", cfg),
	)

	application := app.New(log, cfg.GRPC.Port)

	go application.GRPCSrv.MustRun()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	sigh := <-stop

	log.Info("Выключение", slog.String("причина", sigh.String()))

	application.GRPCSrv.Stop()
	log.Info("Приложение остановлено")
}

func setupLogger() *slog.Logger {
	var log *slog.Logger

	log = slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
	)

	return log

}
