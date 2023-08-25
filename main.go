package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

type GitHubPullRequest struct {
	Action     string `json:"action"`
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
	PullRequest struct {
		Id   int `json:"id"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		HTMLURL string `json:"html_url"`
		Title   string `json:"title"`
	} `json:"pull_request"`
}

var discord *discordgo.Session

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	discord = connectToDiscord()

	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: os.Getenv("REDIS_PASS"),
		DB:       0,
	})

	ctx := context.Background()
	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		log.Println("Error connecting to Redis: ", err)
		return
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
		value := sendDiscordMessage(pr, message)

		redisKey := strconv.Itoa(pr.PullRequest.Id)
		redisValue := value.ID

		redisClient.Set(r.Context(), redisKey, redisValue, 0)
	} else if pr.Action == "closed" {
		discordMessageId := redisClient.Get(r.Context(), strconv.Itoa(pr.PullRequest.Id)).Val()
		discord.ChannelMessageDelete(os.Getenv("DISCORD_CHANNEL_ID"), discordMessageId)
		redisClient.Del(r.Context(), strconv.Itoa(pr.PullRequest.Id))
	}
}

func sendDiscordMessage(pr GitHubPullRequest, message string) *discordgo.Message {
	discordChannelId := os.Getenv("DISCORD_CHANNEL_ID")

	value, err := discord.ChannelMessageSend(discordChannelId, message)

	if err != nil {
		log.Println("Error sending message to Discord: ", err)
	}

	return value
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
