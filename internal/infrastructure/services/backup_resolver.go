package services

import (
	"fmt"

	appservices "tango/internal/application/services"
	"tango/internal/domain"
)

type backupStrategyResolver struct {
	mysql    appservices.BackupStrategy
	mariadb  appservices.BackupStrategy
	postgres appservices.BackupStrategy
	mongo    appservices.BackupStrategy
}

type restoreStrategyResolver struct {
	mysql    appservices.RestoreStrategy
	mariadb  appservices.RestoreStrategy
	postgres appservices.RestoreStrategy
	mongo    appservices.RestoreStrategy
}

type storageDriverResolver struct {
	local appservices.StorageDriver
}

func NewBackupStrategyResolver(mysql appservices.BackupStrategy, mariadb appservices.BackupStrategy, postgres appservices.BackupStrategy, mongo appservices.BackupStrategy) appservices.BackupStrategyResolver {
	return &backupStrategyResolver{mysql: mysql, mariadb: mariadb, postgres: postgres, mongo: mongo}
}

func NewStorageDriverResolver(local appservices.StorageDriver) appservices.StorageDriverResolver {
	return &storageDriverResolver{local: local}
}

func NewRestoreStrategyResolver(mysql appservices.RestoreStrategy, mariadb appservices.RestoreStrategy, postgres appservices.RestoreStrategy, mongo appservices.RestoreStrategy) appservices.RestoreStrategyResolver {
	return &restoreStrategyResolver{mysql: mysql, mariadb: mariadb, postgres: postgres, mongo: mongo}
}

func (r *backupStrategyResolver) Resolve(dbType domain.DatabaseType, method domain.BackupMethod) (appservices.BackupStrategy, error) {
	if dbType == domain.DatabaseTypeMySQL && method == domain.BackupMethodLogicalDump {
		return r.mysql, nil
	}
	if dbType == domain.DatabaseTypeMariaDB && method == domain.BackupMethodLogicalDump {
		return r.mariadb, nil
	}
	if dbType == domain.DatabaseTypePostgres && method == domain.BackupMethodLogicalDump {
		return r.postgres, nil
	}
	if dbType == domain.DatabaseTypeMongoDB && method == domain.BackupMethodLogicalDump {
		return r.mongo, nil
	}
	return nil, fmt.Errorf("backup strategy not implemented for %s/%s", dbType, method)
}

func (r *storageDriverResolver) Resolve(storageType domain.StorageType) (appservices.StorageDriver, error) {
	if storageType == domain.StorageTypeLocal {
		return r.local, nil
	}
	return nil, fmt.Errorf("storage driver not implemented for %s", storageType)
}

func (r *restoreStrategyResolver) Resolve(dbType domain.DatabaseType, method domain.BackupMethod) (appservices.RestoreStrategy, error) {
	if dbType == domain.DatabaseTypeMySQL && method == domain.BackupMethodLogicalDump {
		return r.mysql, nil
	}
	if dbType == domain.DatabaseTypeMariaDB && method == domain.BackupMethodLogicalDump {
		return r.mariadb, nil
	}
	if dbType == domain.DatabaseTypePostgres && method == domain.BackupMethodLogicalDump {
		return r.postgres, nil
	}
	if dbType == domain.DatabaseTypeMongoDB && method == domain.BackupMethodLogicalDump {
		return r.mongo, nil
	}
	return nil, fmt.Errorf("restore strategy not implemented for %s/%s", dbType, method)
}
