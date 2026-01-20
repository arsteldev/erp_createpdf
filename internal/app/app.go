package app

import (
	grpcapp "github.com/arsteldev/createPDF/internal/app/grpc"
	"log/slog"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(log *slog.Logger, grpcPort int) *App {
	return &App{
		GRPCSrv: grpcapp.New(log, grpcPort),
	}
}
