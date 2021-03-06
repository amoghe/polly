package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"

	goji "goji.io"

	"goji.io/pat"
)

var (
	// GithubAuthURL is the endpoints that github exposes for oauth2
	GithubAuthURL = "https://github.com/login/oauth/authorize"
	// GithubTokenURL is the URL at which we get the token
	GithubTokenURL = "https://github.com/login/oauth/access_token"
	// DefaultScopes represents the minimum scope we need to operate
	DefaultScopes = []github.Scope{github.ScopeReadPublicKey, github.ScopeReadOrg}
)

const (
	// RouteLogin is used for Github OAuth2 login flow
	RouteLogin = "/github/login"
	// RouteCallback is used for the Github OAuth2 callback
	RouteCallback = "/github/callback"
	// RouteBackdoor is used to set session cookie for a given PAT
	RouteBackdoor = "/github/backdoor"
	// RouteVerify is used to verify the is the token in the session is ok
	RouteVerify = "/github/verify"
	// RouteLogout is the route to logout
	RouteLogout = "/logout"
)

// GithubConfig holds the config for our Github app
type GithubConfig struct {
	OrgName  string
	ClientID string
	Secret   string
}

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
	githubConfig       GithubConfig
	stateCookieMaker   CookieMaker
	sessionCookieMaker CookieMaker
}

// NewAuthRouter returns a http.Handler that handles routes pertaining to authentication
func NewAuthRouter(githubCfg GithubConfig) AuthenticatingRouter {
	scopes := make([]string, len(DefaultScopes))
	for _, scope := range DefaultScopes {
		scopes = append(scopes, string(scope))
	}
	oauth2Cfg := oauth2.Config{
		ClientID:     githubCfg.ClientID,
		ClientSecret: githubCfg.Secret,
		RedirectURL:  "http://localhost:8080/auth" + RouteCallback,
		Endpoint:     oauth2.Endpoint{AuthURL: GithubAuthURL, TokenURL: GithubTokenURL},
		Scopes:       scopes,
	}
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
	a.mux.HandleFunc(pat.Get(RouteBackdoor), a.HandleBackdoor)
	a.mux.HandleFunc(pat.Get(RouteVerify), a.HandleVerify)
	return &a
}

// ServeHTTP allows authRouter satisfy the http.Handler interface
func (a *authRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

// sessionState is what we store in the session to keep track of the user
type sessionState struct {
	//UserID      int
	OAuth2Token oauth2.Token
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
		randomState := a.setRandomState(w)
		http.Redirect(w, r, a.oauth2Config.AuthCodeURL(randomState), http.StatusFound)
		return
	}

	auth, err := a.VerifyAuthToken(state.OAuth2Token)
	if err != nil {
		randomState := a.setRandomState(w)
		http.Redirect(w, r, a.oauth2Config.AuthCodeURL(randomState), http.StatusFound)
		return
	}
	log.Println("auth is:", auth)

	// TODO: check auth.Scopes for sufficient permissions

	// TODO: where to redirect when user is already authenticated?
	http.Redirect(w, r, "/", http.StatusFound)
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

	_, err = a.VerifyAuthToken(*token)
	if err != nil {
		handleUnauthorized(w, err.Error())
		return
	}

	state := sessionState{OAuth2Token: *token}
	a.setSessionState(w, state)

	// TODO: send them to their "homepage"
	http.Redirect(w, r, "/github/organizations", http.StatusFound)
}

//HandleBackdoor sets the session cookie for the given username+password (provided via BasicAuth)
func (a *authRouter) HandleBackdoor(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok {
		handleMissingParam(w, errors.New("missing auth credentials"))
		return
	}
	if len(user) <= 0 || len(pass) <= 0 {
		handleMissingParam(w, errors.New("username/password too short"))
		return
	}

	log.Println("[BACKDOOR] Attempting to create authorization for", user)
	authNote := "polly-backdoor"
	baTransp := github.BasicAuthTransport{
		Username: user,
		Password: pass,
	}
	ghClient := github.NewClient(baTransp.Client())
	auth, _, err := ghClient.Authorizations.Create(&github.AuthorizationRequest{
		Scopes:       DefaultScopes,
		Note:         &authNote,
		ClientID:     &a.githubConfig.ClientID,
		ClientSecret: &a.githubConfig.Secret,
	})
	if err != nil {
		log.Println("[BACKDOOR] api error:", err)
		handleGithubAPIError(w, err)
		return
	}

	token := oauth2.Token{AccessToken: *auth.Token}
	_, err = a.VerifyAuthToken(token)
	if err != nil {
		log.Println("[BACKDOOR] token verification error:", err)
		handleGithubAPIError(w, err)
		return
	}

	state := sessionState{OAuth2Token: token}
	a.setSessionState(w, state)
	log.Printf("[BACKDOOR] Issued session for user %s", user)
}

// HandleVerify verifies whether the token in the session associated with the request is valid
func (a *authRouter) HandleVerify(w http.ResponseWriter, r *http.Request) {
	tok, err := a.AuthTokenFromRequest(r)
	if err != nil {
		handleUnauthorized(w, "missing auth token or session")
		return
	}
	auth, err := a.VerifyAuthToken(*tok)
	if err != nil {
		handleUnauthorized(w, fmt.Sprintf("failed to verify token: %s", err.Error()))
		return
	}
	log.Println("[BACKDOOR] verified token (last 8)", *auth.TokenLastEight)
	w.WriteHeader(http.StatusOK)
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

// VerifyAuthToken verifies the given OAuth2 token.
func (a *authRouter) VerifyAuthToken(tok oauth2.Token) (*github.Authorization, error) {
	// Use the app creds to verify that the token is valid and has the needed scopes
	client := github.NewClient(&http.Client{
		Transport: &github.BasicAuthTransport{
			Username: a.githubConfig.ClientID,
			Password: a.githubConfig.Secret,
		},
	})
	auth, _, err := client.Authorizations.Check(a.githubConfig.ClientID, tok.AccessToken)
	if err != nil {
		return nil, err
	}
	scopeSet := map[github.Scope]bool{}
	for _, scope := range auth.Scopes {
		scopeSet[scope] = true
	}
	for _, scope := range DefaultScopes {
		if _, there := scopeSet[scope]; !there {
			return nil, errors.Errorf("auth token is missing scope: %s", scope)
		}
	}

	// Now use a new client (using users the token) to check membership to the org
	tc := oauth2.NewClient(oauth2.NoContext, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tok.AccessToken}))
	client = github.NewClient(tc)
	mem, _, err := client.Organizations.GetOrgMembership("", a.githubConfig.OrgName)
	if err != nil {
		return nil, err
	}
	log.Println("mem:", mem)
	if *mem.State != "active" {
		return nil, errors.New("not an active member of organization")
	}

	return auth, nil
}
