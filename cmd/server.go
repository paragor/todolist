package cmd

import (
	"fmt"
	"log"

	"github.com/paragor/todo/pkg/cron"
	"github.com/paragor/todo/pkg/db"
	"github.com/paragor/todo/pkg/events"
	"github.com/paragor/todo/pkg/httpserver"
	"github.com/paragor/todo/pkg/models"
	"github.com/paragor/todo/pkg/service"
	"github.com/paragor/todo/pkg/telegram"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run todolist server",
	Run: func(cmd *cobra.Command, args []string) {
		runner := service.NewRunner()
		runnable := []service.Runnable{}
		var repo models.Repository
		{
			dbConfig := cfg.Server.Database
			switch dbConfig.Type {
			case "file":
				if dbConfig.File.Path == "" {
					log.Fatalln("database config: type is set as file, but path is not provided")
				}
				originRepo := db.NewInMemoryTasksRepository(cfg.Server.Database.File.Path)
				runnable = append(runnable, originRepo)
				repo = originRepo
			case "postgresql":
				if dbConfig.Postgresql.Url == "" {
					log.Fatalln("database config: type is set as postgresql, but url is not provided")
				}
				originRepo := db.NewPostgresqlTasksRepository(cfg.Server.Database.Postgresql.Url)
				runnable = append(runnable, originRepo)
				repo = originRepo
			}
		}
		repo = events.NewSpyRepository(repo)

		authConfig := &httpserver.AuthChainConfig{
			AuthBaseConfig:     nil,
			AuthTelegramConfig: nil,
			AuthTokenConfig:    nil,
		}
		if cfg.Server.Telegram.Enabled {
			if cfg.Server.Telegram.Token == "" {
				log.Fatalln("telegram token is empty")
			}
			if cfg.Server.Telegram.UserId == 0 {
				log.Fatalln("telegram user id is empty")
			}
			authConfig.AuthTelegramConfig = &httpserver.AuthTelegramConfig{
				Token:     cfg.Server.Telegram.Token,
				TrustedId: cfg.Server.Telegram.UserId,
			}
			telegramServer := telegram.NewTelegramServer(cfg.Server.Telegram.Token, cfg.Server.Telegram.UserId, cfg.Server.PublicUrl, repo)
			runnable = append(runnable, telegramServer)

			if cfg.Server.Telegram.EverydayAgenda.Enabled {
				runnable = append(runnable, cron.NewRepeatableCron(func() error {
					if err := telegramServer.TriggerAgenda(); err != nil {
						return fmt.Errorf("cant trigger agenda: %w", err)
					}
					return nil
				}, cron.RepeatEveryDayAt(cfg.Server.Telegram.EverydayAgenda.At)))
			}
		}
		if cfg.Server.TokenAuth.Enabled {
			if cfg.Server.TokenAuth.ClientToken == "" {
				log.Fatalln("TokenAuth.ClientToken is empty")
			}
			authConfig.AuthTokenConfig = &httpserver.AuthTokenConfig{Token: cfg.Server.TokenAuth.ClientToken}
		}
		if cfg.Server.BaseAuth.Enabled {
			if cfg.Server.BaseAuth.Login == "" {
				log.Fatalln("BaseAuth.Login is empty")
			}
			if cfg.Server.BaseAuth.Password == "" {
				log.Fatalln("BaseAuth.Password is empty")
			}
			authConfig.AuthBaseConfig = &httpserver.AuthBaseConfig{
				Login:    cfg.Server.BaseAuth.Login,
				Password: cfg.Server.BaseAuth.Password,
			}
		}
		if cfg.Server.OidcAuth.Enabled {
			if cfg.Server.OidcAuth.ClientId == "" {
				log.Fatalln("OidcAuth.ClientId is empty")
			}
			if cfg.Server.OidcAuth.ClientSecret == "" {
				log.Fatalln("OidcAuth.ClientSecret is empty")
			}
			if cfg.Server.OidcAuth.IssuerUrl == "" {
				log.Fatalln("OidcAuth.IssuerUrl is empty")
			}
			if cfg.Server.OidcAuth.CookieKey == "" {
				log.Fatalln("OidcAuth.CookieKey is empty")
			}
			if len(cfg.Server.OidcAuth.WhitelistEmails) == 0 {
				log.Fatalln("OidcAuth.CookieKey is empty")
			}
			if len([]byte(cfg.Server.OidcAuth.CookieKey)) != 32 {
				log.Fatalln("OidcAuth.CookieKey should be base64 of 32 bytes. example: 'pwgen 32'")
			}
			authConfig.AuthOidcConfig = &httpserver.AuthOidcConfig{
				ClientId:        cfg.Server.OidcAuth.ClientId,
				ClientSecret:    cfg.Server.OidcAuth.ClientSecret,
				IssuerUrl:       cfg.Server.OidcAuth.IssuerUrl,
				CookieKey:       cfg.Server.OidcAuth.CookieKey,
				WhitelistEmails: cfg.Server.OidcAuth.WhitelistEmails,
				Scopes:          cfg.Server.OidcAuth.Scopes,
			}
		}
		if !cfg.Server.AuthEnabled {
			authConfig = nil
		}
		httpServer, err := httpserver.NewHttpServer(
			cfg.Server.ListenAddr,
			repo,
			authConfig,
			cfg.Server.PublicUrl,
			cfg.Server.DiagnosticEndpointsEnabled,
		)
		if err != nil {
			log.Fatalln("cant create http server: %w", err)
		}

		runnable = append(runnable, httpServer)
		errChan := runner.Run(runnable...)
		log.Println("started!")
		haveAnyError := false
		for err := range errChan {
			if err != nil {
				haveAnyError = true
			}
			log.Println("application error chan", err)
		}
		if haveAnyError {
			log.Fatalln("something happened...")
		}
		log.Println("graceful exit")
	},
}
