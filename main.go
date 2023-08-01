package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type GitHubPullRequest struct {
	Action     string `json:"action"`
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
	PullRequest struct {
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		HTMLURL string `json:"html_url"`
		Title   string `json:"title"`
	} `json:"pull_request"`
}

var discord = connectToDiscord()

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	log.Println(discord.State.User.Username + " is now running.")

	r := mux.NewRouter()
	r.HandleFunc("/github-webhook", gitHubWebHookHandler).Methods("POST")

	fmt.Println("Starting server on port :8888")
	http.ListenAndServe(":8888", r)
}

func gitHubWebHookHandler(w http.ResponseWriter, r *http.Request) {
	var pr GitHubPullRequest

	if err := json.NewDecoder(r.Body).Decode(&pr); err != nil {
		log.Println("Error decoding JSON: ", err)
		return
	}

	log.Println(pr)

	if pr.Action == "opened" {
		message := fmt.Sprintf("%s: %s", pr.Repository.Name, pr.PullRequest.HTMLURL)
		sendDiscordMessage(pr, message)
	}
}

func sendDiscordMessage(pr GitHubPullRequest, message string) {
	discordChannelId := os.Getenv("DISCORD_CHANNEL_ID")

	_, err := discord.ChannelMessageSend(discordChannelId, message)

	if err != nil {
		log.Println("Error sending message to Discord: ", err)
	}
}

func connectToDiscord() *discordgo.Session {
	discordBotToken := os.Getenv("DISCORD_BOT_TOKEN")

	discord, err := discordgo.New("Bot " + discordBotToken)

	if err != nil {
		log.Println("Error creating Discord session: ", err)
		return nil
	}

	if err := discord.Open(); err != nil {
		log.Println("Error opening connection: ", err)
		return nil
	}

	return discord
}
