package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	probing "github.com/prometheus-community/pro-bing"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type ContainerInfo struct {
	Names  []string `json:"Names"`
	State  string   `json:"State"`
	Status string   `json:"Status"`
}

func getDockerContainers() string {
	// Create an HTTP client that communicates over the Unix Domain Socket
	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
	}

	// Query the native Docker Engine API directly for all containers
	resp, err := client.Get("http://localhost/v1.45/containers/json?all=1")
	if err != nil {
		return "❌ Error: Unable to communicate with Docker socket."
	}
	defer resp.Body.Close()

	var containers []ContainerInfo
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return "❌ Error: Failed to parse container metadata."
	}

	if len(containers) == 0 {
		return "ℹ️ No containers found on this system."
	}

	var sb strings.Builder
	for _, c := range containers {
		name := "unknown"
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		statusEmoji := "🔴"
		switch c.State {
		case "running":
			statusEmoji = "🟢"
		case "paused":
			statusEmoji = "🟡"
		}

		sb.WriteString(fmt.Sprintf("%s **%s**\n└─ *Status:* %s\n\n", statusEmoji, name, c.Status))
	}

	return sb.String()
}

func runPingTest(target string) string {
	pinger, err := probing.NewPinger(target)
	if err != nil {
		return "❌ Configuration Error"
	}

	// Crucial for running inside micro-environments/containers without root privileges
	pinger.SetPrivileged(false)
	pinger.Count = 1
	pinger.Timeout = time.Second * 3

	err = pinger.Run() // Blocks until finished
	if err != nil {
		return "🔴 Request Timeout / Unreachable"
	}

	stats := pinger.Statistics()
	if stats.PacketLoss == 100 {
		return "🔴 100% Packet Loss (Offline)"
	}

	// Format output: "Avg: 12.4ms (0% loss)"
	return fmt.Sprintf("⚡ **%v** (📉 %.0f%% loss)", stats.AvgRtt.Round(time.Millisecond), stats.PacketLoss)
}

func startDailyDigest(s *discordgo.Session, channelID string) {
	go func() {
		for {
			// 1. Calculate the exact duration until the next 06:00 AM
			now := time.Now()
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location())

			// If it's already past 6:00 AM today, schedule it for tomorrow morning
			if now.After(nextRun) {
				nextRun = nextRun.Add(24 * time.Hour)
			}

			log.Printf("[Digest] Next automated health report scheduled for: %v", nextRun)
			time.Sleep(time.Until(nextRun))

			// 2. Woke up! Compile hardware metrics (reusing your existing status logic)
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

			// 3. Fetch running container services
			dockerReport := getDockerContainers()

			// 4. Combine reports into a clean morning layout
			embed := &discordgo.MessageEmbed{
				Title:       "🌅 gip-hm-stb-01 • Daily Automated Health Digest",
				Description: fmt.Sprintf("### 📊 System Resources\n%s\n\n%s\n\n%s\n\n%s\n\n### 🐳 Managed Container Services\n%s", ramMsg, rootMsg, ssdMsg, sdMsg, dockerReport),
				Color:       0x9B59B6, // Beautiful sunrise purple accent
				Timestamp:   time.Now().Format(time.RFC3339),
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Automated Routine Health Status Check",
				},
			}

			// 5. Inject the dispatch straight into your administrative channel
			_, err := s.ChannelMessageSendEmbed(channelID, embed)
			if err != nil {
				log.Printf("[Digest Error] Failed to dispatch health digest message: %v", err)
			}
		}
	}()
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

			case "ping-isp":
				// 1. Tell Discord INSTANTLY to display a loading state ("Bot is thinking...")
				// This completely resets the 3-second timeout window to 15 minutes!
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				})
				if err != nil {
					log.Printf("Error sending deferred response: %v", err)
					return
				}

				// 2. Run your metric tests safely without rushing the CPU
				routerPing := runPingTest("192.168.100.1")
				internetPing := runPingTest("1.1.1.1")

				// Format the text string block
				networkReport := fmt.Sprintf("🏠 **Local Gateway (192.168.100.1):** %s\n\n🌐 **Internet Backbone (1.1.1.1):** %s", routerPing, internetPing)

				// 3. Follow up by overwriting the "thinking" status with your final card layout
				_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{
						{
							Title:       "[gip-hm-stb-01] • Network Health Diagnostics",
							Description: networkReport,
							Color:       0x00FF88, // Clean minty green
							Footer: &discordgo.MessageEmbedFooter{
								Text: "Target Network: IndiHome Fiber",
							},
						},
					},
				})
				if err != nil {
					log.Printf("Error editing final interaction response: %v", err)
				}
			}

		}
	})

	err = dg.Open()
	if err != nil {
		log.Fatalf("Error opening connection: %v", err)
	}

	// Fetch a dedicated logging channel ID from your environment variables
	logChannelID := os.Getenv("DISCORD_LOG_CHANNEL_ID")
	if logChannelID != "" {
		startDailyDigest(dg, logChannelID)
		log.Println("🚀 Automated Morning Health Digest subsystem initialized.")
	} else {
		log.Println("⚠️  DISCORD_LOG_CHANNEL_ID not set; skipping digest automation initialization.")
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
		{
			Name:        "ping-isp",
			Description: "Run real-time network latency diagnostics for IndiHome",
		},
	}
	// log.Println("Wiping guild commands...") // check if the build is okay or nah
	// _, _ = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, guildID, commands)
	_, _ = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", commands)

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}
