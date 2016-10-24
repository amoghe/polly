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
	lead = kingpin.Flag("team-lead", "Member is a team lead").Bool()
	skey = kingpin.Flag("ssh-pubkey", "SSH public key").String()
)

func main() {
	kingpin.Parse()
	client, _ := gerrit.NewClient(fmt.Sprintf("http://%s:%d", *gAddr, *gPort), nil)
	client.Authentication.SetDigestAuth(*adminUser, *adminPass)

	if *user == "" {
		log.Fatalf("User not specified")
	}

	groups := []string{"team-members"}
	if *lead {
		groups = append(groups, "team-leads")
	}

	user, _, err := client.Accounts.CreateAccount(*user, &gerrit.AccountInput{
		Name:         "test user",
		Email:        "test@user.com",
		HTTPPassword: *pass,
		Groups:       groups,
		SSHKey:       *skey,
	})
	if err != nil {
		log.Fatalln("Failed to create user:", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(user)
}
