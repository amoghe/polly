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

	project = kingpin.Flag("proj-name", "Name of project to create").Required().String()
	descrip = kingpin.Flag("proj-desc", "Description of project").String()
)

func main() {
	kingpin.Parse()
	client, _ := gerrit.NewClient(fmt.Sprintf("http://%s:%d", *gAddr, *gPort), nil)

	if *project == "" {
		log.Fatalf("User not specified")
	}
	if *descrip == "" {
		*descrip = *project
	}

	client.Authentication.SetDigestAuth(*adminUser, *adminPass)

	proj, _, err := client.Projects.CreateProject(*project, &gerrit.ProjectInput{
		Owners:      []string{"team-leads"},
		Description: *descrip,
	})
	if err != nil {
		log.Fatalln("Failed to create project:", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(proj)
}
