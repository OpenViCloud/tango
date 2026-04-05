package main

import "testing"

func TestRuntimeDirFromComposeFile(t *testing.T) {
	tests := []struct {
		name        string
		composeFile string
		want        string
	}{
		{
			name:        "empty compose file",
			composeFile: "",
			want:        "",
		},
		{
			name:        "default tango runtime",
			composeFile: "/opt/tango/docker-compose.yml",
			want:        "/opt/tango",
		},
		{
			name:        "nested tango runtime",
			composeFile: "/opt/tango/dev/docker-compose.yml",
			want:        "/opt/tango/dev",
		},
		{
			name:        "non tango runtime",
			composeFile: "/srv/app/docker-compose.yml",
			want:        "",
		},
		{
			name:        "root dir compose file",
			composeFile: "/docker-compose.yml",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runtimeDirFromComposeFile(tt.composeFile); got != tt.want {
				t.Fatalf("runtimeDirFromComposeFile(%q) = %q, want %q", tt.composeFile, got, tt.want)
			}
		})
	}
}
