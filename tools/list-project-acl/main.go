package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/andygrunwald/go-gerrit"
)

var (
	gAddr = kingpin.Flag("gerrit-addr", "Gerrit address.").Default("127.0.0.1").IP()
	gPort = kingpin.Flag("gerrit-port", "Gerrit port").Default("8080").Int()

	adminUser = kingpin.Flag("admin-user", "Admin username").Default("admin").String()
	adminPass = kingpin.Flag("admin-pass", "Admin password").Default("supersecret").String()

	name = kingpin.Flag("username", "Name of project to display").Required().String()
)

func main() {
	kingpin.Parse()
	client, _ := gerrit.NewClient(fmt.Sprintf("http://%s:%d", *gAddr, *gPort), nil)
	client.Authentication.SetDigestAuth(*adminUser, *adminPass)

	pi, _, err := client.Access.ListAccessRights(&gerrit.ListAccessRightsOptions{
		Project: []string{*name},
	})
	if err != nil {
		log.Fatalln("Failed to list access rights for project:", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(pi)
}
