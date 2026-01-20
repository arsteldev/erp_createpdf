package grpc

import (
	"fmt"
	"github.com/arsteldev/createPDF/internal/services/createPDF"
	createpdffile "github.com/arsteldev/createPDF/proto"
	"google.golang.org/grpc"
	"log/slog"
	"net"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	PDFServer  *createPDF.PDFServer
	port       int
}

func New(log *slog.Logger, port int) *App {
	return &App{
		log: log,
		gRPCServer: grpc.NewServer(
			grpc.MaxSendMsgSize(50*1024*1024),
			grpc.MaxRecvMsgSize(50*1024*1024)),
		PDFServer: &createPDF.PDFServer{
			Log: log,
		},
		port: port,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "grpcapp.Run"

	log := a.log.With(slog.String("op", op), slog.Int("port", a.port))

	createpdffile.RegisterPDFCreatorServer(a.gRPCServer, a.PDFServer)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("gRPC сервер запускается", slog.String("address", l.Addr().String()))

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *App) Stop() {
	const op = "grpcapp.Stop"
	a.log.With(slog.String("op", op)).Info("останавливается gRPC сервер", slog.Int("port", a.port))

	a.gRPCServer.GracefulStop()
}
