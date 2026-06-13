package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Fail2BanPayload struct {
	IP       string `json:"ip"`
	Jail     string `json:"jail"`
	Failures string `json:"failures"`
	Action   string `json:"action"`
}

// 🌅 Automated Morning 06:00 AM Cron Routine
func startDailyDigest(s *discordgo.Session, channelID string) {
	go func() {
		for {
			now := time.Now()
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location())
			if now.After(nextRun) {
				nextRun = nextRun.Add(24 * time.Hour)
			}

			log.Printf("[Digest] Next scheduled health matrix dispatch window locked: %v", nextRun)
			time.Sleep(time.Until(nextRun))

			// Re-use your resource metrics collector string formatting...
			mockMetricsReport := "🧠 RAM and Storage allocations optimal."

			embed := &discordgo.MessageEmbed{
				Title:       "🌅 gip-hm-stb-01 • Daily Automated Health Digest",
				Description: mockMetricsReport,
				Color:       0x9B59B6,
				Timestamp:   time.Now().Format(time.RFC3339),
			}
			_, _ = s.ChannelMessageSendEmbed(channelID, embed)
		}
	}()
}

// 🌐 Fail2Ban Webhook API Server Listener
func startHTTPServer(s *discordgo.Session, channelID string) {
	http.HandleFunc("/api/security/alert", func(w http.ResponseWriter, r *http.Request) {
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
		if payload.Action == "ban" {
			embed = &discordgo.MessageEmbed{
				Title:       "🚨 SECURITY BREACH ALERT: IP BANNED",
				Description: fmt.Sprintf("### 🔒 Fail2Ban Jail Triggered\n👤 **Offending IP:** `%s`\n⛓️ **Active Jail:** `%s`\n❌ **Failed Attempts:** %s counts", payload.IP, payload.Jail, payload.Failures),
				Color:       0xD32F2F,
				Timestamp:   time.Now().Format(time.RFC3339),
			}
		}

		if embed != nil {
			_, _ = s.ChannelMessageSendEmbed(channelID, embed)
		}
		w.WriteHeader(http.StatusOK)
	})

	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("Failed to spin up local API thread: %v", err)
		}
	}()
}
