package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ContainerInfo maps the raw structural array schema returned by the Docker Engine API
type ContainerInfo struct {
	ID     string   `json:"Id"`
	Names  []string `json:"Names"`
	State  string   `json:"State"`
	Status string   `json:"Status"`
}

type Fail2BanPayload struct {
	IP       string `json:"ip"`
	Jail     string `json:"jail"`
	Failures string `json:"failures"`
	Action   string `json:"action"`
}

// 🌅 Automated Morning 06:00 AM Cron Routine
func startDailyDigest(s *discordgo.Session, channelID string) {
	go func() {
		loc, err := time.LoadLocation("Asia/Jakarta")
		if err != nil {
			log.Printf("[Digest Error] Failed to load local timezone: %v, defaulting to system clock", err)
			loc = time.Local
		}
		for {
			now := time.Now().In(loc)
			// Force the target date parameters to match your local timezone frame
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, loc)
			if now.After(nextRun) {
				nextRun = nextRun.Add(24 * time.Hour)
			}

			log.Printf("[Digest] Next automated morning health report locked for: %v", nextRun)
			time.Sleep(time.Until(nextRun))

			// Your daily digest now gracefully re-uses your true raw socket analyzer!
			dockerReport := getDockerContainers()

			embed := &discordgo.MessageEmbed{
				Title:       "🌅 gip-hm-stb-01 • Daily Automated Health Digest",
				Description: fmt.Sprintf("### 🐳 Managed Container Services\n%s", dockerReport),
				Color:       0x9B59B6,
				Timestamp:   time.Now().Format(time.RFC3339),
			}
			_, _ = s.ChannelMessageSendEmbed(channelID, embed)
		}
	}()
}

// 🌐 Fail2Ban Webhook API Server Listener
func startHTTPServer(s *discordgo.Session, channelID string) {
	// Create a private multiplexer instead of using the global http namespace
	mux := http.NewServeMux()

	mux.HandleFunc("/api/security/alert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload Fail2BanPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Bad payload parameters", http.StatusBadRequest)
			return
		}

		var embed *discordgo.MessageEmbed
		switch payload.Action {
		case "ban":
			embed = &discordgo.MessageEmbed{
				Title:       "🚨 SECURITY BREACH ALERT: IP BANNED",
				Description: fmt.Sprintf("### 🔒 Fail2Ban Jail Triggered\n👤 **Offending IP:** `%s`\n⛓️ **Active Jail:** `%s`\n❌ **Failed Attempts:** %s counts", payload.IP, payload.Jail, payload.Failures),
				Color:       0xD32F2F,
				Timestamp:   time.Now().Format(time.RFC3339),
			}
		case "unban":
			embed = &discordgo.MessageEmbed{
				Title:       "🔓 SECURITY INFO: BAN EXPIRED",
				Description: fmt.Sprintf("### 🕒 Cooldown Period Finished\n👤 **Released IP:** `%s`\n⛓️ **Jail Context:** `%s`", payload.IP, payload.Jail),
				Color:       0xF5B041,
				Timestamp:   time.Now().Format(time.RFC3339),
			}
		}

		if embed != nil {
			_, err := s.ChannelMessageSendEmbed(channelID, embed)
			if err != nil {
				log.Printf("[Security Alert Error] Failed to dispatch embed: %v", err)
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	})

	// Spin up the listener server inside its own thread using our custom mux
	go func() {
		server := &http.Server{
			Addr:    ":8080",
			Handler: mux,
		}
		log.Println("🌐 Isolate HTTP API Server listening on port :8080...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Internal web listener error: %v", err)
		}
	}()
}

// 🐋 Your Original Raw Unix Domain Socket Implementation Restored!
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
