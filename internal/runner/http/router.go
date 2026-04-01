package http

import (
	nethttp "net/http"
)

func NewRouter(handler *Handler) nethttp.Handler {
	mux := nethttp.NewServeMux()
	mux.HandleFunc("/healthz", handler.Healthz)
	mux.HandleFunc("/internal/mysql/logical-dump", handler.MySQLLogicalDump)
	mux.HandleFunc("/internal/mysql/logical-restore", handler.MySQLLogicalRestore)
	mux.HandleFunc("/internal/postgres/logical-dump", handler.PostgresLogicalDump)
	mux.HandleFunc("/internal/postgres/logical-restore", handler.PostgresLogicalRestore)
	mux.HandleFunc("/internal/mongo/logical-dump", handler.MongoLogicalDump)
	mux.HandleFunc("/internal/mongo/logical-restore", handler.MongoLogicalRestore)
	return mux
}
