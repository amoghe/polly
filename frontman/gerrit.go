package main

import (
	"encoding/json"
	"log"
	"net/http"

	"goji.io/pat"

	"github.com/alioygur/gores"
	"github.com/amoghe/polly/frontman/datastore"
	"github.com/pkg/errors"
)

const (
	localGerritURL = "http://127.0.0.1"

	gerritAdminUsername = "admin"
	gerritAdminPassword = "supersecret"
)

// CreatePollyOrganization creates an organization in the polly db
func (s *Server) CreatePollyOrganization(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	org := datastore.Organization{}
	if err := dec.Decode(&org); err != nil {
		handleJSONDecodeError(w, errors.Wrap(err, "invalid organization in request body"))
		return
	}

	org.Repositories = []datastore.Repository{} // blank out the association field
	err := datastore.InsertOrganization(s.db, &org)
	if err != nil {
		handleGormError(w, errors.Wrap(err, "failed to insert organization in db"))
		return
	}

	gores.JSON(w, http.StatusOK, org)
}

// CreatePollyRepository creates a project in the polly db
func (s *Server) CreatePollyRepository(w http.ResponseWriter, r *http.Request) {
	orgName := pat.Param(r, "org_name")
	if orgName == "" {
		handleMissingParam(w, errors.New("org name not specified"))
		return
	}

	org, err := datastore.GetOrganizationByName(s.db.Debug(), orgName)
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
	if err = datastore.InsertRepository(s.db.Debug(), &rep); err != nil {
		handleGormError(w, errors.Wrap(err, "failed to insert repository in db"))
		return
	}

	log.Println("AAA early return")
	//
	// gclt, err := gerrit.NewClient(localGerritURL, nil)
	// if err != nil {
	// 	handleError(w, errors.Wrap(err, "failed to setup client to gerrit server"))
	// 	return
	// }
	// gclt.Authentication.SetDigestAuth(gerritAdminUsername, gerritAdminPassword)
	//
	// proj, resp, err := gclt.Projects.CreateProject(rep.Name, &gerrit.ProjectInput{
	// 	Name:              rep.Name,
	// 	CreateEmptyCommit: false,
	// })
	// if err != nil {
	// 	handleError(w, errors.Wrap(err, "failed to create project in gerrit"))
	// 	return
	// }
	// if resp.StatusCode != http.StatusOK {
	// 	handleError(w, errors.Errorf("incorrect response code from gerrit (%d)", resp.StatusCode))
	// 	return
	// }
	// log.Println("Created project", proj.Name)
	gores.JSON(w, http.StatusOK, &rep)
}
