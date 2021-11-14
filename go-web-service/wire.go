//+build wireinject

package main

import (
	"example/go-web-service/internal/biz"
	"example/go-web-service/internal/data"
	"example/go-web-service/internal/service"
	"github.com/google/wire"
)

// InitUserService wire
func InitUserService() *service.UserService {
	wire.Build(service.NewUserService, biz.NewUserBiz, data.NewUserRepo, data.NewDB, data.NewCache)
	return &service.UserService{}
}
