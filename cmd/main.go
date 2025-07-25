package main

import (
	"github.com/flyflow-devs/flyflow/internal/config"
	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/flyflow-devs/flyflow/internal/server"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var rootCmd = &cobra.Command{
	Use:   "voice-api",
	Short: "Voice API",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.NewConfig()
		if err != nil {
			log.Printf("error loading config: %v", err)
		}
		db := server.InitDB(cfg, false)
		logger.InitLogger(cfg.Env)

		s := server.NewServer(cfg, db)
		logger.S.Info("Serving on port " + cfg.Port)

		go func() {
			if err := http.ListenAndServe(":"+cfg.Port, s.Router); err != nil && err != http.ErrServerClosed {
				log.Fatalf("ListenAndServe(): %v", err)
			}
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit

		// Wait for WebSocket connections to close
		s.WG.Wait()
	},
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database operations",
}

var autoMigrateCmd = &cobra.Command{
	Use:   "automigrate",
	Short: "Automatically migrate the database schema",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.NewConfig()
		if err != nil {
			log.Printf("error loading config: %v", err)
		}

		server.InitDB(cfg, true)
		logger.S.Info("Database migration completed successfully.")

		r := mux.NewRouter()

		// Add a health check endpoint
		r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		logger.S.Info("Serving on port " + cfg.Port)
		logger.S.Fatal(http.ListenAndServe(":"+cfg.Port, r))
	},
}

func init() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Printf("error loading config: %v", err)
	}
	logger.InitLogger(cfg.Env)

	dbCmd.AddCommand(autoMigrateCmd)
	rootCmd.AddCommand(dbCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
