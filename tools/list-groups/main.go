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
)

func main() {
	kingpin.Parse()
	client, _ := gerrit.NewClient(fmt.Sprintf("http://%s:%d", *gAddr, *gPort), nil)
	client.Authentication.SetDigestAuth(*adminUser, *adminPass)

	groups, _, err := client.Groups.ListGroups(&gerrit.ListGroupsOptions{})
	if err != nil {
		log.Fatalln("Failed to create user:", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(groups)
}
