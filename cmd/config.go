package cmd

import (
	"github.com/spf13/cobra"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"path"
	"time"
)

var cfg = newDefaultConfig()
var cfgFile = or(os.Getenv("TODOLIST_CONFIG_PATH"), path.Join(homeDir, "config.yaml"))

func init() {
	rootCmd.AddCommand(configPersistCmd)
}

var configPersistCmd = &cobra.Command{
	Use:   "config-persist",
	Short: "Persist config with defaults to filesystem",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := yaml.Marshal(cfg)
		if err != nil {
			log.Fatalf("cant marshal config: %s", err.Error())
		}
		if err := os.WriteFile(cfgFile, data, 0600); err != nil {
			log.Fatalf("cant write config: %s", err.Error())
		}
		log.Println("config written at", cfgFile)
	},
}

type Config struct {
	Server struct {
		DiagnosticEndpointsEnabled bool   `yaml:"diagnostic_endpoints_enabled"`
		DatabaseFile               string `yaml:"database_file"`
		ListenAddr                 string `yaml:"listen_addr"`
		PublicUrl                  string `yaml:"public_url"`
		AuthEnabled                bool   `yaml:"auth_enabled"`
		TokenAuth                  struct {
			Enabled     bool   `yaml:"enabled"`
			ClientToken string `yaml:"client_token"`
		} `yaml:"token_auth"`
		BaseAuth struct {
			Enabled  bool   `yaml:"enabled"`
			Login    string `yaml:"login"`
			Password string `yaml:"password"`
		} `yaml:"base_auth"`
		OidcAuth struct {
			Enabled         bool     `yaml:"enabled"`
			ClientId        string   `yaml:"client_id"`
			ClientSecret    string   `yaml:"client_secret"`
			IssuerUrl       string   `yaml:"issuer_url"`
			Scopes          []string `yaml:"scopes"`
			CookieKey       string   `yaml:"cookie_key"`
			WhitelistEmails []string `yaml:"whitelist_emails"`
		} `yaml:"oidc_auth"`
		Telegram struct {
			Enabled        bool   `yaml:"enabled"`
			Token          string `yaml:"token"`
			UserId         int64  `yaml:"userId"`
			EverydayAgenda struct {
				Enabled bool      `yaml:"enabled"`
				At      time.Time `yaml:"at"`
			} `yaml:"everyday_agenda"`
		} `yaml:"telegram"`
	}
	Client struct {
		RemoteAddr  string `yaml:"remote_addr"`
		ServerToken string `yaml:"server_token"`
	}
}

func newDefaultConfig() *Config {
	c := &Config{}
	c.Server.ListenAddr = ":8080"
	c.Server.DatabaseFile = path.Join(homeDir, "database.json")
	c.Server.DiagnosticEndpointsEnabled = true

	c.Server.TokenAuth.ClientToken = "api_password"

	c.Server.OidcAuth.Scopes = []string{oidc.ScopeOpenID, oidc.ScopeEmail, oidc.ScopeProfile}
	c.Server.OidcAuth.IssuerUrl = "https://accounts.google.com"
	c.Server.OidcAuth.CookieKey = "kiel4teof4Eoziheigiesh7ooquiepho" //pwgen 32

	c.Client.RemoteAddr = "http://127.0.0.1:8080"
	c.Client.ServerToken = "api_password"

	return c
}

func or[T any](x, y T) T {
	var zero T
	if any(x) != any(zero) {
		return x
	}
	return y
}
