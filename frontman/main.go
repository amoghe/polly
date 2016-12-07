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

	_ "github.com/mattn/go-sqlite3"
)

var (
	// sessionStore encodes and decodes session data stored in signed cookies
	sessionStore = sessions.NewCookieStore([]byte("pollySecret"), nil)
)

// Server represents the server
type Server struct {
	db  *gorm.DB
	mux *goji.Mux
}

// returns the first non empty string in the specfied array, fatals if all are empty
func oneOfOrDie(opts []string) string {
	for _, s := range opts {
		if s != "" {
			return s
		}
	}
	log.Fatal("Missing param")
	return ""
}

// main creates and starts a Server listening.
func main() {
	var (
		listenAddress = "0.0.0.0:8080"
		// github
		clientID     = flag.String("client-id", "", "Github Client ID")
		clientSecret = flag.String("client-secret", "", "Github Client Secret")
		// database
		dbType  = flag.String("db-type", "sqlite3", "Type of database")
		dbDSN   = flag.String("db-dsn", "/tmp/polly", "Database DSN")
		orgName = flag.String("github-org-name", "", "Github org  name")
		// gerrit
		gerritAddr      = flag.String("gerrit-addr", "localhost:10080", "Address of gerrit server")
		gerritAdminUser = flag.String("gerrit-admin-user", "admin", "Admin user (gerrit)")
		gerritAdminPass = flag.String("gerrit-admin-pass", "supersecret", "Admin pass (gerrit)")
		// cfg structs
		githubConfig = GithubAppConfig{
			GithubClientID:     oneOfOrDie([]string{*clientID, os.Getenv("GITHUB_CLIENT_ID")}),
			GithubClientSecret: oneOfOrDie([]string{*clientSecret, os.Getenv("GITHUB_CLIENT_SECRET")}),
		}
		gerritConfig = gerritConfig{
			Addr:     *gerritAddr,
			Username: *gerritAdminUser,
			Password: *gerritAdminPass,
		}
	)

	// allow consumer credential flags to override config fields
	flag.Parse()

	if len(*orgName) <= 0 {
		log.Fatal("Missing Github org name")
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
	err = http.ListenAndServe(listenAddress, NewServer(githubConfig, gerritConfig, db))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

	log.Println("Frontman exiting")
}

// NewServer returns a new ServeMux with app routes.
func NewServer(githubCfg GithubAppConfig, gerritCfg gerritConfig, db *gorm.DB) *Server {
	var (
		mux          = goji.NewMux()
		authRouter   = NewAuthRouter(githubCfg)
		githubRouter = NewGithubRouter(authRouter.AuthTokenFromRequest)
		gerritRouter = NewGerritRouter(db, gerritCfg, authRouter.AuthTokenFromRequest)
	)

	mux.Handle(pat.New("/auth/*"), authRouter)     // Auth routes
	mux.Handle(pat.New("/github/*"), githubRouter) // Github  routes
	mux.Handle(pat.New("/gerrit/*"), gerritRouter) // Gerrit routes

	return &Server{
		mux: mux,
		db:  db,
	}
}

// ServeHTTP allows Server to be a mux
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//log.Println("Serving req:", r.URL.String())
	s.mux.ServeHTTP(w, r)
}
