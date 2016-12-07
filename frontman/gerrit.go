package main

import (
	"encoding/json"
	"log"
	"net/http"

	goji "goji.io"

	"goji.io/pat"

	"github.com/alioygur/gores"
	"github.com/amoghe/polly/frontman/datastore"
	gerrit "github.com/andygrunwald/go-gerrit"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

const (
	localGerritURL = "http://127.0.0.1:10080"

	gerritAdminUsername = "admin"
	gerritAdminPassword = "supersecret"
)

type gerritRouter struct {
	mux            *goji.Mux
	db             *gorm.DB
	tokenExtractor TokenExtractor
}

// OrganizationExposure is how we expose a github organization
type OrganizationExposure struct {
	datastore.Organization
}

// NewGerritRouter returns a goji.Mux that handles routes pertaining to Gerrit config
func NewGerritRouter(db *gorm.DB, te TokenExtractor) http.Handler {
	g := gerritRouter{
		mux:            goji.SubMux(),
		db:             db,
		tokenExtractor: te,
	}
	g.mux.HandleFunc(pat.Post("/organizations"), g.ImportOrganization)
	g.mux.HandleFunc(pat.Post("/organizations/:org_name/repositories"), g.ImportRepository)
	g.mux.HandleFunc(pat.Get("/servers"), g.ListAvailableServers)
	return &g
}

// ServeHTTP allows gerritRouter to satisfy http.Handler
func (g *gerritRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

// ImportOrganization creates an organization in the polly db
func (g *gerritRouter) ImportOrganization(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	org := OrganizationExposure{datastore.Organization{}}
	if err := dec.Decode(&org); err != nil {
		handleJSONDecodeError(w, errors.Wrap(err, "invalid organization in request body"))
		return
	}

	org.Repositories = []datastore.Repository{} // blank out the association field
	err := datastore.InsertOrganization(g.db.Debug(), &org.Organization)
	if err != nil {
		handleGormError(w, errors.Wrap(err, "failed to insert organization in db"))
		return
	}

	// _, err = datastore.GetServerForOrganization(g.db.Debug(), org.Name)
	// if err == gorm.ErrRecordNotFound {
	// 	fmt.Fprintln(w, `{"error": "Your organization is not supported yet"}`)
	// 	return
	// }
	// if err != nil {
	// 	handleGormError(w, errors.Wrap(err, "failed to insert organization in db"))
	// 	return
	// }

	log.Println("Created org", org.Name, " using backend server:", org.Server.IPAddr)
	gores.JSON(w, http.StatusOK, org)
}

// ImportRepository creates a project in the polly db
func (g *gerritRouter) ImportRepository(w http.ResponseWriter, r *http.Request) {
	orgName := pat.Param(r, "org_name")
	if orgName == "" {
		handleMissingParam(w, errors.New("org name not specified"))
		return
	}

	org, err := datastore.GetOrganizationByName(g.db.Debug(), orgName)
	if err != nil {
		handleGormError(w, errors.Wrap(err, "failed to find organization"))
		return
	}

	dec := json.NewDecoder(r.Body)
	rep := datastore.Repository{OrganizationID: org.Name}
	if err = dec.Decode(&rep); err != nil {
		handleJSONDecodeError(w, errors.Wrap(err, "invalid repository in request body"))
		return
	}
	if err = datastore.InsertRepository(g.db.Debug(), &rep); err != nil {
		handleGormError(w, errors.Wrap(err, "failed to insert repository in db"))
		return
	}

	log.Println("Setting up gerrit server")
	gclt, err := gerrit.NewClient(localGerritURL, nil)
	if err != nil {
		handleGerritAPIError(w, errors.Wrap(err, "failed to setup client to gerrit server"))
		return
	}
	gclt.Authentication.SetDigestAuth(gerritAdminUsername, gerritAdminPassword)

	proj, resp, err := gclt.Projects.CreateProject(rep.Name, &gerrit.ProjectInput{
		Name:              rep.Name,
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
	gores.JSON(w, http.StatusOK, &rep)
}

func (g *gerritRouter) ListAvailableServers(w http.ResponseWriter, r *http.Request) {
	srvs, err := datastore.GetAvailableServer(g.db.Debug())
	if err != nil {
		handleGormError(w, err)
		return
	}
	gores.JSONIndent(w, http.StatusOK, srvs, "", "  ")
}
