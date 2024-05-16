package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	auth "gsm-auth/gen"
	"gsm-auth/internal/config"
	"gsm-auth/pkg/database"
	"gsm-auth/pkg/lib/logger"
	"gsm-auth/pkg/lib/utils"

	"google.golang.org/grpc"
)

type server struct {
	auth.UnimplementedAuthServer
	db      *database.Database
	loggers *logger.Loggers
}

func (s *server) HashAndStore(ctx context.Context, in *auth.PhoneNumber) (*auth.HashedCode, error) {
	hash := utils.GenerateHash(in.Number)

	err := s.db.GetDB().Set(ctx, in.Number, hash, 3*time.Minute).Err()
	if err != nil {
		s.loggers.ErrorLogger.Error("failed to set key-value pair in Redis: %v", err)
		return nil, err
	}

	s.loggers.InfoLogger.Info("successfully set key-value pair in Redis: %s-%s", in.Number, hash)

	return &auth.HashedCode{Code: hash}, nil
}

func main() {
	cfg := config.LoadConfig()

	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	loggers, err := logger.SetupLogger("production")
	if err != nil {
		log.Fatalf("failed to set up logger: %v", err)
	}

	s := grpc.NewServer()
	auth.RegisterAuthServer(s, &server{db: db, loggers: loggers})

	address := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	loggers.InfoLogger.Info("Shutting down the server gracefully...")

	db.Close()
	os.Exit(0)
}
