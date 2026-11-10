package repository

import (
	"context"
	"net"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/process"
)

// Identity describes the project root using its origin or its only configured remote.
func Identity(ctx context.Context, root string) contract.RepositoryIdentity {
	local := contract.RepositoryIdentity{Host: "local", Namespace: "local", Name: filepath.Base(root)}
	prefix, err := process.Command(ctx, root, "git", "rev-parse", "--show-prefix").Output()
	if err != nil || strings.TrimSpace(string(prefix)) != "" {
		return local
	}

	remote := "origin"
	address, err := process.Command(ctx, root, "git", "config", "--get", "remote.origin.url").Output()
	if err != nil {
		list, err := process.Command(ctx, root, "git", "remote").Output()
		names := strings.Fields(string(list))
		if err != nil || len(names) != 1 {
			return local
		}
		remote = names[0]
		address, err = process.Command(ctx, root, "git", "config", "--get", "remote."+remote+".url").Output()
		if err != nil {
			return local
		}
	}
	identity, ok := ParseRemote(strings.TrimSpace(string(address)))
	if !ok {
		return local
	}
	identity.Remote = remote
	return identity
}

// ParseRemote normalizes hosted Git URLs without connecting to the remote.
func ParseRemote(address string) (contract.RepositoryIdentity, bool) {
	if address == "" || strings.ContainsAny(address, " \t\r\n\\{}<>") {
		return contract.RepositoryIdentity{}, false
	}
	if !strings.Contains(address, "://") {
		// Git's SCP form has no slash before the colon that separates host and path.
		authority, path, found := strings.Cut(address, ":")
		if user, host, hasUser := strings.Cut(authority, "@"); hasUser && user != "" {
			authority = host
		}
		if !found || strings.ContainsAny(authority, "/@?#") || len(authority) == 1 {
			return contract.RepositoryIdentity{}, false
		}
		address = "ssh://" + authority + "/" + strings.TrimLeft(path, "/")
	}
	parsed, err := url.Parse(address)
	if err != nil || parsed.Hostname() == "" {
		return contract.RepositoryIdentity{}, false
	}
	defaultPorts := map[string]string{"ssh": "22", "https": "443", "http": "80", "git": "9418"}
	defaultPort, supported := defaultPorts[parsed.Scheme]
	if !supported {
		return contract.RepositoryIdentity{}, false
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "" || strings.ContainsAny(host, " \t\r\n") {
		return contract.RepositoryIdentity{}, false
	}
	if port := parsed.Port(); port != "" && port != defaultPort {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	path := strings.Trim(parsed.EscapedPath(), "/")
	parts := strings.Split(path, "/")
	for i, part := range parts {
		value, err := url.PathUnescape(part)
		if err != nil || value == "" || value == "." || value == ".." || strings.ContainsAny(value, "/\\\r\n\x00") {
			return contract.RepositoryIdentity{}, false
		}
		parts[i] = value
	}
	last := len(parts) - 1
	parts[last] = strings.TrimSuffix(parts[last], ".git")
	if parts[last] == "" || parts[last] == "." || parts[last] == ".." {
		return contract.RepositoryIdentity{}, false
	}
	key := strings.Join(parts, "/")
	if host == "github.com" {
		key = strings.ToLower(key)
	}
	return contract.RepositoryIdentity{
		Host: host, Namespace: strings.Join(parts[:last], "/"), Name: parts[last], ID: host + "/" + key,
	}, true
}
