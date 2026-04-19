package main

// @title Nlxiaoy Blog API
// @version 1.0
// @description Nlxiaoy Blog 后端 API（Admin + Public V1）。
// @BasePath /api
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @securityDefinitions.apikey AdminSession
// @in cookie
// @name fiber_session

import (
	"log"
	"server/config"
	"server/internal/app"
)

// swag init -g .\cmd\app\main.go -d . -o docs --parseInternal --parseDependency 自动生成路由

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}
	app.Run(cfg)
}
