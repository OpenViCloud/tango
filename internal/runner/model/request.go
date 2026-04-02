package model

type MySQLLogicalDumpRequest struct {
	Version         string `json:"version"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Database        string `json:"database"`
	CompressionType string `json:"compression_type"`
}

type MariaDBLogicalDumpRequest struct {
	Version         string `json:"version"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Database        string `json:"database"`
	CompressionType string `json:"compression_type"`
}

type MongoLogicalDumpRequest struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Database        string `json:"database"`
	AuthDatabase    string `json:"auth_database"`
	ConnectionURI   string `json:"connection_uri"`
	CompressionType string `json:"compression_type"`
}

type MongoLogicalRestoreRequest struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Database        string `json:"database"`
	AuthDatabase    string `json:"auth_database"`
	ConnectionURI   string `json:"connection_uri"`
	SourceDatabase  string `json:"source_database"`
	CompressionType string `json:"compression_type"`
}

type PostgresLogicalDumpRequest struct {
	Version         string `json:"version"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Database        string `json:"database"`
	CompressionType string `json:"compression_type"`
}
