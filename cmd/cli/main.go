package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"tango/internal/config"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var cfg = config.Load()

// ── Token helpers ─────────────────────────────────

func credentialsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "demo", "credentials.json")
}

func saveToken(token string) {
	path := credentialsPath()
	os.MkdirAll(filepath.Dir(path), 0700)
	data, _ := json.Marshal(map[string]string{"access_token": token})
	os.WriteFile(path, data, 0600)
}

func loadToken() string {
	data, err := os.ReadFile(credentialsPath())
	if err != nil {
		return ""
	}
	var creds map[string]string
	json.Unmarshal(data, &creds)
	return creds["access_token"]
}

func authHeader() http.Header {
	h := http.Header{}
	token := loadToken()
	if token != "" {
		h.Set("Authorization", "Bearer "+token)
	}
	return h
}

func doRequest(method, path string, body string) (*http.Response, error) {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}

	req, err := http.NewRequest(method, cfg.BaseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header = authHeader()
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

// ── Root ──────────────────────────────────────────

var rootCmd = &cobra.Command{
	Use:   "demo",
	Short: "TANGO CLI for terminal-first setup and runtime control",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(renderRootScreen(cmd))
	},
}

// ── demo version ──────────────────────────────────

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("demo version 0.1.0")
	},
}

// ── demo status ───────────────────────────────────

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the server status",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := http.Get(cfg.BaseURL + "/api/status")
		if err != nil {
			fmt.Println("❌ Could not connect to the server:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)
		fmt.Printf("✅ Status : %v\n", result["status"])
		fmt.Printf("   Version: %v\n", result["version"])
		fmt.Printf("   Uptime : %v\n", result["uptime"])
	},
}

// ── demo login ────────────────────────────────────

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in and save the access token",
	Run: func(cmd *cobra.Command, args []string) {
		// Read email
		fmt.Print("Email: ")
		var email string
		fmt.Scanln(&email)

		// Read password without echoing
		fmt.Print("Password: ")
		bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Println("\n❌ Failed to read password:", err)
			os.Exit(1)
		}
		password := string(bytePassword)
		fmt.Println()

		// Call the API
		body := fmt.Sprintf(`{"email":"%s","password":"%s"}`, email, password)
		resp, err := http.Post(
			cfg.BaseURL+"/api/auth/login",
			"application/json",
			strings.NewReader(body),
		)
		if err != nil {
			fmt.Println("❌ Could not connect to the server:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		token, ok := result["access_token"].(string)
		if !ok {
			fmt.Println("❌ Login failed:", result["error"])
			os.Exit(1)
		}

		saveToken(token)
		fmt.Println("✅ Logged in successfully!")
		fmt.Printf("   Token saved at: %s\n", credentialsPath())
	},
}

// ── demo logout ───────────────────────────────────

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out",
	Run: func(cmd *cobra.Command, args []string) {
		os.Remove(credentialsPath())
		fmt.Println("✅ Logged out")
	},
}

// ── demo user ─────────────────────────────────────

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
}

var userMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Show the current user",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := doRequest("GET", "/api/user/me", "")
		if err != nil {
			fmt.Println("❌ Error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 401 {
			fmt.Println("❌ Not logged in. Run: demo login")
			os.Exit(1)
		}

		var user map[string]any
		json.NewDecoder(resp.Body).Decode(&user)
		fmt.Printf("ID   : %v\n", user["id"])
		fmt.Printf("Email: %v\n", user["email"])
		fmt.Printf("Name : %v %v\n", user["first_name"], user["last_name"])
	},
}

var userGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a user by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := doRequest("GET", "/api/user/"+args[0], "")
		if err != nil {
			fmt.Println("❌ Error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 401 {
			fmt.Println("❌ Not logged in. Run: demo login")
			os.Exit(1)
		}

		var user map[string]any
		json.NewDecoder(resp.Body).Decode(&user)
		fmt.Printf("ID   : %v\n", user["id"])
		fmt.Printf("Email: %v\n", user["email"])
		fmt.Printf("Name : %v %v\n", user["first_name"], user["last_name"])
	},
}

// ── main ──────────────────────────────────────────

func main() {
	userCmd.AddCommand(userMeCmd, userGetCmd)
	rootCmd.AddCommand(versionCmd, statusCmd, loginCmd, logoutCmd, onboardCmd, discordCmd, userCmd)
	rootCmd.Long = renderRootScreen(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// package main

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"os"

// 	"tango/internal/config"

// 	"github.com/spf13/cobra"
// )

// var cfg = config.Load()

// // ── Root command ─────────────────────────────────
// var rootCmd = &cobra.Command{
// 	Use:   "demo",
// 	Short: "Demo CLI for managing tango from the terminal",
// }

// // ── demo status ──────────────────────────────────
// var statusCmd = &cobra.Command{
// 	Use:   "status",
// 	Short: "Check the server status",
// 	Run: func(cmd *cobra.Command, args []string) {
// 		resp, err := http.Get(cfg.BaseURL + "/api/status")
// 		if err != nil {
// 			fmt.Println("❌ Could not connect to the server:", err)
// 			os.Exit(1)
// 		}
// 		defer resp.Body.Close()

// 		var result map[string]any
// 		json.NewDecoder(resp.Body).Decode(&result)
// 		fmt.Printf("✅ Status : %v\n", result["status"])
// 		fmt.Printf("   Version: %v\n", result["version"])
// 		fmt.Printf("   Uptime : %v\n", result["uptime"])
// 	},
// }

// // ── demo user get <id> ────────────────────────────
// var userCmd = &cobra.Command{
// 	Use:   "user",
// 	Short: "Manage users",
// }

// var userGetCmd = &cobra.Command{
// 	Use:   "get <id>",
// 	Short: "Get a user by ID",
// 	Args:  cobra.ExactArgs(1),
// 	Run: func(cmd *cobra.Command, args []string) {
// 		resp, err := http.Get(cfg.BaseURL + "/api/user/" + args[0])
// 		if err != nil {
// 			fmt.Println("❌ Error:", err)
// 			os.Exit(1)
// 		}
// 		defer resp.Body.Close()

// 		var user map[string]any
// 		json.NewDecoder(resp.Body).Decode(&user)
// 		fmt.Printf("ID   : %v\n", user["id"])
// 		fmt.Printf("Email: %v\n", user["email"])
// 		fmt.Printf("Name : %v\n", user["name"])
// 	},
// }

// // ── demo version ─────────────────────────────────
// var versionCmd = &cobra.Command{
// 	Use:   "version",
// 	Short: "Xem version CLI",
// 	Run: func(cmd *cobra.Command, args []string) {
// 		fmt.Println("demo version 0.1.0")
// 	},
// }

// func main() {
// 	userCmd.AddCommand(userGetCmd)
// 	rootCmd.AddCommand(statusCmd, userCmd, versionCmd)

// 	if err := rootCmd.Execute(); err != nil {
// 		os.Exit(1)
// 	}
// }
