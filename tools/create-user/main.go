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

	user = kingpin.Flag("username", "Name of user to create").Required().String()
	pass = kingpin.Flag("password", "Password of user to create").Default("password").String()
)

func main() {
	kingpin.Parse()
	client, _ := gerrit.NewClient(fmt.Sprintf("http://%s:%d", *gAddr, *gPort), nil)

	if *user == "" {
		log.Fatalf("User not specified")
	}

	client.Authentication.SetDigestAuth(*adminUser, *adminPass)

	user, _, err := client.Accounts.CreateAccount(*user, &gerrit.AccountInput{
		Name:         "test user",
		Email:        "test@user.com",
		HTTPPassword: *pass,
		Groups:       []string{"team-members"},
	})
	if err != nil {
		log.Fatalln("Failed to create user:", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(user)
}
