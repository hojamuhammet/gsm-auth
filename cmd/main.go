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

	"google.golang.org/grpc" // Import the gRPC package
)

// The server struct holds the necessary components for gRPC server.
type server struct {
	auth.UnimplementedAuthServer // This is a placeholder required by gRPC which contains empty implementations of service methods.
	db                           *database.Database
	loggers                      *logger.Loggers
}

func (s *server) HashAndStore(ctx context.Context, in *auth.PhoneNumber) (*auth.HashedCode, error) {
	// Generate a hash from the phone number
	hash := utils.GenerateHash(in.Number)

	// Store the phone number and the hash in Redis with an expiration time of 3 minutes
	err := s.db.GetDB().Set(ctx, in.Number, hash, 3*time.Minute).Err()
	if err != nil {
		// Log the error if there is one
		s.loggers.ErrorLogger.Error("failed to set key-value pair in Redis: %v", err)
		return nil, err
	}

	// Log the successful operation
	s.loggers.InfoLogger.Info("successfully set key-value pair in Redis: %s-%s", in.Number, hash)

	// Return the hash as a response
	return &auth.HashedCode{Code: hash}, nil
}

func main() {
	// Load the configuration
	cfg := config.LoadConfig()

	// Initialize the database
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close() // Ensure the database connection is closed when main function returns

	// Set up the logger
	loggers, err := logger.SetupLogger("production")
	if err != nil {
		log.Fatalf("failed to set up logger: %v", err)
	}

	// Create a new gRPC server
	s := grpc.NewServer()
	// Register the server with the gRPC server
	auth.RegisterAuthServer(s, &server{db: db, loggers: loggers})

	// Listen for TCP connections on the specified address and port
	address := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Start a goroutine to serve the gRPC server
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Set up a channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Block until an interrupt signal is received
	<-stop
	loggers.InfoLogger.Info("Shutting down the server gracefully...")

	db.Close() // Close the database connection
	os.Exit(0) // Exit the program
}
