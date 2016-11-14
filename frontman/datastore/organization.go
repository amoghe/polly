package datastore

import "github.com/jinzhu/gorm"

// Organization is the representation of an organization in polly
type Organization struct {
	Name         string       `json:"name" gorm:"primary_key"`
	GithubID     int          `json:"github_id"`
	Repositories []Repository // has-many Repository
}

// InsertOrganization inserts the user into the database
func InsertOrganization(db *gorm.DB, org *Organization) error {
	return db.Debug().Create(org).Error
}

// GetOrganizationByName returns the user with the specified name
func GetOrganizationByName(db *gorm.DB, name string) (*Organization, error) {
	var org Organization
	err := db.Where("name = ?", name).First(&org).Error
	return &org, err
}
