package main

import "github.com/railwayapp/railway-go-sdk"

// This repository manages only its own resources in the environment. Other
// repositories export their own partial name.
// See https://docs.railway.com/infrastructure-as-code#multi-repo-projects
const Partial = "neo-api"

// Railway descreve o serviço neo-api. As variáveis ficam no painel do Railway:
// o arquivo só as preserva pelo nome, sem valores.
func Railway(ctx railway.Context) railway.Project {
	ctx = railway.NewContext(ctx)
	neoApi := railway.ServiceNamed("neo-api", railway.ServiceConfig{
		"source":             railway.Github("andrelbraga/neo-api"),
		"build":              map[string]any{"buildEnvironment": "V3", "builder": "DOCKERFILE", "dockerfilePath": "Dockerfile"},
		"healthcheck":        "/api/health",
		"healthcheckTimeout": 60,
		"replicas":           map[string]any{"iad": 1},
		"variables": map[string]any{
			"ALLOWED_ORIGINS": railway.Preserve(),
			"APP_ENV":         railway.Preserve(),
			"DATABASE_URL":    railway.Preserve(),
			"DB_SCHEMA":       railway.Preserve(),
			"JWT_SECRET":      railway.Preserve(),
		},
	})
	return railway.ProjectNamed("project", []any{neoApi})
}
