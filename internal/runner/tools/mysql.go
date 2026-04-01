package tools

import (
	"context"

	infratools "tango/internal/infrastructure/tools"
)

type MySQLExecutable = infratools.MySQLExecutable

const (
	MySQLExecutableDump   = infratools.MySQLExecutableDump
	MySQLExecutableClient = infratools.MySQLExecutableClient
)

func GetMySQLExecutable(version string, executable MySQLExecutable, installDir string) (string, error) {
	return infratools.GetMySQLExecutable(version, executable, installDir)
}

func VerifyMySQLInstallation(installDir string) error {
	return infratools.VerifyMySQLInstallation(installDir)
}

func DetectMySQLVersion(ctx context.Context, cfg infratools.MySQLConnectionConfig) (string, error) {
	return infratools.DetectMySQLVersion(ctx, cfg)
}
