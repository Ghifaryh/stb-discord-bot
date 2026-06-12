package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	guildID := os.Getenv("DISCORD_GUILD_ID")
	if token == "" {
		log.Fatal("Error: DISCORD_TOKEN environment variable is required")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	// Register slash command handler
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			commandName := i.ApplicationCommandData().Name

			switch commandName {
			case "status":
				// Fetch hardware metrics using gopsutil
				vMem, _ := mem.VirtualMemory()
				// Checking root / for internal space, and your SSD mount point
				rootDisk, _ := disk.Usage("/")
				ssdDisk, _ := disk.Usage("/mnt/ssd")

				ramMsg := fmt.Sprintf("🧠 **RAM:** %dMB / %dMB (%.1f%% used)\n", vMem.Used/1024/1024, vMem.Total/1024/1024, vMem.UsedPercent)
				rootMsg := fmt.Sprintf("💾 **Internal Storage (/)**: %.1fGB / %.1fGB used\n", float64(rootDisk.Used)/1024/1024/1024, float64(rootDisk.Total)/1024/1024/1024)

				ssdMsg := "💽 **SSD (/mnt/ssd)**: Not found or unmounted\n"
				if ssdDisk != nil && ssdDisk.Total > 0 {
					ssdMsg = fmt.Sprintf("💽 **SSD (/mnt/ssd)**: %.1fGB / %.1fGB used (%.1f%% free)\n", float64(ssdDisk.Used)/1024/1024/1024, float64(ssdDisk.Total)/1024/1024/1024, 100-ssdDisk.UsedPercent)
				}

				statusReport := "📊 **STB System Status Report** 📊\n\n" + ramMsg + rootMsg + ssdMsg

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: statusReport,
					},
				})
			case "ping":
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "🏓 Pong! Your STB is awake and listening over Tailscale.",
					},
				})
			}
		}
	})

	err = dg.Open()
	if err != nil {
		log.Fatalf("Error opening connection: %v", err)
	}

	// Register the global slash command
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "status",
			Description: "Fetch current hardware metrics from the STB",
		},
		{
			Name:        "ping",
			Description: "Check if the bot is alive",
		},
	}
	_, _ = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, guildID, commands)

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}
