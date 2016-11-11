package main

import (
	"encoding/json"
	"log"
	"net/http"

	gerrit "github.com/andygrunwald/go-gerrit"
	"github.com/pkg/errors"
)

const (
	localGerritURL = "http://127.0.0.1"

	gerritAdminUsername = "admin"
	gerritAdminPassword = "supersecret"
)

type ImportRepository struct {
	Name string `json:"name"`
}

// CreateGerritRepository creates a project in the gerrit server
func (s *Server) CreateGerritRepository(w http.ResponseWriter, req *http.Request) {
	dec := json.NewDecoder(req.Body)
	imp := ImportRepository{}
	if err := dec.Decode(&imp); err != nil {
		handleError(w, err)
		return
	}

	gclt, err := gerrit.NewClient(localGerritURL, nil)
	if err != nil {
		handleError(w, errors.Wrap(err, "failed to setup client to gerrit server"))
		return
	}
	gclt.Authentication.SetDigestAuth(gerritAdminUsername, gerritAdminPassword)

	proj, resp, err := gclt.Projects.CreateProject(imp.Name, &gerrit.ProjectInput{
		Name:              imp.Name,
		CreateEmptyCommit: false,
	})
	if err != nil {
		handleError(w, errors.Wrap(err, "failed to create project"))
		return
	}
	if resp.StatusCode != http.StatusOK {
		handleError(w, errors.Errorf("incorrect response code from server (%d)", resp.StatusCode))
		return
	}
	log.Println("Created project", proj.Name)

}
