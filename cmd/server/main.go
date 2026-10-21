package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	_ "time/tzdata"

	"expense_tgbot/internal/ai"
	"expense_tgbot/internal/config"
	"expense_tgbot/internal/service"
	"expense_tgbot/internal/storage/postgres"
	"expense_tgbot/internal/transport/rest"
	"expense_tgbot/internal/transport/telegram"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	pgPool, err := postgres.NewPool(ctx, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pgPool.Close()

	userRepository := postgres.NewUserRepository(pgPool)
	expenseRepository := postgres.NewExpenseRepository(pgPool)

	userService, err := service.NewUserService(
		userRepository,
		cfg.TZName,
	)
	if err != nil {
		log.Fatalf("failed to create user service: %v", err)
	}
	expenseService := service.NewExpenseService(
		expenseRepository,
	)

	aiClient, err := ai.NewClient(
		cfg.LLM.BaseURL,
		cfg.LLM.APIKey,
		cfg.LLM.Model,
	)
	if err != nil {
		log.Fatalf("failed to create AI client: %v", err)
	}

	toolHandler := ai.NewToolHandler(
		expenseService,
		userService,
	)

	agent := ai.NewAgent(aiClient, toolHandler)
	assistant := ai.NewAssistant(agent)

	assistantService := service.NewAssistantService(assistant)

	tgBot, err := telegram.NewBot(
		cfg.Telegram.BotToken,
		assistantService,
		userService,
	)
	if err != nil {
		log.Fatalf("failed to create Telegram bot: %v", err)
	}

	expenseHandler := rest.NewExpenseHandler(expenseService)
	userHandler := rest.NewUserHandler(userService)

	router := rest.NewRouter(rest.Handlers{
		Expenses: expenseHandler,
		Users:    userHandler,
	})

	server := &http.Server{
		Addr: net.JoinHostPort(
			cfg.HTTP.Host,
			strconv.Itoa(cfg.HTTP.Port),
		),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	appCtx, cancel := signal.NotifyContext(
		ctx,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	go telegram.Start(appCtx, tgBot)

	go func() {
		log.Printf("HTTP server started on %s", server.Addr)

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Printf("HTTP server failed: %v", err)
			cancel()
		}
	}()

	waitForShutdown(appCtx, cancel, server)
}

func waitForShutdown(
	ctx context.Context,
	cancel context.CancelFunc,
	server *http.Server,
) {
	<-ctx.Done()

	log.Println("shutting down application...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown failed: %v", err)
		if err := server.Close(); err != nil {
			log.Printf("server close failed: %v", err)
		}
	}

	log.Println("application stopped")
}
