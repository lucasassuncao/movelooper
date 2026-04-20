package tokens

import (
	"crypto/md5" //#nosec G501 -- MD5 used for non-cryptographic file identification
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

var (
	md5Token    = regexp.MustCompile(`\{md5(?::(\d+))?\}`)
	sha256Token = regexp.MustCompile(`\{sha256:(\d+)\}`)
)

func preProcessHash(template, sourcePath string) string {
	if md5Token.MatchString(template) {
		h := computeFileHash(sourcePath, md5.New()) //#nosec G401 -- MD5 used for non-cryptographic file identification
		template = md5Token.ReplaceAllStringFunc(template, func(tok string) string {
			m := md5Token.FindStringSubmatch(tok)
			n := 8
			if m[1] != "" {
				n, _ = strconv.Atoi(m[1])
			}
			if len(h) < n {
				return h
			}
			return h[:n]
		})
	}
	if sha256Token.MatchString(template) {
		h := computeFileHash(sourcePath, sha256.New())
		template = sha256Token.ReplaceAllStringFunc(template, func(tok string) string {
			m := sha256Token.FindStringSubmatch(tok)
			n, _ := strconv.Atoi(m[1])
			if len(h) < n {
				return h
			}
			return h[:n]
		})
	}
	return template
}

func computeFileHash(path string, h hash.Hash) string {
	f, err := os.Open(filepath.Clean(path)) //#nosec G304 -- path comes from validated file walk
	if err != nil {
		return "unknown"
	}
	defer f.Close()
	if _, err := io.Copy(h, f); err != nil {
		return "unknown"
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
