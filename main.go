package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

func getDockerContainers() string {
	// Initialize the client pointing to the mounted unix socket
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "❌ Error: Unable to connect to Docker engine."
	}
	defer cli.Close()

	// Fetch ALL containers (even stopped ones) using the correct container types package
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return "❌ Error: Failed to fetch container list."
	}

	if len(containers) == 0 {
		return "ℹ️ No containers found on this system."
	}

	var sb strings.Builder
	for _, c := range containers {
		// Clean up container name (removes the leading slash)
		name := "unknown"
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		// Choose emoji based on state
		statusEmoji := "🔴"
		if c.State == "running" {
			statusEmoji = "🟢"
		} else if c.State == "paused" {
			statusEmoji = "🟡"
		}

		// Append styled line
		sb.WriteString(fmt.Sprintf("%s **%s**\n└─ *Status:* %s\n\n", statusEmoji, name, c.Status))
	}

	return sb.String()
}

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	// guildID := os.Getenv("DISCORD_GUILD_ID")
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
				sdDisk, _ := disk.Usage("/mnt/storage")

				ramMsg := fmt.Sprintf("🧠 **RAM:** %dMB / %dMB (%.1f%% used)", vMem.Used/1024/1024, vMem.Total/1024/1024, vMem.UsedPercent)
				rootMsg := fmt.Sprintf("💾 **Internal Storage (/)**: %.1fGB / %.1fGB used", float64(rootDisk.Used)/1024/1024/1024, float64(rootDisk.Total)/1024/1024/1024)

				ssdMsg := "💽 **SSD (/mnt/ssd)**: Not found or unmounted"
				if ssdDisk != nil && ssdDisk.Total > 0 {
					ssdMsg = fmt.Sprintf("💽 **SSD (/mnt/ssd)**: %.1fGB / %.1fGB used (%.1f%% free)", float64(ssdDisk.Used)/1024/1024/1024, float64(ssdDisk.Total)/1024/1024/1024, 100-ssdDisk.UsedPercent)
				}

				sdMsg := "📟 **SD Card (/mnt/storage)**: Not found or unmounted"
				if sdDisk != nil && sdDisk.Total > 0 {
					sdMsg = fmt.Sprintf("📟 **SD Card (/mnt/storage)**: %.1fGB / %.1fGB used (%.1f%% free)", float64(sdDisk.Used)/1024/1024/1024, float64(sdDisk.Total)/1024/1024/1024, 100-sdDisk.UsedPercent)
				}

				// 2. Build the structural description string
				descriptionText := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", ramMsg, rootMsg, ssdMsg, sdMsg)

				// 3. Respond with a Discord Embed
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{
							{
								Title:       "[gip-hm-stb-01] • System Status",
								Description: descriptionText,
								Color:       0xFFAA00, // Hex color code for that Orange/Yellow left border strip
								Footer: &discordgo.MessageEmbedFooter{
									Text: "Last checked: Just now",
								},
							},
						},
					},
				})
				if err != nil {
					log.Printf("Error responding to status command: %v", err)
				}

			case "services":
				dockerReport := getDockerContainers()

				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{
							{
								Title:       "🖥️ gip-hm-stb-01 • Container Monitor",
								Description: dockerReport,
								Color:       0x00AAFF, // Clean blue color accent
							},
						},
					},
				})
				if err != nil {
					log.Printf("Error responding to services command: %v", err)
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
		{
			Name:        "services",
			Description: "List all running Docker containers and their statuses",
		},
	}
	// _, _ = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, guildID, commands)
	_, _ = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", commands)

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}
