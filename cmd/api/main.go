package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BadrChoubai/resonance/internal/auth"
	"github.com/BadrChoubai/resonance/internal/server"
	"github.com/BadrChoubai/resonance/internal/spotify"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(ctx, log); err != nil {
		log.Error("application exited", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger) error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	sp := spotify.New(cfg.clientID, cfg.clientSecret, cfg.redirectURI.String())
	a, err := auth.New(log, sp, auth.NewMemoryStore(), cfg.sessionKey, cfg.redirectURI.Scheme == "https")
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      server.New(log, a, sp),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

type config struct {
	clientID     string
	clientSecret string
	redirectURI  *url.URL
	sessionKey   []byte
}

func loadConfig() (config, error) {
	var cfg config
	var missing []string
	get := func(name string) string {
		v := os.Getenv(name)
		if v == "" {
			missing = append(missing, name)
		}
		return v
	}
	cfg.clientID = get("SPOTIFY_CLIENT_ID")
	cfg.clientSecret = get("SPOTIFY_CLIENT_SECRET")
	rawRedirect := get("SPOTIFY_REDIRECT_URI")
	rawKey := get("SESSION_KEY")
	if len(missing) > 0 {
		return config{}, fmt.Errorf("missing environment variables: %v", missing)
	}

	u, err := url.Parse(rawRedirect)
	if err != nil || u.Host == "" {
		return config{}, fmt.Errorf("SPOTIFY_REDIRECT_URI is not an absolute URL: %q", rawRedirect)
	}
	cfg.redirectURI = u

	key, err := base64.StdEncoding.DecodeString(rawKey)
	if err != nil {
		return config{}, fmt.Errorf("SESSION_KEY is not valid base64: %w", err)
	}
	cfg.sessionKey = key
	return cfg, nil
}
