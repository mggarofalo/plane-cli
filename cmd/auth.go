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

		fmt.Fprintf(os.Stderr, "Authenticated as %s (%s)\n", user.DisplayName, user.Email)
		fmt.Fprintf(os.Stderr, "Profile: %s\n", profile)
		fmt.Fprintf(os.Stderr, "Credentials stored in OS keyring.\n")
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

		fmt.Fprintf(os.Stderr, "Credentials removed for profile %q.\n", profile)
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
			Env:       &auth.EnvSource{},
			Config:    cfg,
		}

		store, err := auth.NewKeyringStore("")
		if err == nil {
			resolver.Store = store
		}

		resolved, err := resolver.Resolve()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Not authenticated: %v\n", err)
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

		return formatter().Format(os.Stdout, status)
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

		fmt.Fprintf(os.Stderr, "Switched to profile %q.\n", args[0])
		return nil
	},
}
