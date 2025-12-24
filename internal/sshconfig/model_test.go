package sshconfig

import "testing"

func TestHostEntry_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		entry *HostEntry
		want  bool
	}{
		{
			name: "valid regular entry",
			entry: &HostEntry{
				Host:     "example",
				HostName: "example.com",
			},
			want: true,
		},
		{
			name: "valid Host * entry",
			entry: &HostEntry{
				Host: "*",
			},
			want: true,
		},
		{
			name: "invalid - empty host",
			entry: &HostEntry{
				Host:     "",
				HostName: "example.com",
			},
			want: false,
		},
		{
			name: "invalid - missing hostname",
			entry: &HostEntry{
				Host:     "example",
				HostName: "",
			},
			want: false,
		},
		{
			name: "valid with all fields",
			entry: &HostEntry{
				Host:         "example",
				HostName:     "example.com",
				User:         "root",
				Port:         "22",
				IdentityFile: "~/.ssh/id_rsa",
				Description:  "Test server",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.IsValid(); got != tt.want {
				t.Errorf("HostEntry.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHostEntry_GetConnectionString(t *testing.T) {
	tests := []struct {
		name  string
		entry *HostEntry
		want  string
	}{
		{
			name: "with user",
			entry: &HostEntry{
				HostName: "example.com",
				User:     "root",
			},
			want: "root@example.com",
		},
		{
			name: "without user",
			entry: &HostEntry{
				HostName: "example.com",
			},
			want: "example.com",
		},
		{
			name: "with IP address",
			entry: &HostEntry{
				HostName: "192.168.1.1",
				User:     "admin",
			},
			want: "admin@192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.GetConnectionString(); got != tt.want {
				t.Errorf("HostEntry.GetConnectionString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHostEntry_GetSSHCommand(t *testing.T) {
	tests := []struct {
		name  string
		entry *HostEntry
		want  string
	}{
		{
			name: "without port",
			entry: &HostEntry{
				HostName: "example.com",
				User:     "root",
			},
			want: "ssh root@example.com",
		},
		{
			name: "with port",
			entry: &HostEntry{
				HostName: "example.com",
				User:     "root",
				Port:     "2222",
			},
			want: "ssh -p 2222 root@example.com",
		},
		{
			name: "without user",
			entry: &HostEntry{
				HostName: "example.com",
			},
			want: "ssh example.com",
		},
		{
			name: "with user and custom port",
			entry: &HostEntry{
				HostName: "192.168.1.1",
				User:     "admin",
				Port:     "22000",
			},
			want: "ssh -p 22000 admin@192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.GetSSHCommand(); got != tt.want {
				t.Errorf("HostEntry.GetSSHCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}
