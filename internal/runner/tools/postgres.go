package tools

import (
	"context"

	infratools "tango/internal/infrastructure/tools"
)

type PostgresExecutable = infratools.PostgresExecutable

const (
	PostgresExecutableDump    = infratools.PostgresExecutableDump
	PostgresExecutableRestore = infratools.PostgresExecutableRestore
)

func GetPostgresExecutable(version string, executable PostgresExecutable, installDir string) (string, error) {
	return infratools.GetPostgresExecutable(version, executable, installDir)
}

func VerifyPostgresInstallation(installDir string) error {
	return infratools.VerifyPostgresInstallation(installDir)
}

func DetectPostgresVersion(ctx context.Context, cfg infratools.PostgresConnectionConfig) (string, error) {
	return infratools.DetectPostgresVersion(ctx, cfg)
}
