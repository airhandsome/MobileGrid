package main

import (
	"github.com/gridsystem-back/internal/handler"
	"github.com/gridsystem-back/pkg/util"
	"github.com/spf13/viper"

	"github.com/gin-gonic/gin"
)

func main() {
	util.InitConfig()

	router := gin.Default()
	handler.RegisterRoutes(router)

	port := viper.GetString("server.port")
	if err := router.Run(":" + port); err != nil {
		panic(err)
	}
}
