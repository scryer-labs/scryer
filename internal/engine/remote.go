package engine

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

//go:embed releases.json
var defaultReleases []byte

type releaseSource struct {
	Repository string `json:"repository"`
	Asset      string `json:"asset"`
}

type releaseAsset struct {
	Name   string
	Digest string
	Size   int64
	URL    string `json:"browser_download_url"`
}

type release struct {
	Tag        string `json:"tag_name"`
	Draft      bool
	Prerelease bool
	Assets     []releaseAsset
}

var repositoryName = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
var stableTag = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func version(value string) ([3]uint64, bool) {
	var result [3]uint64
	match := stableTag.FindStringSubmatch(value)
	if match == nil {
		return result, false
	}
	for i := range result {
		number, err := strconv.ParseUint(match[i+1], 10, 64)
		if err != nil {
			return result, false
		}
		result[i] = number
	}
	return result, true
}

func newer(a, b [3]uint64) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

func (s Store) releaseSource(id string) (releaseSource, error) {
	var defaults, overrides map[string]releaseSource
	if err := json.Unmarshal(defaultReleases, &defaults); err != nil {
		return releaseSource{}, err
	}
	err := readJSON(filepath.Join(s.Home, "engine-sources.json"), &overrides)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return releaseSource{}, fmt.Errorf("engine sources: %w", err)
	}
	source, ok := overrides[id]
	if !ok {
		source = defaults[id]
	}
	if !repositoryName.MatchString(source.Repository) || strings.Contains(source.Repository, "..") || source.Asset == "" ||
		path.Base(source.Asset) != source.Asset || strings.ContainsAny(source.Asset, "\\:") {
		return releaseSource{}, fmt.Errorf("no valid release source for engine %q in engine-sources.json", id)
	}
	return source, nil
}

var releaseHTTP = &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
	if req.URL.Scheme != "https" || len(via) >= 6 {
		return fmt.Errorf("invalid release redirect")
	}
	return nil
}}

func releaseGet(ctx context.Context, address string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	response, err := releaseHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		if response.StatusCode == http.StatusForbidden && response.Header.Get("X-RateLimit-Remaining") == "0" {
			reset, err := strconv.ParseInt(response.Header.Get("X-RateLimit-Reset"), 10, 64)
			if err == nil {
				return nil, fmt.Errorf("GitHub API rate limit reached; retry after %s", time.Unix(reset, 0).UTC().Format(time.RFC3339))
			}
			return nil, fmt.Errorf("GitHub API rate limit reached")
		}
		return nil, fmt.Errorf("GitHub release request returned %s", response.Status)
	}
	return response, nil
}

func (source releaseSource) latest(ctx context.Context) (string, releaseAsset, error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	var best releaseAsset
	var bestVersion [3]uint64
	var tag string
	for page := 1; ; page++ {
		address := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=100&page=%d", source.Repository, page)
		response, err := releaseGet(ctx, address)
		if err != nil {
			return "", best, err
		}
		var releases []release
		err = json.NewDecoder(io.LimitReader(response.Body, 16<<20)).Decode(&releases)
		response.Body.Close()
		if err != nil {
			return "", best, err
		}
		for _, candidate := range releases {
			v, ok := version(candidate.Tag)
			if !ok || candidate.Draft || candidate.Prerelease || tag != "" && !newer(v, bestVersion) {
				continue
			}
			name := strings.NewReplacer("{version}", candidate.Tag, "{os}", runtime.GOOS, "{arch}", runtime.GOARCH).Replace(source.Asset)
			for _, asset := range candidate.Assets {
				if asset.Name == name {
					best, bestVersion, tag = asset, v, candidate.Tag
					break
				}
			}
		}
		if len(releases) < 100 {
			break
		}
	}
	if tag == "" {
		return "", best, fmt.Errorf("no stable %s/%s package in GitHub releases for %s", runtime.GOOS, runtime.GOARCH, source.Repository)
	}
	return tag, best, nil
}

const maxReleaseBytes = 512 << 20

type downloadProgress struct {
	output     io.Writer
	name       string
	size       int64
	downloaded int64
	nextUpdate time.Time
}

func (p *downloadProgress) Write(data []byte) (int, error) {
	p.downloaded += int64(len(data))
	now := time.Now()
	if !now.Before(p.nextUpdate) {
		fmt.Fprintf(p.output, "%s: %.1f / %.1f MiB downloaded\n", p.name,
			float64(p.downloaded)/(1<<20), float64(p.size)/(1<<20))
		p.nextUpdate = now.Add(5 * time.Second)
	}
	return len(data), nil
}

func downloadPackage(ctx context.Context, asset releaseAsset, dir string, output io.Writer) (string, error) {
	expected, err := hex.DecodeString(strings.TrimPrefix(asset.Digest, "sha256:"))
	if err != nil || len(expected) != sha256.Size || !strings.HasPrefix(asset.Digest, "sha256:") {
		return "", fmt.Errorf("release asset lacks a valid GitHub SHA-256 digest")
	}
	address, err := url.Parse(asset.URL)
	if err != nil || address.Scheme != "https" || address.Host != "github.com" || address.User != nil {
		return "", fmt.Errorf("invalid GitHub release asset URL")
	}
	if asset.Size <= 0 || asset.Size > maxReleaseBytes {
		return "", fmt.Errorf("invalid release asset size: %d bytes", asset.Size)
	}
	fmt.Fprintf(output, "Downloading %s (%.1f MiB)\n", asset.Name, float64(asset.Size)/(1<<20))
	downloadCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	response, err := releaseGet(downloadCtx, address.String())
	if err != nil {
		return "", fmt.Errorf("start download: %w", err)
	}
	defer response.Body.Close()
	archive, err := os.CreateTemp(dir, "asset-*.zip")
	if err != nil {
		return "", err
	}
	defer archive.Close()
	hash := sha256.New()
	progress := downloadProgress{output: output, name: asset.Name, size: asset.Size, nextUpdate: time.Now().Add(5 * time.Second)}
	size, err := io.Copy(io.MultiWriter(archive, hash, &progress), io.LimitReader(response.Body, asset.Size+1))
	if err != nil {
		return "", fmt.Errorf("download after %d of %d bytes: %w", size, asset.Size, err)
	}
	if size != asset.Size || !bytes.Equal(expected, hash.Sum(nil)) {
		return "", fmt.Errorf("release asset size or SHA-256 verification failed")
	}
	reader, err := zip.NewReader(archive, size)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(output, "Verified %s; extracting package\n", asset.Name)
	root := filepath.Join(dir, "package")
	if err := extractPackage(ctx, reader, root); err != nil {
		return "", err
	}
	return root, nil
}

func extractPackage(ctx context.Context, reader *zip.Reader, root string) error {
	var total uint64
	for _, entry := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := strings.TrimSuffix(entry.Name, "/")
		if !filepath.IsLocal(filepath.FromSlash(name)) ||
			strings.ContainsAny(name, "\\:") || path.Clean(name) != name || name == "." || name == installRecord {
			return fmt.Errorf("unsafe release archive path %q", entry.Name)
		}
		target := filepath.Join(root, filepath.FromSlash(name))
		if entry.Mode().IsDir() {
			if err := os.MkdirAll(target, 0700); err != nil {
				return err
			}
			continue
		}
		if !entry.Mode().IsRegular() || entry.UncompressedSize64 > maxReleaseBytes-total {
			return fmt.Errorf("unsupported or oversized archive file %q", entry.Name)
		}
		total += entry.UncompressedSize64
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err := extractFile(entry, target); err != nil {
			return err
		}
	}
	return nil
}

func extractFile(entry *zip.File, target string) error {
	input, err := entry.Open()
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600|entry.Mode().Perm()&0111)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(output, io.LimitReader(input, int64(entry.UncompressedSize64)+1))
	if copyErr == nil && uint64(written) != entry.UncompressedSize64 {
		copyErr = fmt.Errorf("archive file size does not match its header: %q", entry.Name)
	}
	return errors.Join(copyErr, output.Close())
}

func (s Store) Fetch(ctx context.Context, id string, upgrade bool, diagnostics, progress io.Writer) (Installation, bool, error) {
	if err := validateID(id); err != nil {
		return Installation{}, false, err
	}
	lookupCtx, cancelLookup := context.WithTimeout(ctx, 3*time.Minute)
	previous, err := s.Find(lookupCtx, id)
	cancelLookup()
	if upgrade && err != nil {
		return Installation{}, false, err
	}
	if !upgrade && !errors.Is(err, ErrNotInstalled) {
		if err == nil {
			err = fmt.Errorf("engine %q is already installed; use engine upgrade", id)
		}
		return Installation{}, false, err
	}
	source, err := s.releaseSource(id)
	if err != nil {
		return Installation{}, false, err
	}
	tag, asset, err := source.latest(ctx)
	if err != nil {
		return Installation{}, false, fmt.Errorf("check releases: %w", err)
	}
	if upgrade {
		current, ok := version(previous.Engine.Version)
		candidate, _ := version(tag)
		if ok && !newer(candidate, current) {
			return previous, false, nil
		}
	}
	dir, err := os.MkdirTemp("", "scryer-release-")
	if err != nil {
		return Installation{}, false, err
	}
	defer os.RemoveAll(dir)
	pkg, err := downloadPackage(ctx, asset, dir, progress)
	if err != nil {
		return Installation{}, false, fmt.Errorf("prepare release %s: %w", tag, err)
	}
	manifest, _, err := ReadPackage(pkg)
	if err != nil {
		return Installation{}, false, err
	}
	if manifest.Engine.Version != strings.TrimPrefix(tag, "v") || manifest.Engine.Source != "github.com/"+source.Repository {
		return Installation{}, false, fmt.Errorf("release tag or repository does not match package manifest")
	}
	if !upgrade {
		previous = Installation{}
	}
	installCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	installed, err := s.install(installCtx, id, pkg, asset.URL, previous, diagnostics)
	return installed, err == nil, err
}
