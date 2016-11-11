package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	goji "goji.io"
	"goji.io/pat"

	"github.com/alioygur/gores"
	"github.com/dghubble/gologin"
	"github.com/dghubble/sessions"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"

	"golang.org/x/oauth2"
)

var (
	// GithubAuthURL is the endpoints that github exposes for oauth2
	GithubAuthURL = "https://github.com/login/oauth/authorize"
	// GithubTokenURL is the URL at which we get the token
	GithubTokenURL = "https://github.com/login/oauth/access_token"

	// sessionStore encodes and decodes session data stored in signed cookies
	sessionStore = sessions.NewCookieStore([]byte("pollySecret"), nil)

	sessionCookieConfig = gologin.CookieConfig{
		Name:     "session-cookie",
		Path:     "/",
		MaxAge:   3600, // FIXME
		HTTPOnly: true,
		Secure:   false, // FIXME
	}

	stateCookieConfig = gologin.CookieConfig{
		Name:     "state-cookie",
		Path:     "/",
		MaxAge:   60,
		HTTPOnly: true,
		Secure:   false, // FIXME
	}
)

// GithubAppConfig holds the config for our Github app
type GithubAppConfig struct {
	GithubClientID     string
	GithubClientSecret string
}

type sessionState struct {
	UserID      int
	OAuth2Token oauth2.Token
}

// Server represents the server
type Server struct {
	githubAppConfig     GithubAppConfig
	stateCookieConfig   gologin.CookieConfig
	sessionCookieConfig gologin.CookieConfig
	oauth2Config        oauth2.Config
	mux                 *goji.Mux
}

// main creates and starts a Server listening.
func main() {
	var (
		listenAddress = "0.0.0.0:8080"

		config = &GithubAppConfig{
			GithubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
			GithubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		}
	)
	// allow consumer credential flags to override config fields
	clientID := flag.String("client-id", "", "Github Client ID")
	clientSecret := flag.String("client-secret", "", "Github Client Secret")
	flag.Parse()

	if *clientID != "" {
		config.GithubClientID = *clientID
	}
	if *clientSecret != "" {
		config.GithubClientSecret = *clientSecret
	}
	if config.GithubClientID == "" {
		log.Fatal("Missing Github Client ID")
	}
	if config.GithubClientSecret == "" {
		log.Fatal("Missing Github Client Secret")
	}

	log.Printf("Starting Server listening on %s\n", listenAddress)
	err := http.ListenAndServe(listenAddress, NewServer(config))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// NewServer returns a new ServeMux with app routes.
func NewServer(config *GithubAppConfig) *Server {

	server := Server{
		githubAppConfig:     *config,
		stateCookieConfig:   stateCookieConfig,
		sessionCookieConfig: sessionCookieConfig,
		oauth2Config: oauth2.Config{
			ClientID:     config.GithubClientID,
			ClientSecret: config.GithubClientSecret,
			RedirectURL:  "http://localhost:8080/github/callback",
			Endpoint:     oauth2.Endpoint{AuthURL: GithubAuthURL, TokenURL: GithubTokenURL},
			Scopes: []string{
				"read:public_key",
				"read:org",
			},
		},
		mux: goji.NewMux(),
	}

	// Auth routes
	server.mux.HandleFunc(pat.New("/auth/github/login"), server.HandleLogin)
	server.mux.HandleFunc(pat.New("/github/callback"), server.HandleCallback)
	server.mux.HandleFunc(pat.New("/logout"), server.HandleLogout)
	// Github API routes
	server.mux.HandleFunc(pat.New("/github/organizations"), server.ListGithubOrganizations)
	server.mux.HandleFunc(pat.New("/github/organizations/:org_name/repositories"), server.ListGithubRepositoriesForOrganization)
	// Gerrit
	server.mux.HandleFunc(pat.New("gerrit/import/repository"), server.CreateGerritRepository)

	return &server
}

// ServeHTTP allows Server to be a mux
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//log.Println("Serving req:", r.URL.String())
	s.mux.ServeHTTP(w, r)
}

// HandleLogout destroys the session on POSTs and redirects to home.
func (s *Server) HandleLogout(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		sessionStore.Destroy(w, s.sessionCookieConfig.Name)
	}
	http.Redirect(w, req, "/", http.StatusFound)
}

// HandleLogin starts the OAuth login process
func (s *Server) HandleLogin(w http.ResponseWriter, req *http.Request) {
	if s.hasSessionState(req) {
		// already authenticated
		http.Redirect(w, req, "/github/organizations", http.StatusFound)
		return
	}

	randomState := s.setRandomState(w)
	http.Redirect(w, req, s.oauth2Config.AuthCodeURL(randomState), http.StatusFound)
}

// HandleCallback handles the outh callback
func (s *Server) HandleCallback(w http.ResponseWriter, req *http.Request) {
	// compare the random state from the callback and the cookie

	if err := req.ParseForm(); err != nil {
		handleError(w, errors.Wrap(err, "Failed to parse form (oauth2 callback)"))
	}

	randState1, err := s.getRandomState(req)
	if err != nil {
		handleError(w, errors.Wrap(err, "failed to extract random CSRF state (oauth2 callback)"))
	}
	randState2 := req.Form.Get("state")
	if randState2 == "" {
		handleError(w, errors.Wrap(err, "request missing code or state (oauth2 callback)"))
	}
	if randState1 != randState2 {
		handleError(w, errors.Wrap(err, "mismatched state, please retry auth (oauth2 callback)"))
	}

	authCode := req.Form.Get("code") // Github docs say this is the code

	token, err := s.oauth2Config.Exchange(req.Context(), authCode)
	if err != nil {
		handleError(w, errors.Wrap(err, "failed to exchange token (oauth2 callback)"))
	}
	s.setSessionState(w, sessionState{UserID: 0, OAuth2Token: *token})

	// TODO: send them to their "homepage"
	http.Redirect(w, req, "/github/organizations", http.StatusFound)
}

//
// - - - Helpers - - -
//

func handleError(w http.ResponseWriter, err error) {
	log.Printf("ERR: (%T) %s\n", err, err)

	var retcode int
	switch err.(type) {
	case *github.ErrorResponse:
		gherr, _ := err.(*github.ErrorResponse)
		retcode = gherr.Response.StatusCode
	default:
		retcode = http.StatusBadRequest
	}
	gores.JSON(w, retcode, struct{ Error string }{Error: err.Error()})
}

// hasSessionState returns true if the user has a cookie containing session state.
func (s *Server) hasSessionState(req *http.Request) bool {
	if _, err := sessionStore.Get(req, s.sessionCookieConfig.Name); err == nil {
		return true
	}
	return false
}

// generate random state and set it in the request (cookie) as well as return it
func (s *Server) setRandomState(w http.ResponseWriter) string {
	rnd := make([]byte, 32)
	rand.Read(rnd)

	val := base64.StdEncoding.EncodeToString(rnd)
	http.SetCookie(w, newCookieFromConfig(s.stateCookieConfig, val))

	return val
}

// get the random state value from the cookie (we set earlier)
func (s *Server) getRandomState(r *http.Request) (string, error) {
	stateCookie, err := r.Cookie(s.stateCookieConfig.Name)
	if err != nil {
		return "", err
	}
	return stateCookie.Value, err
}

// set the session state (user ID and auth token) in the session cookie
func (s *Server) setSessionState(w http.ResponseWriter, sc sessionState) error {
	val, err := json.Marshal(&sc)
	if err != nil {
		return err
	}
	b64 := base64.URLEncoding.EncodeToString(val)
	http.SetCookie(w, newCookieFromConfig(s.sessionCookieConfig, b64))
	return nil
}

// get the session state from the session cookie
func (s *Server) getSessionState(r *http.Request) (sc sessionState, err error) {
	sessionCookie, err := r.Cookie(s.sessionCookieConfig.Name)
	if err != nil {
		return
	}
	dec, err := base64.URLEncoding.DecodeString(sessionCookie.Value)
	if err != nil {
		return
	}
	json.Unmarshal([]byte(dec), &sc)
	return
}

func (s *Server) newGithubClientFromSessionState(ctx context.Context, sc sessionState) *github.Client {
	clt := s.oauth2Config.Client(ctx, &sc.OAuth2Token)
	return github.NewClient(clt)
}

// NewCookie returns a new http.Cookie with the given value and CookieConfig
// properties (name, max-age, etc.).
//
// The MaxAge field is used to determine whether an Expires field should be
// added for Internet Explorer compatability and what its value should be.
func newCookieFromConfig(config gologin.CookieConfig, value string) *http.Cookie {
	cookie := &http.Cookie{
		Name:     config.Name,
		Value:    value,
		Domain:   config.Domain,
		Path:     config.Path,
		MaxAge:   config.MaxAge,
		HttpOnly: config.HTTPOnly,
		Secure:   config.Secure,
	}
	// IE <9 does not understand MaxAge, set Expires if MaxAge is non-zero.
	// if expires, ok := expiresTime(config.MaxAge); ok {
	// 	cookie.Expires = expires
	// }
	return cookie
}
