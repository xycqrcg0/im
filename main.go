package main

import (
	"im/config"
	"im/routers"
)

func main() {
	config.InitConfig()
	r := routers.SetupRouter()
	port := config.AppConfig.App.Port
	if port == "" {
		port = "8000"
	}
	r.Run(port)
}
