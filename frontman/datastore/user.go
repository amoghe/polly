package datastore

import "github.com/jinzhu/gorm"

// Models

// User represents a user
type User struct {
	Username string `gorm:"primary_key"`
	Password string
	GithubID int
}

// InsertUser inserts the user into the database
func InsertUser(db *gorm.DB, user *User) error {
	return db.Debug().Create(user).Error
}

// GetUser returns the user with the specified githubID
func FindUser(db *gorm.DB, githubID int) (*User, error) {
	var user User
	err := db.Find(&user, githubID).Error
	return &user, err
}
