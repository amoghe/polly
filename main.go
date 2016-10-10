package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/dghubble/gologin"
	"github.com/dghubble/gologin/internal"
	"github.com/dghubble/sessions"

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
		HTTPOnly: true, // FIXME
		Secure:   false,
	}

	stateCookieConfig = gologin.CookieConfig{
		Name:     "state-cookie",
		Path:     "/",
		MaxAge:   60,
		HTTPOnly: true, // FIXME
		Secure:   false,
	}
)

// GithubAppConfig holds the config for our Github app
type GithubAppConfig struct {
	GithubClientID     string
	GithubClientSecret string
}

type sessionCookieState struct {
	UserID    int
	AuthToken string
}

// Server represents the server
type Server struct {
	githubAppConfig     GithubAppConfig
	stateCookieConfig   gologin.CookieConfig
	sessionCookieConfig gologin.CookieConfig
	oauth2Config        oauth2.Config
	mux                 *http.ServeMux
}

// main creates and starts a Server listening.
func main() {
	var (
		listenAddress = "localhost:8080"

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
	scc := gologin.DebugOnlyCookieConfig
	scc.Name = "session-cookie"

	server := Server{
		githubAppConfig:     *config,
		stateCookieConfig:   stateCookieConfig,
		sessionCookieConfig: sessionCookieConfig,
		oauth2Config: oauth2.Config{
			ClientID:     config.GithubClientID,
			ClientSecret: config.GithubClientSecret,
			RedirectURL:  "http://localhost:8080/github/callback",
			Endpoint:     oauth2.Endpoint{AuthURL: GithubAuthURL, TokenURL: GithubTokenURL},
			Scopes:       []string{"read:public_key"},
		},
		mux: http.NewServeMux(),
	}

	// Setup routes
	server.mux.HandleFunc("/", server.HandleWelcome)
	server.mux.HandleFunc("/logout", server.HandleLogout)
	server.mux.HandleFunc("/profile", server.HandleProfile)
	server.mux.HandleFunc("/auth/github/login", server.HandleLogin)
	server.mux.HandleFunc("/github/callback", server.HandleCallback)

	return &server
}

// ServeHTTP allows Server to be a mux
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Serving req:", r.URL.String())
	s.mux.ServeHTTP(w, r)
}

// HandleWelcome shows a welcome message and login button.
func (s *Server) HandleWelcome(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	if s.isAuthenticated(req) {
		http.Redirect(w, req, "/profile", http.StatusFound)
		return
	}
	page, _ := ioutil.ReadFile("home.html")
	fmt.Fprintf(w, string(page))
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
	if s.isAuthenticated(req) {
		// already authenticated
		http.Redirect(w, req, "/", http.StatusFound)
		return
	}

	randomState := s.setRandomState(w)
	http.Redirect(w, req, s.oauth2Config.AuthCodeURL(randomState), http.StatusFound)
}

// HandleProfile displays the profile page
func (s *Server) HandleProfile(w http.ResponseWriter, req *http.Request) {
	sc, err := s.getSessionCookieState(req)
	if err != nil {
		handleError(w, err)
	}

	b, _ := json.MarshalIndent(sc, "", "  ")
	w.Write(b)
}

// HandleCallback handles the outh callback
func (s *Server) HandleCallback(w http.ResponseWriter, req *http.Request) {
	// compare the random state from the callback and the cookie

	if err := req.ParseForm(); err != nil {
		handleError(w, err)
	}

	// dump, err := httputil.DumpRequestOut(req, false)
	// if err != nil {
	// 	log.Println("Error dumping request")
	// 	handleError(w, err)
	// }
	// log.Println("CB REQ:", string(dump))

	randState1, err := s.getRandomState(req)
	if err != nil {
		handleError(w, err)
	}
	randState2 := req.Form.Get("state")
	if randState2 == "" {
		handleError(w, errors.New("oauth2 callback: Request missing code or state"))
	}
	if randState1 != randState2 {
		handleError(w, errors.New("oauth2 callback: Mismatched state. Please retry auth"))
	}

	authCode := req.Form.Get("code") // Github docs say this is the code

	token, err := s.oauth2Config.Exchange(req.Context(), authCode)
	if err != nil {
		handleError(w, err)
	}

	b, _ := json.MarshalIndent(token, "", "  ")
	log.Println("token:", string(b))
	s.setSessionCookieState(w, sessionCookieState{UserID: 0, AuthToken: token.AccessToken})

	// send them to their "homepage"
	http.Redirect(w, req, "/profile", http.StatusFound)
}

//
// - - - Helpers - - -
//

func handleError(w http.ResponseWriter, err error) {
	log.Println("ERR: ", err)
}

// isAuthenticated returns true if the user has a signed session cookie.
func (s *Server) isAuthenticated(req *http.Request) bool {
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
	log.Println("Setting cookie:", val)
	http.SetCookie(w, internal.NewCookie(s.stateCookieConfig, val))
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

// set the user ID and auth token in the session cookie
func (s *Server) setSessionCookieState(w http.ResponseWriter, sc sessionCookieState) error {
	val, err := json.Marshal(&sc)
	if err != nil {
		return err
	}
	b64 := base64.URLEncoding.EncodeToString(val)
	http.SetCookie(w, internal.NewCookie(s.sessionCookieConfig, b64))
	return nil
}

func (s *Server) getSessionCookieState(r *http.Request) (sc sessionCookieState, err error) {
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
