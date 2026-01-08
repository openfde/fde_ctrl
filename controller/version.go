package controller

import (
	"bufio"
	"errors"
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type VersionController struct {
}

type VersionRequest struct {
	Version string
}

func (impl VersionController) Setup(rg *gin.RouterGroup) {
	v1 := rg.Group("/v1")
	v1.POST("/version/check", impl.versionHandler)
}


// parseDebianPackages parses RFC822-like "Packages" blocks into a slice of field maps.
func parseDebianPackages(content string) []map[string]string {
	var entries []map[string]string
	var cur map[string]string
	var lastKey string

	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := sc.Text()

		// blank line => end of block
		if strings.TrimSpace(line) == "" {
			if cur != nil {
				entries = append(entries, cur)
				cur = nil
				lastKey = ""
			}
			continue
		}

		// continuation line (starts with space or tab)
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			if cur != nil && lastKey != "" {
				// append continuation; join with newline to preserve content
				cur[lastKey] = cur[lastKey] + "\n" + strings.TrimSpace(line)
			}
			continue
		}

		// key: value
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			// malformed line; skip
			continue
		}
		key := strings.TrimSpace(line[:colon])
		val := strings.TrimSpace(line[colon+1:])

		if cur == nil {
			cur = make(map[string]string)
		}
		cur[key] = val
		lastKey = key
	}
	if cur != nil {
		entries = append(entries, cur)
	}
	return entries
}

// compareVersions returns -1 if a<b, 0 if equal, 1 if a>b.
// Rule:
// 1) Compare dot-separated numeric segments before '-' (semantic-like).
// 2) If equal, compare leading numeric part after '-' as date (if present).
// 3) If still equal or no date, fallback to lexicographic.
func compareVersions(a, b string) int {
	parse := func(s string) (sem []int, date int, hasDate bool) {
		parts := strings.SplitN(s, "-", 2)
		for _, p := range strings.Split(parts[0], ".") {
			if p == "" {
				sem = append(sem, 0)
				continue
			}
			n, err := strconv.Atoi(p)
			if err != nil {
				// non-numeric segment treated as 0 to avoid crash
				n = 0
			}
			sem = append(sem, n)
		}
		if len(parts) == 2 {
			suf := parts[1]
			i := 0
			for i < len(suf) && suf[i] >= '0' && suf[i] <= '9' {
				i++
			}
			if i > 0 {
				d, err := strconv.Atoi(suf[:i])
				if err == nil {
					date = d
					hasDate = true
				}
			}
		}
		return
	}

	semA, dateA, hasDateA := parse(a)
	semB, dateB, hasDateB := parse(b)

	// compare semantic segments
	maxLen := len(semA)
	if len(semB) > maxLen {
		maxLen = len(semB)
	}
	for i := 0; i < maxLen; i++ {
		va, vb := 0, 0
		if i < len(semA) {
			va = semA[i]
		}
		if i < len(semB) {
			vb = semB[i]
		}
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
	}

	// same semantic, compare date if both have it
	if hasDateA && hasDateB {
		if dateA < dateB {
			return -1
		}
		if dateA > dateB {
			return 1
		}
	} else if hasDateA != hasDateB {
		// prefer one that has date
		if hasDateA {
			return 1
		}
		return -1
	}

	// fallback lexicographic
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// LatestForPackage returns the latest entry for a given package name.
func LatestForPackage(entries []map[string]string, pkg string) (map[string]string, error) {
	var best map[string]string
	var bestVer string
	for _, e := range entries {
		if e["Package"] != pkg {
			continue
		}
		v := e["Version"]
		if v == "" {
			continue
		}
		if best == nil || compareVersions(bestVer, v) < 0 {
			best = e
			bestVer = v
		}
	}
	if best == nil {
		return nil, fmt.Errorf("package %s not found", pkg)
	}
	return best, nil
}

type versionResponse struct {
	Version string
	IsNewer int
	DownloadURL string
	Size string
	MD5 string
}

const FDE_APT_FILE = "/etc/apt/sources.list.d/openfde.list"

func (impl VersionController) versionHandler(c *gin.Context) {
	arch, repoURL, release := "", "", ""
	var request VersionRequest
	err := c.ShouldBind(&request)
	if err != nil {
		logger.Error("version_request_parse",err,nil)
		response.ResponseParamterError(c, err)
		return
	}
	logger.Info("parse_version_request",request)
	f, err := os.Open(FDE_APT_FILE)
	if err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)

		re := regexp.MustCompile(`^\s*deb(?:-src)?\s+(?:\[([^\]]+)\]\s+)?(\S+)\s+(\S+)`)
		archRe := regexp.MustCompile(`(?:^|\s)arch=([^\s,]+)`)

		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			m := re.FindStringSubmatch(line)
			if len(m) == 4 {
				opts := m[1]
				if a := archRe.FindStringSubmatch(opts); len(a) > 1 {
					arch = a[1]
				}
				repoURL = m[2]
				release = m[3]
				break
			}
		}
	}
	if repoURL == "" {
		response.ResponseError(c, http.StatusPreconditionRequired, errors.New("repo files not found"))
		return
	}
	targetURL := repoURL + "/dists/" + release + "/main/binary-" + arch + "/Packages"
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		response.ResponseError(c, http.StatusPreconditionRequired, errors.New("create http client failed"))
		return
	} else {
		resp, err := client.Do(req)
		if err != nil {
			logger.Error("request_failed", targetURL, err)
			response.ResponseError(c, http.StatusPreconditionRequired, errors.New("do http request failed failed"))
			return
		} else {
			defer resp.Body.Close()
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				response.ResponseError(c, http.StatusPreconditionRequired, errors.New("read http client failed"))
				return
			}
			entries := parseDebianPackages(string(bodyBytes))
			best, err := LatestForPackage(entries, "openfde14")
			if err != nil {
				response.ResponseError(c, http.StatusPreconditionRequired, errors.New("failed to find openfde14 package"))
				return
			}
			if v := strings.TrimSpace(request.Version); v != "" {
				cmp := compareVersions(v, best["Version"])
				response.Response(c, versionResponse{
					Version: best["Version"], 
					IsNewer: cmp, 
					DownloadURL: repoURL + best["Filename"],
					MD5: best["MD5sum"],
					Size: best["Size"],
				})
				return
			}
		}
	}
}

