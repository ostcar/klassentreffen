package web

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"iter"
	"log"
	"maps"
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/ostcar/klassentreffen/auth"
	"github.com/ostcar/klassentreffen/config"
	"github.com/ostcar/klassentreffen/mail"
	"github.com/ostcar/klassentreffen/model"
	"github.com/ostcar/klassentreffen/web/template"
	"github.com/ostcar/sticky"
)

const emailText = `Hallo,

klicke auf den folgenden Link, um dich anzumelden:

%s

Dieser Link ist 24 Stunden gültig.

Viele Grüße
Das Klassentreffen-Team`

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
	cfg   config.Config
	model *sticky.Sticky[model.Model]
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

	mux.Handle("/assets/", handleStatic())
	mux.Handle("/", handleError(s.handleAuth(s.handleHome)))
	mux.Handle("/login", handleError(s.handleLogin))
	mux.Handle("/logout", handleError(s.handleLogout))
	mux.Handle("/save", handleError(s.handleAuth(s.handleParticipantSave)))
	mux.Handle("/admin", handleError(s.handleAuth(s.handleAdminSave)))

	s.Handler = mux
}

// handleHome handles a GET-Request of a logged in user.
//
// If the user exists, shows the participant list. Filtered for non admins.
//
// If the user does not exist, the create form is shown.
func (s server) handleHome(w http.ResponseWriter, r *http.Request, user auth.User) error {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return nil
	}

	model, close := s.model.ForReading()
	defer close()

	me, exists := model.Participant[user.Mail]

	if !exists {
		// User does not exist in database, show create page
		return template.ParticipantFormCreate(user.Mail, "").Render(r.Context(), w)
	}

	participants := maps.Values(model.Participant)
	if !me.Admin {
		participants = filterPublic(filterVerified(participants))
	}

	return template.ParticipantList(slices.Collect(participants), me).Render(r.Context(), w)
}

func filterPublic(full iter.Seq[model.Participant]) iter.Seq[model.Participant] {
	return func(yield func(model.Participant) bool) {
		for participant := range full {
			if participant.Public {
				yield(participant)
			}
		}
	}
}

func filterVerified(full iter.Seq[model.Participant]) iter.Seq[model.Participant] {
	return func(yield func(model.Participant) bool) {
		for participant := range full {
			if participant.Verified {
				yield(participant)
			}
		}
	}
}

// handleLogin accepts POST-Requests to login.
//
// It sends the login email.
func (s server) handleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return nil
	}

	email := strings.TrimSpace(r.FormValue("email"))
	if email == "" {
		return template.Login("E-Mail-Adresse ist erforderlich").Render(r.Context(), w)
	}

	// Create JWT token for email
	user := auth.User{
		Mail: email,
	}

	// Create login URL with token
	loginURL, err := user.SetURL(s.cfg.BaseURL+"/", []byte(s.cfg.Secret))
	if err != nil {
		return fmt.Errorf("creating login URL: %w", err)
	}

	if err := mail.Send(s.cfg, email, fmt.Sprintf(emailText, loginURL)); err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	return template.LoginSent().Render(r.Context(), w)
}

func (s server) handleLogout(w http.ResponseWriter, r *http.Request) error {
	auth.Logout(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}

// handleParticipantSave creates or updates an participant.
func (s server) handleParticipantSave(w http.ResponseWriter, r *http.Request, user auth.User) error {
	m, saveEvent, close := s.model.ForWriting()
	defer close()

	me, isUpdate := m.Participant[user.Mail]

	if r.Method != "POST" {
		if isUpdate {
			return template.ParticipantFormUpdate(user.Mail, me, "Name is required").Render(r.Context(), w)
		}
		return template.ParticipantFormCreate(user.Mail, "Name is required").Render(r.Context(), w)
	}

	formMail := strings.TrimSpace(r.FormValue("email"))
	if formMail != user.Mail {
		log.Printf("User messages with email. auth-mail: %s, form-mail: %s", user.Mail, formMail)
		return s.handleLogout(w, r)
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		if isUpdate {
			return template.ParticipantFormUpdate(user.Mail, me, "Name is required").Render(r.Context(), w)
		}
		return template.ParticipantFormCreate(user.Mail, "Name is required").Render(r.Context(), w)
	}

	isFirstUser := len(m.Participant) == 0

	participant := model.Participant{
		Mail:     user.Mail,
		Name:     name,
		OldName:  strings.TrimSpace(r.FormValue("old_name")),
		Info:     r.FormValue("info") == "on",
		Attend:   r.FormValue("attend") == "on",
		Public:   r.FormValue("public") == "on",
		Admin:    me.Admin || isFirstUser,
		Verified: me.Verified || isFirstUser,
	}

	if err := saveEvent(m.SaveParticipant(participant)); err != nil {
		return fmt.Errorf("save participant: %w", err)
	}

	// Redirect to home
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}

func (s server) handleAdminSave(w http.ResponseWriter, r *http.Request, user auth.User) error {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return nil
	}

	m, saveEvent, close := s.model.ForWriting()
	defer close()

	me, ok := m.Participant[user.Mail]
	if !ok || !me.Admin {
		w.WriteHeader(http.StatusForbidden)
		return nil
	}

	email := r.FormValue("email")
	if email == "" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return nil
	}

	participant, exists := m.Participant[email]

	if !exists {
		// TODO: somehow show the error to the user
		return nil
	}

	newEmail := strings.TrimSpace(r.FormValue("new_email"))

	if _, alreadyExists := m.Participant[newEmail]; alreadyExists {
		// TODO: somehow show the error to the user
		return nil
	}

	participant.Mail = newEmail
	participant.Name = strings.TrimSpace(r.FormValue("name"))
	participant.OldName = strings.TrimSpace(r.FormValue("old_name"))
	participant.Info = r.FormValue("info") == "on"
	participant.Attend = r.FormValue("attend") == "on"
	participant.Public = r.FormValue("public") == "on"
	participant.Admin = r.FormValue("admin") == "on"
	participant.Verified = r.FormValue("verified") == "on"

	if err := saveEvent(m.SaveParticipant(participant)); err != nil {
		return fmt.Errorf("save participant: %w", err)
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
	return nil
}

func handleStatic() http.Handler {
	files, err := fs.Sub(publicFiles, "assets")
	if err != nil {
		// This only happens on startup time.
		panic(err)
	}

	return http.StripPrefix("/assets/", http.FileServer(http.FS(files)))
}

func handleError(handler func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			log.Printf("Error: %v", err)
			http.Error(w, "Ups, something went wrong", 500)
		}
	}
}

type errorHandler func(w http.ResponseWriter, r *http.Request) error
type authenticatedHandler func(w http.ResponseWriter, r *http.Request, user auth.User) error

func (s server) handleAuth(next authenticatedHandler) errorHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		user, fromURL, err := auth.FromRequest(w, r, []byte(s.cfg.Secret))
		if err != nil {
			return fmt.Errorf("read auth data: %w", err)
		}

		if fromURL {
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
			return nil
		}

		if user.IsAnonymous() {
			// User is not authenticated, show login page
			return template.Login("").Render(r.Context(), w)
		}

		return next(w, r, user)
	}
}
