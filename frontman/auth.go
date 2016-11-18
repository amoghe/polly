package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"

	goji "goji.io"

	"goji.io/pat"
)

const (
	// RouteLogin is used for Github OAuth2 login flow
	RouteLogin = "/github/login"
	// RouteCallback is used for the Github OAuth2 callback
	RouteCallback = "/github/callback"
	// RouteBackdoor is used to set session cookie for a given PAT
	RouteBackdoor = "/github/backdoor"
	// RouteLogout is the route to logout
	RouteLogout = "/logout"
)

// AuthenticatingRouter is an http.Handler that can additionally return the github.Client for the currently
// authenticated user (based on the oauth token saved in the session state)
type AuthenticatingRouter interface {
	AuthTokenFromRequest(*http.Request) (*oauth2.Token, error)
	http.Handler
}

// authRouter is the mux that handles all auth related routes
type authRouter struct {
	mux                *goji.Mux
	oauth2Config       oauth2.Config
	githubConfig       GithubAppConfig
	stateCookieMaker   CookieMaker
	sessionCookieMaker CookieMaker
}

// NewAuthRouter returns a http.Handler that handles routes pertaining to authentication
func NewAuthRouter(githubCfg GithubAppConfig, oauth2Cfg oauth2.Config) AuthenticatingRouter {
	a := authRouter{
		mux:                goji.SubMux(),
		githubConfig:       githubCfg,
		oauth2Config:       oauth2Cfg,
		stateCookieMaker:   stateCookieMaker,
		sessionCookieMaker: sessionCookieMaker,
	}
	a.mux.HandleFunc(pat.Get(RouteLogin), a.HandleLogin)
	a.mux.HandleFunc(pat.Get(RouteCallback), a.HandleCallback)
	a.mux.HandleFunc(pat.Post(RouteLogout), a.HandleLogout)
	a.mux.HandleFunc(pat.Post(RouteBackdoor), a.HandleBackdoor)
	return &a
}

// ServeHTTP allows authRouter satisfy the http.Handler interface
func (a *authRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

// HandleLogout destroys the session on POSTs and redirects to home.
func (a *authRouter) HandleLogout(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		sessionStore.Destroy(w, a.sessionCookieMaker.Name)
	}
	http.Redirect(w, req, "/", http.StatusFound)
}

// HandleLogin starts the OAuth login process
func (a *authRouter) HandleLogin(w http.ResponseWriter, r *http.Request) {

	state, err := a.getSessionState(r)
	if err != nil {
		http.Redirect(w, r, "/auth"+RouteLogin, http.StatusFound)
		return
	}

	client := github.NewClient(&http.Client{
		Transport: &github.BasicAuthTransport{
			Username: a.githubConfig.GithubClientID,
			Password: a.githubConfig.GithubClientSecret,
		},
	})
	auth, _, err := client.Authorizations.Check(a.githubConfig.GithubClientID, state.OAuth2Token.AccessToken)
	if err != nil {
		log.Println("auth check err:", err)
		http.Redirect(w, r, "/auth"+RouteLogin, http.StatusFound)
		return
	}
	log.Println("auth is:", auth)

	// TODO: check auth.Scopes for sufficient permissions

	randomState := a.setRandomState(w)
	http.Redirect(w, r, a.oauth2Config.AuthCodeURL(randomState), http.StatusFound)
}

// HandleCallback handles the outh callback
func (a *authRouter) HandleCallback(w http.ResponseWriter, r *http.Request) {

	// compare the random state from the callback and the cookie
	if err := r.ParseForm(); err != nil {
		handleUnauthorized(w, "Failed to parse form (oauth2 callback)")
		return
	}

	randState1, err := a.getRandomState(r)
	if err != nil {
		handleUnauthorized(w, "failed to extract random CSRF state (oauth2 callback)")
		return
	}
	randState2 := r.Form.Get("state")
	if randState2 == "" {
		handleUnauthorized(w, "request missing code or state (oauth2 callback)")
		return
	}
	if randState1 != randState2 {
		handleUnauthorized(w, "mismatched state, please retry auth (oauth2 callback)")
		return
	}

	authCode := r.Form.Get("code") // Github docs say this is the code
	token, err := a.oauth2Config.Exchange(r.Context(), authCode)
	if err != nil {
		handleUnauthorized(w, "failed to exchange token (oauth2 callback)")
		return
	}

	client := github.NewClient(a.oauth2Config.Client(r.Context(), token))
	usr, _, err := client.Users.Get("")
	if err != nil {
		handleUnauthorized(w, "failed to get authenticated user")
		return
	}

	state := sessionState{UserID: *usr.ID, OAuth2Token: *token}
	a.setSessionState(w, state)

	// TODO: send them to their "homepage"
	http.Redirect(w, r, "/github/organizations", http.StatusFound)
}

//HandleBackdoor sets the session cookie when provided a legit PAT
func (a *authRouter) HandleBackdoor(w http.ResponseWriter, r *http.Request) {
	pats, there := r.URL.Query()["pat"]
	if !there || len(pats) <= 0 {
		handleMissingParam(w, errors.New("Missing PAT token"))
		return
	}
	if len(pats[0]) <= 0 {
		handleMissingParam(w, errors.New("Invalid PAT (zero length)"))
		return
	}

	token := oauth2.Token{AccessToken: pats[0]}
	client := github.NewClient(a.oauth2Config.Client(r.Context(), &token))
	usr, _, err := client.Users.Get("")
	if err != nil {
		handleUnauthorized(w, "failed to get authenticated user with given token")
		return
	}

	state := sessionState{UserID: *usr.ID, OAuth2Token: token}
	a.setSessionState(w, state)
	log.Println("Issued backdoor session for user:", usr)
}

//
// - - - Helpers - - -
//

// hasSessionState returns true if the user has a cookie containing session state.
func (a *authRouter) hasSessionState(req *http.Request) bool {
	if _, err := sessionStore.Get(req, a.sessionCookieMaker.Name); err == nil {
		return true
	}
	return false
}

// generate random state and set it in the request (cookie) as well as return it
func (a *authRouter) setRandomState(w http.ResponseWriter) string {
	rnd := make([]byte, 32)
	rand.Read(rnd)

	val := base64.StdEncoding.EncodeToString(rnd)
	http.SetCookie(w, a.stateCookieMaker.NewCookie(val))

	return val
}

// get the random state value from the cookie (we set earlier)
func (a *authRouter) getRandomState(r *http.Request) (string, error) {
	stateCookie, err := r.Cookie(a.stateCookieMaker.Name)
	if err != nil {
		return "", err
	}
	return stateCookie.Value, err
}

// set the session state (user ID and auth token) in the session cookie
func (a *authRouter) setSessionState(w http.ResponseWriter, sc sessionState) error {
	val, err := json.Marshal(&sc)
	if err != nil {
		return err
	}
	b64 := base64.URLEncoding.EncodeToString(val)
	http.SetCookie(w, a.sessionCookieMaker.NewCookie(b64))
	return nil
}

// get the session state from the session cookie
func (a *authRouter) getSessionState(r *http.Request) (sessionState, error) {
	sessionCookie, err := r.Cookie(a.sessionCookieMaker.Name)
	if err != nil {
		return sessionState{}, err
	}
	dec, err := base64.URLEncoding.DecodeString(sessionCookie.Value)
	if err != nil {
		return sessionState{}, err
	}
	sc := sessionState{}
	if err := json.Unmarshal([]byte(dec), &sc); err != nil {
		return sc, err
	}
	return sc, nil
}

// AuthTokenFromRequest returns the oauth2 token from the request (session cookie)
func (a *authRouter) AuthTokenFromRequest(r *http.Request) (*oauth2.Token, error) {
	state, err := a.getSessionState(r)
	if err != nil {
		return nil, err
	}
	return &state.OAuth2Token, nil
}
