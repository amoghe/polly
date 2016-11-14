package datastore

import "github.com/jinzhu/gorm"

// OpenDatabase opens a connection to the db wrapped by gorm
func OpenDatabase(dbtype, dsn string) (*gorm.DB, error) {
	return gorm.Open(dbtype, dsn)
}

// MigrateDatabase runs migrations on the database
func MigrateDatabase(db *gorm.DB) error {
	return db.AutoMigrate(&User{},
		&Organization{},
		&Repository{}).Error
}
