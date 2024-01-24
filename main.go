package main

import (
	"fmt"
	"os"

	"github.com/Logan9312/Ark-Whitelist-Bot/routers"
	"github.com/joho/godotenv"

	"github.com/Logan9312/Ark-Whitelist-Bot/bot"
)

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file:", err)
	}

	//Connects main bot
	_, err = bot.BotConnect(os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		fmt.Println(err)
	}

	//go commands.SetRoles(mainSession)
	fmt.Println("Bot is running!")

	routers.HealthCheck()

}