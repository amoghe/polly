package datastore

import (
	"log"
	"os"
	"testing"

	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"
)

func newInMemoeryDB() *gorm.DB {
	db, err := OpenDatabase("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Panicln("Failed to create in-memory db:", err)
	}
	db.SetLogger(log.New(os.Stdout, "\n", 0))
	if err = MigrateDatabase(db); err != nil {
		log.Panicln("Failed to migrate db:", err)
	}
	return db
}

func TestUpsertUser(t *testing.T) {
	db := newInMemoeryDB()

	user1 := User{
		Username: "foobar",
		GithubID: 1234,
	}

	err := UpsertUser(db, &user1)
	if err != nil {
		t.Errorf("failed to upsert user (err: %v): %v", err, user1)
	}
}
