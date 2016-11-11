package main

import (
	"net/http"

	"github.com/alioygur/gores"
	"github.com/google/go-github/github"
	"goji.io/pat"
)

// ListGithubOrganizations returns the authenticated users membership
func (s *Server) ListGithubOrganizations(w http.ResponseWriter, req *http.Request) {
	sc, err := s.getSessionState(req)
	if err != nil {
		handleError(w, err)
		return
	}

	client := s.newGithubClient(req.Context(), &sc.OAuth2Token)
	mems, _, err := client.Organizations.ListOrgMemberships(&github.ListOrgMembershipsOptions{State: "active"})
	if err != nil {
		handleError(w, err)
		return
	}

	// we'll repurpose github.Membership as our "Organization"
	gores.JSON(w, http.StatusOK, mems)
}

// ListGithubRepositoriesForOrganization lists repos for the specified org membership
func (s *Server) ListGithubRepositoriesForOrganization(w http.ResponseWriter, req *http.Request) {
	orgID := pat.Param(req, "org_id")

	sc, err := s.getSessionState(req)
	if err != nil {
		handleError(w, err)
		return
	}
	client := s.newGithubClient(req.Context(), &sc.OAuth2Token)

	repos, _, err := client.Repositories.ListByOrg(orgID, nil)
	if err != nil {
		handleError(w, err)
		return
	}

	// we'll repurpose github.Repository as our "Repository"
	gores.JSON(w, http.StatusOK, repos)
}
