package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	// 1. Load configuration attributes
	token := os.Getenv("DISCORD_TOKEN")
	guildID := os.Getenv("DISCORD_GUILD_ID")
	logChannelID := os.Getenv("DISCORD_LOG_CHANNEL_ID")

	if token == "" {
		log.Fatal("Error: DISCORD_TOKEN environment variable is required")
	}

	// 2. Initialize Discord Session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	// Register slash command route listeners (Declared in commands.go)
	dg.AddHandler(handleSlashCommands)

	err = dg.Open()
	if err != nil {
		log.Fatalf("Error opening connection: %v", err)
	}
	defer dg.Close()

	// 3. Register Slash Commands to Target Server (Declared in commands.go)
	registerSlashCommands(dg, guildID)

	startHTTPServer(dg, logChannelID)
	log.Println("🌐 Internal Security Webhook listener deployed on port :8080.")

	// 4. Initialize Background Service Engines (Declared in services.go)
	if logChannelID != "" {
		startDailyDigest(dg, logChannelID)
		log.Println("🚀 Automated Morning Health Digest subsystem initialized.")
	} else {
		log.Println("⚠️  DISCORD_LOG_CHANNEL_ID not set; 06:00 AM morning digest suspended.")
	}

	// 5. Keep system process alive cleanly
	log.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
