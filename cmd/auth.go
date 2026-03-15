package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/auth"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authSwitchCmd)
	authCmd.AddCommand(authSessionCmd)
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with a Plane instance",
	Long:  "Log in to a Plane instance. Stores credentials securely in the OS keyring.",
	Example: `  plane auth login
  plane auth login --api-url https://plane.example.com`,
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		cfg, err := auth.LoadConfig()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		profile := cfg.ActiveProfile
		if envProfile := os.Getenv(auth.EnvProfile); envProfile != "" {
			profile = envProfile
		}

		// Get API URL
		apiURL := flagAPIURL
		if apiURL == "" {
			existing := cfg.ActiveProfileConfig().APIURL
			prompt := "Plane API URL"
			if existing != "" {
				prompt += fmt.Sprintf(" [%s]", existing)
			}
			fmt.Fprintf(os.Stderr, "%s: ", prompt)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input == "" {
				input = existing
			}
			apiURL = input
		}
		if apiURL == "" {
			return fmt.Errorf("API URL is required")
		}
		apiURL = strings.TrimRight(apiURL, "/")

		// Get workspace
		workspace := flagWorkspace
		if workspace == "" {
			existing := cfg.ActiveProfileConfig().Workspace
			prompt := "Workspace slug"
			if existing != "" {
				prompt += fmt.Sprintf(" [%s]", existing)
			}
			fmt.Fprintf(os.Stderr, "%s: ", prompt)
			input, _ := reader.ReadString('\n')
			workspace = strings.TrimSpace(input)
			if workspace == "" {
				workspace = existing
			}
		}

		// Get API key
		apiKey := flagAPIKey
		if apiKey == "" {
			fmt.Fprint(os.Stderr, "API key: ")
			tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			if err != nil {
				return fmt.Errorf("reading API key: %w", err)
			}
			apiKey = string(tokenBytes)
		}
		apiKey = strings.TrimSpace(apiKey)

		if err := auth.ValidateTokenFormat(apiKey); err != nil {
			return fmt.Errorf("invalid API key: %w", err)
		}

		// Validate by calling /api/v1/users/me/
		client := api.NewClient(apiURL, apiKey, workspace, flagVerbose, os.Stderr)
		users := api.NewUsersService(client)
		user, err := users.Me(context.Background())
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		// Store in keyring
		store, err := auth.NewKeyringStore("")
		if err != nil {
			return fmt.Errorf("opening keyring: %w", err)
		}

		if err := store.Set(profile+"/api-key", apiKey); err != nil {
			return fmt.Errorf("storing credential: %w", err)
		}

		// Save config
		cfg.ActiveProfile = profile
		cfg.SetProfile(profile, auth.Profile{
			APIURL:    apiURL,
			Workspace: workspace,
		})
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		if !flagQuiet {
			fmt.Fprintf(os.Stderr, "Authenticated as %s (%s)\n", user.DisplayName, user.Email)
			fmt.Fprintf(os.Stderr, "Profile: %s\n", profile)
			fmt.Fprintf(os.Stderr, "Credentials stored in OS keyring.\n")
		}
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := auth.LoadConfig()
		if err != nil {
			return err
		}

		store, err := auth.NewKeyringStore("")
		if err != nil {
			return fmt.Errorf("opening keyring: %w", err)
		}

		profile := cfg.ActiveProfile
		_ = store.Delete(profile + "/api-key")
		_ = store.Delete(profile + "/session-token")

		if !flagQuiet {
			fmt.Fprintf(os.Stderr, "Credentials removed for profile %q.\n", profile)
		}
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := auth.LoadConfig()
		if err != nil {
			return err
		}

		resolver := &auth.Resolver{
			FlagToken: flagAPIKey,
			Env:       &auth.EnvSource{Quiet: flagQuiet},
			Config:    cfg,
		}

		store, err := auth.NewKeyringStore("")
		if err == nil {
			resolver.Store = store
		}

		resolved, err := resolver.Resolve()
		if err != nil {
			if !flagQuiet {
				fmt.Fprintf(os.Stderr, "Not authenticated: %v\n", err)
			}
			return nil
		}
		defer resolved.Credential.Clear()

		profileCfg := cfg.ActiveProfileConfig()

		status := map[string]string{
			"profile":   cfg.ActiveProfile,
			"source":    resolved.Source,
			"api_url":   profileCfg.APIURL,
			"workspace": profileCfg.Workspace,
			"token":     resolved.Credential.Masked(),
		}

		// Check for session cookie
		if store != nil {
			_, err := store.Get(cfg.ActiveProfile + "/session-token")
			if err == nil {
				status["session"] = "stored"
			}
		}

		return Formatter().Format(os.Stdout, status)
	},
}

var authSessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Set a session cookie for internal API access (e.g., pages)",
	Long: `Store a session cookie for endpoints that require session-based auth.

Some Plane endpoints (e.g., pages) use internal APIs that don't accept
API keys. Copy the session_id cookie value from your browser's dev tools
(Application → Cookies → session_id) and paste it here.`,
	Example: `  plane auth session
  plane auth session --api-key <session_id_value>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := auth.LoadConfig()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		profile := cfg.ActiveProfile

		// Get session cookie value
		sessionCookie := flagAPIKey // reuse --api-key flag for convenience
		if sessionCookie == "" {
			fmt.Fprint(os.Stderr, "Session cookie (session_id value from browser): ")
			tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			if err != nil {
				return fmt.Errorf("reading session cookie: %w", err)
			}
			sessionCookie = strings.TrimSpace(string(tokenBytes))
		}
		if sessionCookie == "" {
			return fmt.Errorf("session cookie is required")
		}

		// Validate by trying to access the API
		profileCfg := cfg.ActiveProfileConfig()
		apiURL := profileCfg.APIURL
		if flagAPIURL != "" {
			apiURL = flagAPIURL
		}
		if apiURL == "" {
			return fmt.Errorf("no API URL configured. Run 'plane auth login' first")
		}

		client := api.NewSessionClient(apiURL, sessionCookie, profileCfg.Workspace, flagVerbose, os.Stderr)
		users := api.NewUsersService(client)
		user, err := users.Me(context.Background())
		if err != nil {
			return fmt.Errorf("session validation failed: %w", err)
		}

		// Store in keyring
		store, err := auth.NewKeyringStore("")
		if err != nil {
			return fmt.Errorf("opening keyring: %w", err)
		}
		if err := store.Set(profile+"/session-token", sessionCookie); err != nil {
			return fmt.Errorf("storing session cookie: %w", err)
		}

		if !flagQuiet {
			fmt.Fprintf(os.Stderr, "Session stored for %s (%s)\n", user.DisplayName, user.Email)
			fmt.Fprintf(os.Stderr, "Profile: %s\n", profile)
			fmt.Fprintf(os.Stderr, "Session cookie stored in OS keyring.\n")
		}
		return nil
	},
}

var authSwitchCmd = &cobra.Command{
	Use:   "switch <profile>",
	Short: "Switch to a different profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := auth.LoadConfig()
		if err != nil {
			return err
		}

		store, err := auth.NewKeyringStore("")
		if err != nil {
			return fmt.Errorf("opening keyring: %w", err)
		}

		pm := &auth.ProfileManager{Config: cfg, Store: store}
		if err := pm.Switch(args[0]); err != nil {
			return err
		}

		if !flagQuiet {
			fmt.Fprintf(os.Stderr, "Switched to profile %q.\n", args[0])
		}
		return nil
	},
}
