package ghpr

import "testing"

func TestHostFromRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		want      string
		wantErr   bool
	}{
		{
			name:      "https",
			remoteURL: "https://github.com/owner/repo.git",
			want:      "github.com",
		},
		{
			name:      "https enterprise with port",
			remoteURL: "https://ghe.example.com:8443/owner/repo.git",
			want:      "ghe.example.com:8443",
		},
		{
			name:      "ssh scp-like",
			remoteURL: "git@github.com:owner/repo.git",
			want:      "github.com",
		},
		{
			name:      "unsupported",
			remoteURL: "/tmp/repo",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HostFromRemoteURL(tt.remoteURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("HostFromRemoteURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
