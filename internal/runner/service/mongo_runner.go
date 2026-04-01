package service

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"tango/internal/runner/model"
	runnertools "tango/internal/runner/tools"
)

type MongoRunner struct {
	toolsDir string
}

func NewMongoRunner(toolsDir string) *MongoRunner {
	return &MongoRunner{toolsDir: strings.TrimSpace(toolsDir)}
}

func (r *MongoRunner) RunLogicalDump(ctx context.Context, req *model.MongoLogicalDumpRequest, writer io.Writer) (string, error) {
	if req == nil {
		return "", fmt.Errorf("mongo dump request is required")
	}
	uri, err := runnertools.BuildMongoURI(req.Host, req.Port, req.Username, req.Password, req.AuthDatabase, req.ConnectionURI)
	if err != nil {
		return "", err
	}
	binPath, err := runnertools.GetMongoExecutable(r.toolsDir, "mongodump")
	if err != nil {
		return "", err
	}
	args := []string{
		"--uri=" + uri,
		"--db=" + req.Database,
		"--archive",
	}
	if strings.EqualFold(req.CompressionType, "gzip") {
		args = append(args, "--gzip")
	}
	cmd := exec.CommandContext(ctx, binPath, args...)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start mongodump: %w", err)
	}
	copyErrCh := make(chan error, 1)
	go func() {
		_, err := io.Copy(writer, stdoutPipe)
		if err != nil {
			copyErrCh <- fmt.Errorf("copy mongodump output: %w", err)
			return
		}
		copyErrCh <- nil
	}()
	stderrBytes, _ := io.ReadAll(stderrPipe)
	waitErr := cmd.Wait()
	copyErr := <-copyErrCh
	if waitErr != nil {
		return "", fmt.Errorf("mongodump failed: %v, stderr: %s", waitErr, strings.TrimSpace(string(stderrBytes)))
	}
	if copyErr != nil {
		return "", copyErr
	}
	return buildMongoArtifactNameFromRequest(req.Database, req.CompressionType), nil
}

func (r *MongoRunner) RunLogicalRestore(ctx context.Context, req *model.MongoLogicalRestoreRequest, reader io.Reader) error {
	if req == nil {
		return fmt.Errorf("mongo restore request is required")
	}
	uri, err := runnertools.BuildMongoURI(req.Host, req.Port, req.Username, req.Password, req.AuthDatabase, req.ConnectionURI)
	if err != nil {
		return err
	}
	binPath, err := runnertools.GetMongoExecutable(r.toolsDir, "mongorestore")
	if err != nil {
		return err
	}
	args := []string{
		"--uri=" + uri,
		"--archive",
	}
	if strings.EqualFold(req.CompressionType, "gzip") {
		args = append(args, "--gzip")
	}
	sourceDatabase := strings.TrimSpace(req.SourceDatabase)
	targetDatabase := strings.TrimSpace(req.Database)
	if sourceDatabase != "" && targetDatabase != "" && sourceDatabase != targetDatabase {
		args = append(args,
			"--nsFrom="+sourceDatabase+".*",
			"--nsTo="+targetDatabase+".*",
		)
	}
	cmd := exec.CommandContext(ctx, binPath, args...)
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start mongorestore: %w", err)
	}
	copyErrCh := make(chan error, 1)
	go func() {
		defer stdinPipe.Close()
		_, err := io.Copy(stdinPipe, reader)
		if err != nil {
			copyErrCh <- fmt.Errorf("stream restore input: %w", err)
			return
		}
		copyErrCh <- nil
	}()
	stderrBytes, _ := io.ReadAll(stderrPipe)
	waitErr := cmd.Wait()
	copyErr := <-copyErrCh
	if copyErr != nil {
		return copyErr
	}
	if waitErr != nil {
		return fmt.Errorf("mongorestore failed: %v, stderr: %s", waitErr, strings.TrimSpace(string(stderrBytes)))
	}
	return nil
}

func buildMongoArtifactNameFromRequest(database string, compressionType string) string {
	base := strings.TrimSpace(database)
	if base == "" {
		base = "mongodb-backup"
	}
	if strings.EqualFold(compressionType, "gzip") {
		return base + ".archive.gz"
	}
	return base + ".archive"
}
