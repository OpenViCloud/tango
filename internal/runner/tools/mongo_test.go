package tools

import "testing"

func TestBuildMongoURIWithAuthSource(t *testing.T) {
	uri, err := BuildMongoURI("mongo", 27017, "root", "secret", "admin", "")
	if err != nil {
		t.Fatalf("BuildMongoURI() error = %v", err)
	}
	expected := "mongodb://root:secret@mongo:27017/?authSource=admin"
	if uri != expected {
		t.Fatalf("uri = %s, want %s", uri, expected)
	}
}

func TestBuildMongoURIUsesProvidedConnectionURI(t *testing.T) {
	uri, err := BuildMongoURI("mongo", 27017, "root", "secret", "admin", "mongodb://example")
	if err != nil {
		t.Fatalf("BuildMongoURI() error = %v", err)
	}
	if uri != "mongodb://example" {
		t.Fatalf("uri = %s", uri)
	}
}
