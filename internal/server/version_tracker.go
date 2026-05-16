package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// TODO: dual-source. Mirror at gitlab.com/sartoopjj/thefeed; fall back to
// /api/v4/projects/sartoopjj%2Fthefeed/releases?per_page=1 — same pattern
// as scripts/install.sh resolve_source().
const latestReleaseURL = "https://api.github.com/repos/sartoopjj/thefeed/releases/latest"

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// startLatestVersionTracker periodically fetches latest GitHub release version
// and stores it in the dedicated version channel.
func startLatestVersionTracker(ctx context.Context, feed *Feed) {
	update := func() {
		v, err := fetchLatestReleaseVersion(ctx)
		if err != nil {
			log.Printf("[version] check latest release failed: %v", err)
			return
		}
		feed.SetLatestVersion(v)
	}

	update()
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			update()
		}
	}
}

func fetchLatestReleaseVersion(parent context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "thefeed-server")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("latest release status: %s", resp.Status)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	v := strings.TrimSpace(rel.TagName)
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return "", fmt.Errorf("empty latest release tag")
	}
	return v, nil
}
