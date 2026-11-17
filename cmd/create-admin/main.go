package main

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"strings"

	"neo360-api/internal/config"
	"neo360-api/internal/domain/account/repository"
	"neo360-api/internal/infra/db"
	"neo360-api/internal/pkg/security"
)

// create-admin cria uma conta da equipe Neo360 (papel admin). Lê os dados de variáveis de ambiente
// para a senha não aparecer na lista de processos nem no histórico do shell.
//
//	ADMIN_EMAIL=... ADMIN_PASSWORD=... ADMIN_NAME=... ./create-admin
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func run() error {
	name := strings.TrimSpace(os.Getenv("ADMIN_NAME"))
	email := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL")))
	password := os.Getenv("ADMIN_PASSWORD")
	if name == "" {
		return errors.New("defina ADMIN_NAME")
	}
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return errors.New("defina ADMIN_EMAIL com um e-mail válido")
	}
	if len(password) < 12 {
		return errors.New("defina ADMIN_PASSWORD com pelo menos 12 caracteres")
	}

	cfg := config.Load()
	ctx := context.Background()
	database, err := db.Open(ctx, cfg.DatabaseURL, cfg.DBSchema)
	if err != nil {
		return err
	}
	defer database.Close() //nolint:errcheck

	hash, err := security.NewBcryptHasher().Hash(password)
	if err != nil {
		return err
	}
	id, err := repository.NewAccountRepository(database).CreateAdmin(ctx, name, email, hash)
	if err != nil {
		return err
	}
	fmt.Printf("admin criado: %s (%s)\n", email, id)
	return nil
}
