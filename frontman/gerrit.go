package main

import (
	"log"
	"net/http"

	goji "goji.io"

	"goji.io/pat"

	"github.com/alioygur/gores"
	gerrit "github.com/andygrunwald/go-gerrit"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type gerritRouter struct {
	cfg            GerritConfig
	mux            *goji.Mux
	db             *gorm.DB
	tokenExtractor TokenExtractor
}

// GerritConfig holds the settings of the backing gerrit server
type GerritConfig struct {
	Addr     string
	Username string
	Password string
}

// NewGerritRouter returns a goji.Mux that handles routes pertaining to Gerrit config
func NewGerritRouter(db *gorm.DB, cfg GerritConfig, te TokenExtractor) http.Handler {
	g := gerritRouter{
		cfg:            cfg,
		mux:            goji.SubMux(),
		db:             db,
		tokenExtractor: te,
	}
	g.mux.HandleFunc(pat.Put("/repositories/:name"), g.ImportRepository)
	return &g
}

// ServeHTTP allows gerritRouter to satisfy http.Handler
func (g *gerritRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

// ImportRepository creates a project in the polly db and imports the repo into gerrit
func (g *gerritRouter) ImportRepository(w http.ResponseWriter, r *http.Request) {
	repoName := pat.Param(r, "name")
	if repoName == "" {
		handleMissingParam(w, errors.New("repository name not specified"))
		return
	}

	log.Println("Setting up gerrit server")
	gclt, err := gerrit.NewClient(g.cfg.Addr, nil)
	if err != nil {
		handleGerritAPIError(w, errors.Wrap(err, "failed to setup client to gerrit server"))
		return
	}
	gclt.Authentication.SetDigestAuth(g.cfg.Username, g.cfg.Password)

	proj, resp, err := gclt.Projects.CreateProject(repoName, &gerrit.ProjectInput{
		Name:              repoName,
		CreateEmptyCommit: false,
	})
	if err != nil {
		handleGerritAPIError(w, errors.Wrap(err, "failed to create project in gerrit"))
		return
	}
	if resp.StatusCode != http.StatusOK {
		handleGerritAPIError(w, errors.Errorf("incorrect response code from gerrit (%d)", resp.StatusCode))
		return
	}
	token, _ := g.tokenExtractor(r)
	log.Println("Created project", proj.Name)
	log.Println("Import the repo using token:", token.AccessToken)
	gores.JSON(w, http.StatusOK, nil)
}
