package main

import (
	"net/http"

	goji "goji.io"

	"github.com/alioygur/gores"
	"github.com/amoghe/polly/frontman/datastore"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"goji.io/pat"
)

// GithubRoutes returns a goji.Mux that handles routes pertaining to Github data
func (s *Server) GithubRoutes() *goji.Mux {
	m := goji.SubMux()
	m.HandleFunc(pat.Get("/organizations"), s.ListGithubOrganizations)
	m.HandleFunc(pat.Get("/organizations/:org_name/repositories"), s.ListGithubRepositoriesForOrganization)
	return m
}

// ListGithubOrganizations returns the authenticated users membership
func (s *Server) ListGithubOrganizations(w http.ResponseWriter, req *http.Request) {
	client, err := s.githubClientFromSessionState(req)
	if err != nil {
		handleSessionExtractError(w, err)
		return
	}

	opt := github.ListOrgMembershipsOptions{State: "active"}
	mems, _, err := client.Organizations.ListOrgMemberships(&opt)
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

	client, err := s.githubClientFromSessionState(req)
	if err != nil {
		handleSessionExtractError(w, err)
		return
	}

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
