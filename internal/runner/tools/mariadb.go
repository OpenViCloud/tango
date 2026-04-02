package tools

import (
	"context"

	infratools "tango/internal/infrastructure/tools"
)

type MariaDBExecutable = infratools.MariaDBExecutable

const (
	MariaDBExecutableDump   = infratools.MariaDBExecutableDump
	MariaDBExecutableClient = infratools.MariaDBExecutableClient
)

func GetMariaDBExecutable(version string, executable MariaDBExecutable, installDir string) (string, error) {
	return infratools.GetMariaDBExecutable(version, executable, installDir)
}

func VerifyMariaDBInstallation(installDir string) error {
	return infratools.VerifyMariaDBInstallation(installDir)
}

func DetectMariaDBVersion(ctx context.Context, cfg infratools.MariaDBConnectionConfig) (string, error) {
	return infratools.DetectMariaDBVersion(ctx, cfg)
}
