package rds

import (
	"errors"
	"time"
)

var (
	ErrInvalidDBIdentifier  = errors.New("DB instance identifier cannot be empty")
	ErrInvalidEngine        = errors.New("engine cannot be empty")
	ErrInvalidInstanceClass = errors.New("instance class cannot be empty")
	ErrInvalidStorage       = errors.New("allocated storage must be at least 20 GB")
	ErrInvalidMasterUser    = errors.New("master username cannot be empty")
	ErrInvalidPassword      = errors.New("master password must be provided")
)

// DBInstance represents an RDS database instance in the domain model
type DBInstance struct {
	// Identification
	DBInstanceIdentifier string
	DBInstanceArn        string

	// Engine configuration
	Engine        string
	EngineVersion string

	// Compute and storage
	DBInstanceClass  string
	AllocatedStorage int32

	// Authentication
	MasterUsername string
	MasterPassword string

	// Database
	DBName string
	Port   int32

	// Network and access
	Endpoint           string
	MultiAZ            bool
	PubliclyAccessible bool

	// Security
	StorageEncrypted bool

	// Backup
	BackupRetentionPeriod int32
	PreferredBackupWindow string

	// Deletion
	SkipFinalSnapshot bool
	DeletionPolicy    string

	// Tags
	Tags map[string]string

	// State
	Status       string
	LastSyncTime *time.Time
}

// Validate checks if the DB instance configuration is valid
func (db *DBInstance) Validate() error {
	if db.DBInstanceIdentifier == "" {
		return ErrInvalidDBIdentifier
	}

	if db.Engine == "" {
		return ErrInvalidEngine
	}

	if db.DBInstanceClass == "" {
		return ErrInvalidInstanceClass
	}

	if db.AllocatedStorage < 20 {
		return ErrInvalidStorage
	}

	if db.MasterUsername == "" {
		return ErrInvalidMasterUser
	}

	if db.MasterPassword == "" {
		return ErrInvalidPassword
	}

	// Validate backup retention period (0-35 days)
	if db.BackupRetentionPeriod < 0 || db.BackupRetentionPeriod > 35 {
		return errors.New("backup retention period must be between 0 and 35 days")
	}

	return nil
}

// IsAvailable checks if the DB instance is available
func (db *DBInstance) IsAvailable() bool {
	return db.Status == "available"
}

// SetDefaults sets default values for optional fields
func (db *DBInstance) SetDefaults() {
	if db.Port == 0 {
		// Set default port based on engine
		switch db.Engine {
		case "postgres":
			db.Port = 5432
		case "mysql", "mariadb":
			db.Port = 3306
		case "sqlserver-ex", "sqlserver-web", "sqlserver-se", "sqlserver-ee":
			db.Port = 1433
		case "oracle-se2", "oracle-ee":
			db.Port = 1521
		default:
			db.Port = 3306
		}
	}

	if db.BackupRetentionPeriod == 0 {
		db.BackupRetentionPeriod = 7 // 7 days default
	}

	if db.DeletionPolicy == "" {
		db.DeletionPolicy = "Delete"
	}
}
