package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"goji.io/pat"

	goji "goji.io"

	"github.com/amoghe/polly/frontman/datastore"
	"github.com/dghubble/sessions"
	"github.com/jinzhu/gorm"
	"golang.org/x/oauth2"

	_ "github.com/mattn/go-sqlite3"
)

var (
	// GithubAuthURL is the endpoints that github exposes for oauth2
	GithubAuthURL = "https://github.com/login/oauth/authorize"
	// GithubTokenURL is the URL at which we get the token
	GithubTokenURL = "https://github.com/login/oauth/access_token"

	// sessionStore encodes and decodes session data stored in signed cookies
	sessionStore = sessions.NewCookieStore([]byte("pollySecret"), nil)
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
	githubAppConfig GithubAppConfig
	oauth2Config    oauth2.Config
	db              *gorm.DB
	mux             *goji.Mux
}

// main creates and starts a Server listening.
func main() {
	var (
		listenAddress = "0.0.0.0:8080"

		config = &GithubAppConfig{
			GithubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
			GithubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		}
		clientID     = flag.String("client-id", "", "Github Client ID")
		clientSecret = flag.String("client-secret", "", "Github Client Secret")
		dbType       = flag.String("db-type", "sqlite3", "Type of database")
		dbDSN        = flag.String("db-dsn", "/tmp/polly", "Database DSN")
	)
	// allow consumer credential flags to override config fields
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

	log.Println("Connecting to db", *dbType, "at", *dbDSN)
	db, err := datastore.OpenDatabase(*dbType, *dbDSN)
	if err != nil {
		log.Fatal("Failed to open db handle: ", err)
	}

	err = datastore.MigrateDatabase(db)
	if err != nil {
		log.Fatal("Failed to migrate db: ", err)
	}

	log.Println("Starting Server listening on:", listenAddress)
	err = http.ListenAndServe(listenAddress, NewServer(config, db))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

	log.Println("Frontman exiting")
}

// NewServer returns a new ServeMux with app routes.
func NewServer(githubCfg *GithubAppConfig, db *gorm.DB) *Server {

	server := &Server{
		githubAppConfig: *githubCfg,
		oauth2Config: oauth2.Config{
			ClientID:     githubCfg.GithubClientID,
			ClientSecret: githubCfg.GithubClientSecret,
			RedirectURL:  "http://localhost:8080/auth" + RouteCallback,
			Endpoint: oauth2.Endpoint{
				AuthURL:  GithubAuthURL,
				TokenURL: GithubTokenURL},
			Scopes: []string{
				"read:public_key",
				"read:org",
			},
		},
		mux: goji.NewMux(),
		db:  db,
	}

	authenticator := NewAuthRouter(server.githubAppConfig, server.oauth2Config)

	server.mux.Handle(pat.New("/auth/*"), authenticator)                                         // Auth routes
	server.mux.Handle(pat.New("/github/*"), NewGithubRouter(authenticator.AuthTokenFromRequest)) // Github  routes
	server.mux.Handle(pat.New("/gerrit/*"), NewGerritRouter(db))                                 // Gerrit routes
	return server
}

// ServeHTTP allows Server to be a mux
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//log.Println("Serving req:", r.URL.String())
	s.mux.ServeHTTP(w, r)
}
