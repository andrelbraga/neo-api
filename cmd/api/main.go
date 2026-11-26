package main

import (
	"context"
	"os"

	"neo360-api/internal/config"
	accountRoute "neo360-api/internal/domain/account/route"
	brokerRepository "neo360-api/internal/domain/broker/repository"
	brokerRoute "neo360-api/internal/domain/broker/route"
	brokerService "neo360-api/internal/domain/broker/service"
	developmentRepository "neo360-api/internal/domain/development/repository"
	developmentRoute "neo360-api/internal/domain/development/route"
	developmentService "neo360-api/internal/domain/development/service"
	leadRoute "neo360-api/internal/domain/lead/route"
	"neo360-api/internal/domain/media"
	mediaRoute "neo360-api/internal/domain/media/route"
	organizationRoute "neo360-api/internal/domain/organization/route"
	"neo360-api/internal/infra/api"
	"neo360-api/internal/infra/db"
	"neo360-api/internal/infra/storage"
	"neo360-api/internal/pkg/logger"
	"neo360-api/internal/pkg/security"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.IsDeployed())

	if err := cfg.Validate(); err != nil {
		log.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	database, err := db.Open(context.Background(), cfg.DatabaseURL, cfg.DBSchema)
	if err != nil {
		log.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer database.Close() //nolint:errcheck

	// Bucket das fotos (ADR-0029). Sem S3 configurado, a API sobe e só o envio de fotos fica desligado.
	var bucket media.Bucket
	b, err := storage.New(cfg.Storage)
	switch {
	case err != nil:
		log.Error("storage unavailable", "error", err)
		os.Exit(1)
	case b != nil:
		bucket = b
	default:
		log.Info("storage not configured: photo upload disabled")
	}

	app := api.New(cfg, database, log)
	app.Setup()
	app.HealthCheck()

	brokers := brokerService.NewBrokerService(brokerRepository.NewBrokerRepository(database), security.NewBcryptHasher())
	developments := developmentService.NewDevelopmentService(developmentRepository.NewDevelopmentRepository(database))

	accountRoute.Register(app, brokers)
	brokerRoute.Register(app, brokers)
	organizationRoute.Register(app)
	developmentRoute.Register(app, developments)
	leadRoute.Register(app, developments)
	mediaRoute.Register(app, bucket)

	app.Server()
}
