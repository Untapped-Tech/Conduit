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

	authPkg "github.com/untappedtech/conduit/internal/auth"
	"github.com/untappedtech/conduit/internal/config"
	"github.com/untappedtech/conduit/internal/db"
	httpPkg "github.com/untappedtech/conduit/internal/http"
	"github.com/untappedtech/conduit/internal/service"
)

func main() {
	configFilePath := flag.String("config", "", "Path to configuration file")
	flag.StringVar(configFilePath, "c", "", "Path to configuration file (short)")

	generateConfigFormat := flag.String("generate-config", "", "Generate default config file of specified type (json|yaml|toml|xml) and exit")
	flag.StringVar(generateConfigFormat, "g", "", "Short for --generate-config")
	flag.Parse()

	if *generateConfigFormat != "" {
		targetOutputFileName := fmt.Sprintf("config.%s", *generateConfigFormat)
		if generateError := config.GenerateDefaultConfig(*generateConfigFormat, targetOutputFileName); generateError != nil {
			log.Fatalf("failed to generate config: %v", generateError)
		}
		log.Printf("Successfully generated default configuration file: %s", targetOutputFileName)
		os.Exit(0)
	}

	serverConfig, loadError := config.Load(*configFilePath)
	if loadError != nil {
		log.Fatalf("failed to load configuration: %v", loadError)
	}

	authChain, authChainError := authPkg.BuildAuthChain(serverConfig)
	if authChainError != nil {
		log.Fatalf("failed to initialize auth chain: %v", authChainError)
	}
	defer func() {
		for _, provider := range authChain {
			_ = provider.Close()
		}
	}()

	tokenExtractor := authPkg.NewDefaultTokenExtractor()
	responseEncoder := httpPkg.NewResponseEncoder()

	operationalDB, dbError := db.NewDatabase(serverConfig)
	if dbError != nil {
		log.Fatalf("failed to initialize operational database [%s]: %v", serverConfig.Database.Driver, dbError)
	}

	apiService := service.NewAPIService(operationalDB, serverConfig)
	serverInstance := httpPkg.NewServer(apiService, serverConfig, responseEncoder)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		log.Printf("Server listening on %s:%d (Driver: %s, Auth Chain Depth: %d)", 
			serverConfig.Server.Host, serverConfig.Server.Port, serverConfig.Database.Driver, len(authChain))
		if listenError := serverInstance.ListenAndServe(authChain, tokenExtractor, responseEncoder); listenError != nil && !errors.Is(listenError, http.ErrServerClosed) {
			log.Fatalf("Server listener failure: %v", listenError)
		}
	}()

	<-signalChannel
	log.Println("[INFO] Interrupt signal caught. Commencing graceful teardown...")

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if shutdownError := serverInstance.Shutdown(shutdownContext); shutdownError != nil {
		log.Printf("[ERROR] HTTP server shutdown error: %v", shutdownError)
	}

	if dbCloseError := operationalDB.Close(); dbCloseError != nil {
		log.Printf("[ERROR] Database closure error: %v", dbCloseError)
	}

	log.Println("[INFO] Graceful teardown complete. All persistent database pages flushed to disk.")
}
