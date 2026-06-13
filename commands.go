package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	probing "github.com/prometheus-community/pro-bing"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "ping",
		Description: "Check bot processing latency",
	},
	{
		Name:        "status",
		Description: "Get underlying node hardware consumption parameters",
	},
	{
		Name:        "services",
		Description: "Scan host container orchestration engine",
	},
	{
		Name:        "ping-isp",
		Description: "Perform background latency checks against local gateway and backbone fiber",
	},
	{
		Name:        "link",
		Description: "Generate a direct access link to a specified local service port",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "port",
				Description: "The internal port number (e.g., 8080, 9000, 443)",
				Required:    true,
			},
		},
	},
	{
		Name:        "open-wiki",
		Description: "Generate a direct access link to a specified wikipedia article",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "article",
				Description: "The title of the Wikipedia article to open",
				Required:    true,
			},
		},
	},
}

func runPingTest(target string) string {
	pinger, err := probing.NewPinger(target)
	if err != nil {
		return "❌ Configuration Error"
	}

	pinger.SetPrivileged(true)
	pinger.Count = 3
	pinger.Timeout = time.Second * 3

	err = pinger.Run()
	if err != nil {
		return "🔴 Request Timeout / Unreachable"
	}

	stats := pinger.Statistics()
	if stats.PacketLoss == 100 {
		return "🔴 100% Packet Loss (Offline)"
	}

	return fmt.Sprintf("⚡ **%v** (📉 %.0f%% loss)", stats.AvgRtt.Round(time.Millisecond), stats.PacketLoss)
}

func registerSlashCommands(s *discordgo.Session, guildID string) {
	log.Println("🛰️ Syncing application slash commands with Discord home server...")
	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, guildID, commands)
	if err != nil {
		log.Printf("❌ Error syncing application commands: %v", err)
		return
	}
	log.Println("✅ Application slash commands synced successfully!")
}

func handleSlashCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "ping":
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("🏓 Pong! Latency response window: %v", s.HeartbeatLatency().Round(time.Millisecond)),
			},
		})

	case "status":
		// 1. Tell Discord to show a clean, native loading state (No text content!)
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral, // Keeps it private if you want
			},
		})
		if err != nil {
			log.Printf("Error sending deferred status response: %v", err)
			return
		}

		// 2. Fetch live host allocations
		vMem, _ := mem.VirtualMemory()
		rootDisk, _ := disk.Usage("/")
		ssdDisk, _ := disk.Usage("/mnt/ssd")
		sdDisk, _ := disk.Usage("/mnt/storage")

		ramMsg := fmt.Sprintf("🧠 **RAM:** %dMB / %dMB (%.1f%% used)", vMem.Used/1024/1024, vMem.Total/1024/1024, vMem.UsedPercent)
		rootMsg := fmt.Sprintf("💾 **Internal Storage (/)**: %.1fGB / %.1fGB used", float64(rootDisk.Used)/1024/1024/1024, float64(rootDisk.Total)/1024/1024/1024)

		ssdMsg := "💽 **SSD (/mnt/ssd)**: Unmounted"
		if ssdDisk != nil && ssdDisk.Total > 0 {
			ssdMsg = fmt.Sprintf("💽 **SSD (/mnt/ssd)**: %.1fGB / %.1fGB used (%.1f%% free)", float64(ssdDisk.Used)/1024/1024/1024, float64(ssdDisk.Total)/1024/1024/1024, 100-ssdDisk.UsedPercent)
		}

		sdMsg := "📟 **SD Card (/mnt/storage)**: Unmounted"
		if sdDisk != nil && sdDisk.Total > 0 {
			sdMsg = fmt.Sprintf("📟 **SD Card (/mnt/storage)**: %.1fGB / %.1fGB used (%.1f%% free)", float64(sdDisk.Used)/1024/1024/1024, float64(sdDisk.Total)/1024/1024/1024, 100-sdDisk.UsedPercent)
		}

		// 3. Edit the deferred response directly with your clean embed card
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				{
					Title:       "📊 gip-hm-stb-01 • Core Resource Allocations",
					Description: fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", ramMsg, rootMsg, ssdMsg, sdMsg),
					Color:       0x3498DB,
					Timestamp:   time.Now().Format(time.RFC3339),
				},
			},
		})
		if err != nil {
			log.Printf("Error editing status interaction response: %v", err)
		}

	case "services":
		// 1. Instantly respond with a native Discord loading state
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			log.Printf("Error sending deferred services response: %v", err)
			return
		}

		// 2. Query the native Docker Engine API directly over the Unix socket
		dockerReport := getDockerContainers()

		// 3. Edit the response with your crisp emerald green embed matrix card
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				{
					Title:       "🐳 Managed Container Engine Orchestration",
					Description: dockerReport,
					Color:       0x2ECC71,
					Timestamp:   time.Now().Format(time.RFC3339),
				},
			},
		})
		if err != nil {
			log.Printf("Error editing services interaction response: %v", err)
		}

	case "ping-isp":
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			log.Printf("Error sending deferred layout: %v", err)
			return
		}

		routerPing := runPingTest("192.168.100.1")
		internetPing := runPingTest("1.1.1.1")

		networkReport := fmt.Sprintf("🏠 **Local Gateway (192.168.100.1):** %s\n\n🌐 **Internet Backbone (1.1.1.1):** %s", routerPing, internetPing)

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				{
					Title:       "[gip-hm-stb-01] • Network Health Diagnostics",
					Description: networkReport,
					Color:       0x00FF88,
					Footer: &discordgo.MessageEmbedFooter{
						Text: "Target Network: IndiHome Fiber",
					},
				},
			},
		})

	case "link":
		options := i.ApplicationCommandData().Options
		targetPort := options[0].StringValue()

		// 2. Build your clean development portal link
		// Adjust the base domain/IP here to match your exact layout preference
		generatedURL := fmt.Sprintf("http://localhost:%s", targetPort)

		// 3. Respond with a clean message (enclosing the URL in <> stops Discord from creating an ugly blank embed preview)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("🔗 **STB Endpoint Portal Wrapper:**\n🌐 Direct Access Link: <%s>", generatedURL),
			},
		})

	case "open-wiki":
		options := i.ApplicationCommandData().Options
		theCode := options[0].StringValue()

		// 1. Format the URL cleanly (Note: Wikipedia articles use /wiki/Title format)
		generatedURL := fmt.Sprintf("https://en.wikipedia.org/wiki/%s", theCode)

		// 2. Respond without the angle brackets so Discord scrapes the link preview
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("🔗 **Instant Wiki Portal Wrapper:**\n🌐 Direct Access Link: %s", generatedURL),
			},
		})
	}
}
