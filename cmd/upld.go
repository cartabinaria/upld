package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/go-github/github"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/exp/slog"

	"github.com/cartabinaria/upld/auth"
	cartabinaria "github.com/cartabinaria/upld/github"
)

type Config struct {
	Listen     string   `toml:"listen"`
	BaseURL    string   `toml:"base_url"`
	ClientURLs []string `toml:"client_urls"`

	AppID          string `toml:"app_id" required:"true"`
	InstallationID string `toml:"installation_id" required:"true"`
	PrivateKeyPath string `toml:"private_key_path" required:"true"`

	OAuthClientID        string        `toml:"oauth_client_id" required:"true"`
	OAuthClientSecret    string        `toml:"oauth_client_secret" required:"true"`
	OAuthSigningKey      string        `toml:"oauth_signing_key" required:"true"`
	OAuthSessionDuration time.Duration `toml:"oauth_session_duration"`
}

var (
	// Default config values
	config = Config{
		Listen:               "0.0.0.0:3000",
		BaseURL:              "http://localhost:3000",
		OAuthSessionDuration: time.Hour * 12,
	}
	client *github.Client
)

func main() {
	err := loadConfig()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		slog.Error("failed to parse baseURL", "err", err)
		os.Exit(1)
	}

	// client, err = initializeClient()
	// if err != nil {
	// 	slog.Error("failed to initialize client", "err", err)
	// 	os.Exit(1)
	// }
	//
	// // TODO: From here it's all testing code
	// repositories, _, err := client.Repositories.List(context.Background(), "cartabinaria", nil)
	// if err != nil {
	// 	slog.Error("failed to list repositories", "err", err)
	// 	os.Exit(1)
	// }
	//
	// for _, repository := range repositories {
	// 	println(repository.GetName())
	// }
	//
	// comment, _, err := client.Issues.CreateComment(
	// 	context.Background(),
	// 	"cartabinaria",
	// 	"ing-sistemi-informativi-test",
	// 	1,
	// 	&github.IssueComment{
	// 		Body: github.String("Hello, world!"),
	// 	})
	// if err != nil {
	// 	slog.Error("failed to create comment", "err", err)
	// 	os.Exit(1)
	// }
	//
	// slog.Info("comment created", "id", comment.GetID(), "url", comment.GetURL())

	authenticator := auth.NewAuthenticator(&auth.Config{
		BaseURL:      baseURL,
		ClientID:     config.OAuthClientID,
		ClientSecret: config.OAuthClientSecret,
		SigningKey:   []byte(config.OAuthSigningKey),
		Expiration:   config.OAuthSessionDuration,
	})

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("/login", authenticator.LoginHandler)
	mux.HandleFunc("/login/callback", authenticator.CallbackHandler)
	mux.HandleFunc("/whoami", authenticator.WhoAmIHandler)

	// Middlewares
	handler := NewCors(config.ClientURLs, true, mux)

	slog.Info("listening at", "address", config.Listen)
	err = http.ListenAndServe(config.Listen, handler)
	if err != nil {
		slog.Error("failed to serve", "err", err)
	}
}

func initializeClient() (*github.Client, error) {
	app, err := cartabinaria.NewApp(&cartabinaria.AppConfig{
		AppID:   config.AppID,
		KeyPath: config.PrivateKeyPath,
	})
	if err != nil {
		return nil, err
	}

	return app.AuthenticateAsInstallation(config.InstallationID)
}

func loadConfig() (err error) {
	file, err := os.Open("config.toml")
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}

	err = toml.NewDecoder(file).Decode(&config)
	if err != nil {
		return fmt.Errorf("failed to decode config file: %w", err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("failed to close config file: %w", err)
	}

	return nil
}
