package main

import (
	"net/http"

	"github.com/alioygur/gores"
	"github.com/amoghe/polly/frontman/datastore"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"goji.io/pat"
)

// ListGithubOrganizations returns the authenticated users membership
func (s *Server) ListGithubOrganizations(w http.ResponseWriter, req *http.Request) {
	sc, err := s.getSessionState(req)
	if err != nil {
		handleSessionExtractError(w, err)
		return
	}

	clt := s.newGithubClient(req.Context(), &sc.OAuth2Token)
	opt := github.ListOrgMembershipsOptions{State: "active"}
	mems, _, err := clt.Organizations.ListOrgMemberships(&opt)
	if err != nil {
		handleGithubAPIError(w, err)
		return
	}

	// we'll repurpose github.Membership as our "Organization"
	gores.JSON(w, http.StatusOK, mems)
}

// ListGithubRepositoriesForOrganization lists repos for a given org membership
func (s *Server) ListGithubRepositoriesForOrganization(w http.ResponseWriter, req *http.Request) {
	orgName := pat.Param(req, "org_name")
	if orgName == "" {
		handleMissingParam(w, errors.New("org name not specified"))
		return
	}

	sc, err := s.getSessionState(req)
	if err != nil {
		handleSessionExtractError(w, err)
		return
	}
	client := s.newGithubClient(req.Context(), &sc.OAuth2Token)

	repos, _, err := client.Repositories.ListByOrg(orgName, nil)
	if err != nil {
		handleGithubAPIError(w, err)
		return
	}

	ret := []datastore.Repository{}
	for _, repo := range repos {
		ret = append(ret, datastore.Repository{
			Name:     *repo.Name,
			GithubID: *repo.ID,
		})
	}

	gores.JSON(w, http.StatusOK, ret)
}
