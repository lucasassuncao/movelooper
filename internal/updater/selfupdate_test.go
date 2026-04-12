package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func serveFakeRelease(t *testing.T, statusCode int, body interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}))
}

type redirectTransport struct{ target string }

func (r redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Host = r.target
	req2.URL.Scheme = "http"
	return http.DefaultTransport.RoundTrip(req2)
}

func withTestServer(t *testing.T, srv *httptest.Server, fn func()) {
	t.Helper()
	orig := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: redirectTransport{target: srv.Listener.Addr().String()}}
	t.Cleanup(func() { http.DefaultClient = orig })
	fn()
}

// ── normalizeVersion ──────────────────────────────────────────────────────────

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"v1.2.3", "1.2.3"},
		{"1.2.3", "1.2.3"},
		{"v0.0.1", "0.0.1"},
		{"", ""},
		{"vv1.0.0", "v1.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeVersion(tt.input))
		})
	}
}

// ── selectAsset ───────────────────────────────────────────────────────────────

func TestSelectAsset(t *testing.T) {
	tests := []struct {
		name     string
		assets   []ghAsset
		wantName string // empty = nil expected
	}{
		{"nil assets", nil, ""},
		{"empty assets", []ghAsset{}, ""},
		{
			"skips checksum/sig files",
			[]ghAsset{
				{Name: "movelooper_checksums.txt"},
				{Name: "movelooper_linux_amd64.sha256"},
				{Name: "movelooper_linux_amd64.sig"},
			},
			"",
		},
		{
			"prefers windows exe",
			[]ghAsset{
				{Name: "movelooper_linux_amd64"},
				{Name: "movelooper_windows_amd64.exe"},
			},
			"movelooper_windows_amd64.exe",
		},
		{
			"falls back to first on tie",
			[]ghAsset{
				{Name: "movelooper_linux_arm64"},
				{Name: "movelooper_darwin_arm64"},
			},
			"movelooper_linux_arm64",
		},
		{
			"scores arch match",
			[]ghAsset{
				{Name: "movelooper_linux_arm64"},
				{Name: "movelooper_linux_amd64"},
			},
			"movelooper_linux_amd64",
		},
		{
			"exe score without os name",
			[]ghAsset{
				{Name: "movelooper_arm64"},
				{Name: "movelooper_amd64.exe"},
			},
			"movelooper_amd64.exe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectAsset(tt.assets)
			if tt.wantName == "" {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.wantName, got.Name)
			}
		})
	}
}

// ── fetchLatestRelease ────────────────────────────────────────────────────────

func TestFetchLatestRelease(t *testing.T) {
	var capturedAuth string

	tests := []struct {
		name    string
		server  func() *httptest.Server
		token   string
		wantErr string
		check   func(t *testing.T, rel *ghRelease)
	}{
		{
			name: "200 ok returns release",
			server: func() *httptest.Server {
				return serveFakeRelease(t, http.StatusOK, ghRelease{
					TagName: "v2.5.3",
					Assets:  []ghAsset{{Name: "movelooper_linux_amd64"}},
				})
			},
			check: func(t *testing.T, rel *ghRelease) {
				assert.Equal(t, "v2.5.3", rel.TagName)
				assert.Len(t, rel.Assets, 1)
			},
		},
		{
			name:    "404 returns not found error",
			server:  func() *httptest.Server { return serveFakeRelease(t, http.StatusNotFound, nil) },
			wantErr: "no releases found",
		},
		{
			name:    "500 returns status error",
			server:  func() *httptest.Server { return serveFakeRelease(t, http.StatusInternalServerError, nil) },
			wantErr: "500",
		},
		{
			name: "sends bearer token",
			server: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedAuth = r.Header.Get("Authorization")
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(ghRelease{TagName: "v1.0.0"})
				}))
			},
			token: "mytoken",
			check: func(t *testing.T, _ *ghRelease) {
				assert.Equal(t, "Bearer mytoken", capturedAuth)
			},
		},
		{
			name: "invalid json returns decode error",
			server: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("not json"))
				}))
			},
			wantErr: "decoding release",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := tt.server()
			defer srv.Close()

			withTestServer(t, srv, func() {
				rel, err := fetchLatestRelease("owner/repo", tt.token)
				if tt.wantErr != "" {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tt.wantErr)
					return
				}
				require.NoError(t, err)
				if tt.check != nil {
					tt.check(t, rel)
				}
			})
		})
	}
}

// ── download ──────────────────────────────────────────────────────────────────

func TestDownload(t *testing.T) {
	var capturedAuth string
	content := []byte("fake binary content")

	tests := []struct {
		name    string
		server  func() *httptest.Server
		destFn  func(dir string) string
		token   string
		wantErr string
		check   func(t *testing.T, dest string)
	}{
		{
			name: "writes content to dest",
			server: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write(content)
				}))
			},
			destFn: func(dir string) string { return filepath.Join(dir, "binary") },
			check: func(t *testing.T, dest string) {
				got, err := os.ReadFile(dest)
				require.NoError(t, err)
				assert.Equal(t, content, got)
			},
		},
		{
			name: "http error returns status error",
			server: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusForbidden)
				}))
			},
			destFn:  func(dir string) string { return filepath.Join(dir, "binary") },
			wantErr: "403",
		},
		{
			name: "invalid dest path returns error",
			server: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("data"))
				}))
			},
			destFn:  func(_ string) string { return "/nonexistent/dir/binary" },
			wantErr: "creating temp binary",
		},
		{
			name: "sends bearer token",
			server: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedAuth = r.Header.Get("Authorization")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("data"))
				}))
			},
			destFn: func(dir string) string { return filepath.Join(dir, "binary") },
			token:  "tok123",
			check: func(t *testing.T, _ string) {
				assert.Equal(t, "Bearer tok123", capturedAuth)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := tt.server()
			defer srv.Close()

			dest := tt.destFn(t.TempDir())
			err := download(srv.URL, dest, tt.token)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, dest)
			}
		})
	}
}

// ── CleanOldBinary ────────────────────────────────────────────────────────────

func TestCleanOldBinary(t *testing.T) {
	tests := []struct {
		name      string
		createOld bool
	}{
		{"removes .old file when present", true},
		{"noop when .old file absent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exe, err := os.Executable()
			require.NoError(t, err)
			old := exe + ".old"

			os.Remove(old)
			if tt.createOld {
				f, err := os.Create(old)
				require.NoError(t, err)
				f.Close()
			}

			assert.NotPanics(t, func() { CleanOldBinary() })
			_, statErr := os.Stat(old)
			assert.True(t, os.IsNotExist(statErr), ".old file should not exist after CleanOldBinary")
		})
	}
}

// ── SelfUpdate ────────────────────────────────────────────────────────────────

func TestSelfUpdate(t *testing.T) {
	tests := []struct {
		name           string
		repo           string
		currentVersion string
		release        *ghRelease
		wantErr        string
	}{
		{
			name:    "empty repo returns error",
			repo:    "",
			wantErr: "--repo is required",
		},
		{
			name:           "already up to date",
			repo:           "owner/repo",
			currentVersion: "v2.5.3",
			release:        &ghRelease{TagName: "v2.5.3", Assets: []ghAsset{{Name: "asset"}}},
		},
		{
			name:           "no compatible asset",
			repo:           "owner/repo",
			currentVersion: "v1.0.0",
			release:        &ghRelease{TagName: "v9.9.9", Assets: []ghAsset{{Name: "checksums.txt"}}},
			wantErr:        "no compatible binary found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.release != nil {
				srv := serveFakeRelease(t, http.StatusOK, tt.release)
				defer srv.Close()
				withTestServer(t, srv, func() {
					err := SelfUpdate(tt.repo, "", tt.currentVersion)
					if tt.wantErr != "" {
						require.Error(t, err)
						assert.Contains(t, err.Error(), tt.wantErr)
					} else {
						assert.NoError(t, err)
					}
				})
				return
			}
			err := SelfUpdate(tt.repo, "", tt.currentVersion)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestSelfUpdate_DownloadsAndReplacesBinary(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "movelooper")
	require.NoError(t, os.WriteFile(fakeBin, []byte("old"), 0o755))

	newContent := []byte("new binary content")

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/download" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(newContent)
			return
		}
		rel := ghRelease{
			TagName: "v9.9.9",
			Assets: []ghAsset{{
				Name:               "movelooper_linux_amd64",
				BrowserDownloadURL: srv.URL + "/download",
				Size:               int64(len(newContent)),
			}},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(rel)
	}))
	defer srv.Close()

	origExecutable := osExecutable
	osExecutable = func() (string, error) { return fakeBin, nil }
	t.Cleanup(func() { osExecutable = origExecutable })

	withTestServer(t, srv, func() {
		require.NoError(t, SelfUpdate("owner/repo", "", "v1.0.0"))
	})

	got, err := os.ReadFile(fakeBin)
	require.NoError(t, err)
	assert.Equal(t, newContent, got)

	_, err = os.Stat(fakeBin + ".old")
	assert.NoError(t, err, ".old binary should exist")
}
