package main

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"

	goji "goji.io"

	"github.com/alioygur/gores"
	"github.com/amoghe/polly/frontman/datastore"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"goji.io/pat"
)

// TokenExtractor is a func that can "extract" a oauth2 Token from the http.Request for an authenticated session
type TokenExtractor func(r *http.Request) (*oauth2.Token, error)

// githubRouter is the mux that handles all github related endpoints
type githubRouter struct {
	mux            *goji.Mux
	tokenExtractor TokenExtractor
}

// NewGithubRouter returns a mux that is capable of handling all github related routes
func NewGithubRouter(te TokenExtractor) http.Handler {
	g := githubRouter{
		mux:            goji.SubMux(),
		tokenExtractor: te,
	}
	g.mux.HandleFunc(pat.Get("/organizations"), g.ListGithubOrganizations)
	g.mux.HandleFunc(pat.Get("/organizations/:org_name/repositories"), g.ListGithubRepositoriesForOrganization)
	return &g
}

// ServeHTTP allows githubRouter to satisfy http.Handler
func (g *githubRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clt, err := g.githubClientFromRequest(r)
	if err != nil {
		handleUnauthorized(w, "couldn't create github client for current user")
		return
	}
	ctx := context.WithValue(r.Context(), "github-client", clt)
	g.mux.ServeHTTP(w, r.WithContext(ctx))
}

// return a githubClient from the http request (if its from an authenticated user)
func (g *githubRouter) githubClientFromRequest(r *http.Request) (*github.Client, error) {
	token, err := g.tokenExtractor(r)
	if err != nil {
		return nil, err
	}
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token.AccessToken,
	})
	httpClient := oauth2.NewClient(r.Context(), tokenSource)
	return github.NewClient(httpClient), nil
}

// ListGithubOrganizations returns the authenticated users membership
func (g *githubRouter) ListGithubOrganizations(w http.ResponseWriter, r *http.Request) {
	client := r.Context().Value("github-client").(*github.Client)

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
func (g *githubRouter) ListGithubRepositoriesForOrganization(w http.ResponseWriter, r *http.Request) {
	client := r.Context().Value("github-client").(*github.Client)

	orgName := pat.Param(r, "org_name")
	if orgName == "" {
		handleMissingParam(w, errors.New("org name not specified"))
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
