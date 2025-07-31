package web

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"

	"github.com/ostcar/klassentreffen/auth"
	"github.com/ostcar/klassentreffen/config"
	"github.com/ostcar/klassentreffen/model"
	"github.com/ostcar/sticky"
)

//go:embed assets
var publicFiles embed.FS

//go:generate templ generate -path template

// Run starts the server.
func Run(ctx context.Context, s *sticky.Sticky[model.Model], cfg config.Config) error {
	handler := newServer(cfg, s)

	httpSRV := &http.Server{
		Addr:        cfg.WebListenAddr,
		Handler:     handler,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	// Shutdown logic in separate goroutine.
	wait := make(chan error)
	go func() {
		// Wait for the context to be closed.
		<-ctx.Done()

		if err := httpSRV.Shutdown(context.WithoutCancel(ctx)); err != nil {
			wait <- fmt.Errorf("HTTP server shutdown: %w", err)
			return
		}
		wait <- nil
	}()

	fmt.Printf("Listen webserver on: %s\n", cfg.WebListenAddr)
	if err := httpSRV.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("HTTP Server failed: %v", err)
	}

	return <-wait
}

type server struct {
	http.Handler
	cfg      config.Config
	model    *sticky.Sticky[model.Model]
	sendMail func(to string, text string) error
}

func newServer(cfg config.Config, s *sticky.Sticky[model.Model]) server {
	srv := server{
		cfg:   cfg,
		model: s,
	}
	srv.registerHandlers()

	return srv
}

func (s *server) registerHandlers() {
	mux := http.NewServeMux()

	mux.Handle("/assets", handleStatic())
	mux.Handle("/logout", handleError(s.handleLogout))

	mux.Handle("/", handleError(s.handleHome))
}

func (s server) handleHome(w http.ResponseWriter, r *http.Request) error {
	user, err := auth.FromRequest(r, []byte(s.cfg.Secret))
	if err != nil {
		return fmt.Errorf("failed to authenticate user: %w", err)
	}

	if user.IsAnonymous() {
		// TODO
		// - On Get Request, show login page.
		// - On POST Reqeust, get mail-adress
		// from body and send login-mail to this adress. Show user a message in german, that login email was send.

		// SendMail with: mail.Send(s.cfg, to string, text string)
		return nil
	}

	err = s.model.Read(func(model model.Model) error {
		if _, ok := model.Participant[user.Mail]; !ok {
			// TODO
			// User exists in Database. Show update-page
			return nil
		}
		// TODO
		// User does not exist in Database. Show create-page
		return nil
	})
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	return nil
}

// TODO: Create handler to create or update an participant, and list all participants

func (s server) handleLogout(w http.ResponseWriter, r *http.Request) error {
	auth.Logout(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}

func handleStatic() http.Handler {
	files, err := fs.Sub(publicFiles, "assets")
	if err != nil {
		// This only happens on startup time.
		panic(err)
	}

	return http.FileServer(http.FS(files))
}

func handleError(handler func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			log.Printf("Error: %v", err)
			http.Error(w, "Ups, something went wrong", 500)
		}
	}
}
