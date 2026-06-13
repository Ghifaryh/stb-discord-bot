package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	probing "github.com/prometheus-community/pro-bing"
)

// List of commands to register to Discord
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
		Name:        "ping-isp",
		Description: "Perform background latency checks against local gateway and backbone fiber",
	},
}

func registerSlashCommands(s *discordgo.Session, guildID string) {
	log.Println("Registering application commands to home target server...")
	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, guildID, commands)
	if err != nil {
		log.Printf("Error syncing application commands: %v", err)
	}
}

func handleSlashCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Change InteractionCreateApplicationCommand to InteractionApplicationCommand
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

	case "ping-isp":
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			log.Printf("Error sending deferred layout: %v", err)
			return
		}

		// Run asynchronous ping logic
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
	}
}

func runPingTest(target string) string {
	pinger, err := probing.NewPinger(target)
	if err != nil {
		return "❌ Configuration Error"
	}

	pinger.SetPrivileged(true) // Combined with cap_add: [NET_ADMIN], handles low-level containers nicely
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
