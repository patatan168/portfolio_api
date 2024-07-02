package main

import (
	"flag"
	"log"
	"os"
	"server/api"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	// 環境変数取得
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// データベースの設定
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	// Ex)127.0.0.1
	host := os.Getenv("HOST")

	// データベースのURI
	uri := "user=" + dbUser + " password=" + dbPassword + " host=" + host + " port=" + dbPort + " dbname=" + dbName
	app := fiber.New(fiber.Config{
		ProxyHeader:        fiber.HeaderXForwardedFor,
		EnableIPValidation: true, //Trade off performance
	})
	// flag.Parse()がないと値がデフォルト値になる
	flag.Parse()

	// Corsの設定
	frontPort := os.Getenv("FRONT_PORT")
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://127.0.0.1:" + frontPort + ", http://localhost:" + frontPort,
		AllowCredentials: true,
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowHeaders:     "Access-Control-Allow-Origin, Content-Type",
	}))

	// 圧縮の設定
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Todoコネクション
	api.ConnTodo(app, uri)
	// Userコネクション
	api.ConnUser(app, uri)

	app.Listen(":4000")
}
