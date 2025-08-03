package auth

import (
	"errors"
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
func FromRequest(w http.ResponseWriter, r *http.Request, secret []byte) (User, bool, error) {
	tokenString := r.URL.Query().Get(authParamName)
	fromURL := true

	// Check for token in GET parameter first (has priority)
	if tokenString == "" {
		// Fall back to cookie
		cookie, err := r.Cookie(authCookieName)
		if err != nil {
			return User{}, false, nil
		}
		tokenString = cookie.Value
		fromURL = false
	}

	var user User

	_, err := jwt.ParseWithClaims(tokenString, &user, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return User{}, false, fmt.Errorf("parsing token: %w", err)
	}

	if fromURL {
		http.SetCookie(w, &http.Cookie{
			Name:     authCookieName,
			Value:    tokenString,
			Path:     "/",
			MaxAge:   int(loginTime.Seconds()),
			Secure:   true,
			HttpOnly: true,
		})

	}

	return user, fromURL, nil
}

func createToken(mail string, validtime time.Duration, secret []byte) (string, error) {
	// Claims mit Ablaufzeit erstellen
	claims := User{
		Mail: mail,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(validtime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return tokenString, nil
}

// SetCookie sets the cookie to the response.
func (u User) SetCookie(w http.ResponseWriter, secret []byte) error {
	tokenString, err := createToken(u.Mail, loginTime, secret)
	if err != nil {
		return fmt.Errorf("create token: %w", err)
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
func (u User) SetURL(urlString string, secret []byte) (string, error) {
	tokenString, err := createToken(u.Mail, loginTime, secret)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
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

func IsExpiredError(err error) bool {
	return errors.Is(err, jwt.ErrTokenExpired)
}
