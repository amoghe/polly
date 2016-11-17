package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/google/go-github/github"

	goji "goji.io"

	"golang.org/x/oauth2"

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

// AuthRoutes returns a goji.Mux that handles routes pertaining to authentication
func (s *Server) AuthRoutes() *goji.Mux {
	m := goji.SubMux()
	m.HandleFunc(pat.Get(RouteLogin), s.HandleLogin)
	m.HandleFunc(pat.Get(RouteCallback), s.HandleCallback)
	m.HandleFunc(pat.Post(RouteLogout), s.HandleLogout)
	m.HandleFunc(pat.Post(RouteBackdoor), s.HandleBackdoor)
	return m
}

// HandleLogout destroys the session on POSTs and redirects to home.
func (s *Server) HandleLogout(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		sessionStore.Destroy(w, s.sessionCookieMaker.Name)
	}
	http.Redirect(w, req, "/", http.StatusFound)
}

// HandleLogin starts the OAuth login process
func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {

	state, err := s.getSessionState(r)
	if err != nil {
		http.Redirect(w, r, "/auth"+RouteLogin, http.StatusFound)
		return
	}

	client := github.NewClient(&http.Client{
		Transport: &github.BasicAuthTransport{
			Username: s.githubAppConfig.GithubClientID,
			Password: s.githubAppConfig.GithubClientSecret,
		},
	})
	auth, _, err := client.Authorizations.Check(s.githubAppConfig.GithubClientID, state.OAuth2Token.AccessToken)
	if err != nil {
		log.Println("auth check err:", err)
		http.Redirect(w, r, "/auth"+RouteLogin, http.StatusFound)
		return
	}
	log.Println("auth is:", auth)

	// TODO: check auth.Scopes for sufficient permissions

	randomState := s.setRandomState(w)
	http.Redirect(w, r, s.oauth2Config.AuthCodeURL(randomState), http.StatusFound)
}

// HandleCallback handles the outh callback
func (s *Server) HandleCallback(w http.ResponseWriter, r *http.Request) {

	// compare the random state from the callback and the cookie
	if err := r.ParseForm(); err != nil {
		handleUnauthorized(w, "Failed to parse form (oauth2 callback)")
		return
	}

	randState1, err := s.getRandomState(r)
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
	token, err := s.oauth2Config.Exchange(r.Context(), authCode)
	if err != nil {
		handleUnauthorized(w, "failed to exchange token (oauth2 callback)")
		return
	}

	client := github.NewClient(s.oauth2Config.Client(r.Context(), token))
	usr, _, err := client.Users.Get("")
	if err != nil {
		handleUnauthorized(w, "failed to get authenticated user")
		return
	}

	state := sessionState{UserID: *usr.ID, OAuth2Token: *token}
	s.setSessionState(w, state)

	// TODO: send them to their "homepage"
	http.Redirect(w, r, "/github/organizations", http.StatusFound)
}

// HandleBackdoor sets the session cookie when provided a legit PAT
func (s *Server) HandleBackdoor(w http.ResponseWriter, r *http.Request) {
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
	client := github.NewClient(s.oauth2Config.Client(r.Context(), &token))
	usr, _, err := client.Users.Get("")
	if err != nil {
		handleUnauthorized(w, "failed to get authenticated user with given token")
		return
	}

	state := sessionState{UserID: *usr.ID, OAuth2Token: token}
	s.setSessionState(w, state)
	log.Println("Issued backdoor session for user:", usr)
}

//
// - - - Helpers - - -
//

// hasSessionState returns true if the user has a cookie containing session state.
func (s *Server) hasSessionState(req *http.Request) bool {
	if _, err := sessionStore.Get(req, s.sessionCookieMaker.Name); err == nil {
		return true
	}
	return false
}

// generate random state and set it in the request (cookie) as well as return it
func (s *Server) setRandomState(w http.ResponseWriter) string {
	rnd := make([]byte, 32)
	rand.Read(rnd)

	val := base64.StdEncoding.EncodeToString(rnd)
	http.SetCookie(w, s.stateCookieMaker.NewCookie(val))

	return val
}

// get the random state value from the cookie (we set earlier)
func (s *Server) getRandomState(r *http.Request) (string, error) {
	stateCookie, err := r.Cookie(s.stateCookieMaker.Name)
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
	http.SetCookie(w, s.sessionCookieMaker.NewCookie(b64))
	return nil
}

// get the session state from the session cookie
func (s *Server) getSessionState(r *http.Request) (sessionState, error) {
	sessionCookie, err := r.Cookie(s.sessionCookieMaker.Name)
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

func (s *Server) githubClientFromSessionState(r *http.Request) (*github.Client, error) {
	state, err := s.getSessionState(r)
	if err != nil {
		return nil, err
	}
	clt := s.oauth2Config.Client(r.Context(), &state.OAuth2Token)
	return github.NewClient(clt), nil
}
