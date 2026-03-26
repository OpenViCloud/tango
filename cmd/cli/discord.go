package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type discordRuntimeRequest struct {
	Token                      string   `json:"token"`
	RequireMention             bool     `json:"require_mention"`
	EnableTyping               bool     `json:"enable_typing"`
	EnableMessageContentIntent bool     `json:"enable_message_content_intent"`
	AllowedUserIDs             []string `json:"allowed_user_ids"`
}

type discordRuntimeResponse struct {
	Channel                    string   `json:"channel"`
	Running                    bool     `json:"running"`
	TokenConfigured            bool     `json:"token_configured"`
	RequireMention             bool     `json:"require_mention"`
	EnableTyping               bool     `json:"enable_typing"`
	EnableMessageContentIntent bool     `json:"enable_message_content_intent"`
	AllowedUserIDs             []string `json:"allowed_user_ids"`
}

var (
	discordToken                 string
	discordRequireMention        bool
	discordEnableTyping          bool
	discordEnableMessageContent  bool
	discordAllowedUserIDs        string
)

var discordCmd = &cobra.Command{
	Use:   "discord",
	Short: "Manage the Discord channel runtime on the API server",
}

var discordStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show live Discord runtime state",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := doRequest("GET", "/api/runtime/discord/status", "")
		if err != nil {
			fmt.Println("❌ Error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 401 {
			fmt.Println("❌ Not logged in. Run: demo login")
			os.Exit(1)
		}
		if resp.StatusCode >= 400 {
			printAPIError(resp)
			os.Exit(1)
		}

		var status discordRuntimeResponse
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			fmt.Println("❌ Failed to read response:", err)
			os.Exit(1)
		}

		fmt.Printf("Channel          : %s\n", status.Channel)
		fmt.Printf("Running          : %t\n", status.Running)
		fmt.Printf("Token Configured : %t\n", status.TokenConfigured)
		fmt.Printf("Require Mention  : %t\n", status.RequireMention)
		fmt.Printf("Enable Typing    : %t\n", status.EnableTyping)
		fmt.Printf("Message Content  : %t\n", status.EnableMessageContentIntent)
		fmt.Printf("Allowed Users    : %s\n", strings.Join(status.AllowedUserIDs, ","))
	},
}

var discordStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start or replace the Discord runtime with a new config",
	Run: func(cmd *cobra.Command, args []string) {
		req := discordRuntimeRequest{
			Token:                      strings.TrimSpace(discordToken),
			RequireMention:             discordRequireMention,
			EnableTyping:               discordEnableTyping,
			EnableMessageContentIntent: discordEnableMessageContent,
			AllowedUserIDs:             parseCSV(discordAllowedUserIDs),
		}

		if req.Token == "" {
			fmt.Println("❌ Missing token. Use --token or export DISCORD_BOT_TOKEN.")
			os.Exit(1)
		}

		body, _ := json.Marshal(req)
		resp, err := doRequest("POST", "/api/runtime/discord/start", string(body))
		if err != nil {
			fmt.Println("❌ Error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 401 {
			fmt.Println("❌ Not logged in. Run: demo login")
			os.Exit(1)
		}
		if resp.StatusCode >= 400 {
			printAPIError(resp)
			os.Exit(1)
		}

		var status discordRuntimeResponse
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			fmt.Println("❌ Failed to read response:", err)
			os.Exit(1)
		}

		fmt.Println("✅ Discord runtime started/reloaded.")
		fmt.Printf("   Running: %t\n", status.Running)
	},
}

var discordStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the live Discord runtime",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := doRequest("POST", "/api/runtime/discord/stop", "")
		if err != nil {
			fmt.Println("❌ Error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 401 {
			fmt.Println("❌ Not logged in. Run: demo login")
			os.Exit(1)
		}
		if resp.StatusCode >= 400 {
			printAPIError(resp)
			os.Exit(1)
		}

		fmt.Println("✅ Discord runtime stopped.")
	},
}

var discordRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the live Discord runtime with the current config",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := doRequest("POST", "/api/runtime/discord/restart", "")
		if err != nil {
			fmt.Println("❌ Error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 401 {
			fmt.Println("❌ Not logged in. Run: demo login")
			os.Exit(1)
		}
		if resp.StatusCode >= 400 {
			printAPIError(resp)
			os.Exit(1)
		}

		var status discordRuntimeResponse
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			fmt.Println("❌ Failed to read response:", err)
			os.Exit(1)
		}

		fmt.Println("✅ Discord runtime restarted.")
		fmt.Printf("   Running: %t\n", status.Running)
	},
}

func init() {
	discordStartCmd.Flags().StringVar(&discordToken, "token", os.Getenv("DISCORD_BOT_TOKEN"), "Discord bot token")
	discordStartCmd.Flags().BoolVar(&discordRequireMention, "require-mention", true, "Require bot mention in guild channels")
	discordStartCmd.Flags().BoolVar(&discordEnableTyping, "enable-typing", true, "Send typing indicators while replying")
	discordStartCmd.Flags().BoolVar(&discordEnableMessageContent, "enable-message-content-intent", false, "Enable privileged MESSAGE_CONTENT intent")
	discordStartCmd.Flags().StringVar(&discordAllowedUserIDs, "allowed-user-ids", "", "Comma-separated Discord user IDs allowed to talk to the bot")

	discordCmd.AddCommand(discordStatusCmd, discordStartCmd, discordRestartCmd, discordStopCmd)
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func printAPIError(resp *http.Response) {
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		fmt.Printf("❌ API error: %s\n", resp.Status)
		return
	}
	fmt.Printf("❌ API error: %s\n", strings.TrimSpace(string(body)))
}
