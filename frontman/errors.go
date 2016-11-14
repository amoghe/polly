package main

import (
	"log"
	"net/http"

	"github.com/alioygur/gores"
	"github.com/google/go-github/github"
	"github.com/jinzhu/gorm"
)

type errorResponseBody struct {
	Error string `json:"error"`
}

func handleUnauthorized(w http.ResponseWriter, msg string) {
	log.Println("Unauthorized (oauth2 callback):", msg)
	gores.JSON(w, http.StatusUnauthorized, errorResponseBody{Error: msg})
	return
}

func handleSessionExtractError(w http.ResponseWriter, err error) {
	msg := "failed to extract auth data from session"
	if err == http.ErrNoCookie {
		msg = "missing auth cookie"
	}
	log.Println(msg)
	gores.JSON(w, http.StatusUnauthorized, errorResponseBody{Error: msg})
}

func handleJSONDecodeError(w http.ResponseWriter, err error) {
	log.Println("Failed to decode JSON:", err)
	gores.JSON(w, http.StatusBadRequest, errorResponseBody{Error: err.Error()})
}

func handleMissingParam(w http.ResponseWriter, err error) {
	log.Println("Missing parameter in request:", err)
	gores.JSON(w, http.StatusBadRequest, errorResponseBody{Error: err.Error()})
}

func handleGormError(w http.ResponseWriter, err error) {
	retcode := http.StatusNotFound
	if err == gorm.ErrRecordNotFound {
		retcode = http.StatusBadRequest
	}
	log.Println("Error from gorm:", err)
	gores.JSON(w, retcode, errorResponseBody{Error: err.Error()})
}

func handleGithubAPIError(w http.ResponseWriter, err error) {
	retcode := http.StatusBadGateway // bad upstream gateway?
	retmesg := err.Error()
	if e, ok := err.(*github.ErrorResponse); ok {
		retcode = e.Response.StatusCode
		retmesg = e.Message
	}
	gores.JSON(w, retcode, struct{ Error string }{Error: retmesg})
}

// private
func _handleError(w http.ResponseWriter, err error) {
	log.Printf("ERR: (%T) %s\n", err, err)
	gores.JSON(w, http.StatusBadRequest, errorResponseBody{Error: err.Error()})
}
