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
			case "ping":
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "🏓 Pong! Your STB is awake and listening over Tailscale.",
					},
				})

			case "status":
				// 1. Fetch your hardware metrics (keeping your existing logic)
				vMem, _ := mem.VirtualMemory()
				rootDisk, _ := disk.Usage("/")
				ssdDisk, _ := disk.Usage("/mnt/ssd")

				ramMsg := fmt.Sprintf("🧠 **RAM:** %dMB / %dMB (%.1f%% used)", vMem.Used/1024/1024, vMem.Total/1024/1024, vMem.UsedPercent)
				rootMsg := fmt.Sprintf("💾 **Internal Storage (/)**: %.1fGB / %.1fGB used", float64(rootDisk.Used)/1024/1024/1024, float64(rootDisk.Total)/1024/1024/1024)

				ssdMsg := "💽 **SSD (/mnt/ssd)**: Not found or unmounted"
				if ssdDisk != nil && ssdDisk.Total > 0 {
					ssdMsg = fmt.Sprintf("💽 **SSD (/mnt/ssd)**: %.1fGB / %.1fGB used (%.1f%% free)", float64(ssdDisk.Used)/1024/1024/1024, float64(ssdDisk.Total)/1024/1024/1024, 100-ssdDisk.UsedPercent)
				}

				// 2. Build the structural description string
				descriptionText := fmt.Sprintf("%s\n\n%s\n\n%s", ramMsg, rootMsg, ssdMsg)

				// 3. Respond with a Discord Embed
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{
							{
								Title:       "gip-hm-stb-01 System Status", // Matches the "Note" header in your screenshot
								Description: descriptionText,
								Color:       0xFFAA00, // Hex color code for that Orange/Yellow left border strip
								Footer: &discordgo.MessageEmbedFooter{
									Text: "Last checked: Just now", // Replaces the "Last edited" meta note
								},
							},
						},
					},
				})
				if err != nil {
					log.Printf("Error responding to status command: %v", err)
				}
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
			Name:        "ping",
			Description: "Check if the bot is alive",
		},
		{
			Name:        "status",
			Description: "Fetch a prettified version of the hardware status",
		},
	}
	_, _ = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, guildID, commands)

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}
