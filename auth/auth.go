package auth

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	authCookieName = "klassentreffen"
	authParamName  = "token"
	loginTime      = 24 * time.Hour
)

// User represents a logged in user.
type User struct {
	jwt.RegisteredClaims

	Mail string `json:"mail"`
}

// FromRequest reads the user from a request.
func FromRequest(r *http.Request, secred []byte) (User, error) {
	tokenString := r.URL.Query().Get(authParamName)

	// Check for token in GET parameter first (has priority)
	if tokenString == "" {
		// Fall back to cookie
		cookie, err := r.Cookie(authCookieName)
		if err != nil {
			return User{}, fmt.Errorf("reading cookie: %w", err)
		}
		tokenString = cookie.Value
	}

	var user User

	if _, err := jwt.ParseWithClaims(tokenString, &user, func(token *jwt.Token) (any, error) {
		return secred, nil
	}); err != nil {
		return User{}, fmt.Errorf("parsing token: %w", err)
	}

	return user, nil
}

// SetCookie sets the cookie to the response.
func (u User) SetCookie(w http.ResponseWriter, secred []byte) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, u)

	tokenString, err := token.SignedString(secred)
	if err != nil {
		return fmt.Errorf("signing token: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    tokenString,
		Path:     "/",
		MaxAge:   int(loginTime.Seconds()),
		Secure:   true,
		HttpOnly: true,
	})

	return nil
}

// SetURL sets the JWT token as a GET parameter in the URL and returns the modified URL.
func (u User) SetURL(urlString string, secred []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, u)

	tokenString, err := token.SignedString(secred)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return "", fmt.Errorf("parsing URL: %w", err)
	}

	query := parsedURL.Query()
	query.Set(authParamName, tokenString)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

// Logout removes the auth cookie
func Logout(w http.ResponseWriter) {
	c := &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}

	http.SetCookie(w, c)
}

// IsAnonymous tells, if the user is not logged in.
func (u User) IsAnonymous() bool {
	return u.Mail == ""
}
