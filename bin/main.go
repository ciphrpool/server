package main

import (
	"backend/lib/server"
	"fmt"
	"os"
	"strconv"

	_ "github.com/joho/godotenv/autoload"
)

func main() {

	server, err := server.New()
	if err != nil {
		panic(fmt.Sprintf("cannot start server: %s", err))
	}

	server.Start()

	port, _ := strconv.Atoi(os.Getenv("PORT"))
	err = server.Listen(fmt.Sprintf(":%d", port))
	if err != nil {
		panic(fmt.Sprintf("cannot start server: %s", err))
	}
}
