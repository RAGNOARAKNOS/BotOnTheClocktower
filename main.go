package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var DiscordBotToken string

func checkNilError(e error) {
	if e != nil {
		log.Fatal("Error message! Something has gone wrong...")
	}
}

func newMessage(discord *discordgo.Session, message *discordgo.MessageCreate) {
	// prevent the bot from reacting to its own messages
	if message.Author.ID == discord.State.User.ID {
		fmt.Println("dont talk to myself")
		return
	}

	// Deconstruct any new messages into paramaterised strings
	msgContents := strings.Fields(message.Content)

	for i := 0; i < len(msgContents); i++ {
		fmt.Printf("MessageParam# %d MessageParamContent %q \n", i, msgContents[i])
	}

	// If the first param is a recognised command process it, otherwise return
	if strings.Contains(msgContents[0], "!botc") {
		fmt.Println("calculate a response")
		discord.ChannelMessageSend(message.ChannelID, "BOTC is alive!")
	}
}

func main() {
	// Load configurable parameters via GOENV lib from .env file
	envErr := godotenv.Load()
	checkNilError(envErr)

	// create a session with the discord API
	DiscordBotToken := os.Getenv("BOTAPIKEY")
	discord, discordErr := discordgo.New("Bot " + DiscordBotToken)
	checkNilError(discordErr)

	// register the funtion to process new messages
	discord.AddHandler(newMessage)

	// open the session
	discord.Open()
	defer discord.Close() // close session when function terminates

	// await the ctl+c interrupt to exit the program (indefinite loop)
	fmt.Println("Bot running...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

}
