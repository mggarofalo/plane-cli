package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	selfupdate2 "github.com/mggarofalo/plane-cli/internal/selfupdate"
	"golang.org/x/term"
)

var (
	flagNoUpdateCheck bool

	// updateNotice is set by the background check goroutine if a new version
	// is available. It is read after command execution to print the notice.
	updateNotice     string
	updateNoticeMu   sync.Mutex
	updateCheckDone  = make(chan struct{})
	updateCheckStart sync.Once
)

// startUpdateCheck launches a background goroutine that checks for a newer
// version. The check is skipped when:
//   - --no-update-check flag is set
//   - PLANE_NO_UPDATE_CHECK=1 environment variable is set
//   - stdout is not a TTY (piped/scripted usage)
//   - version is "dev" (development build)
//   - less than 24 hours since the last check
func startUpdateCheck() {
	updateCheckStart.Do(func() {
		go func() {
			defer close(updateCheckDone)

			if flagNoUpdateCheck {
				return
			}
			if os.Getenv("PLANE_NO_UPDATE_CHECK") == "1" {
				return
			}
			if !term.IsTerminal(int(os.Stderr.Fd())) {
				return
			}
			if version == "dev" {
				return
			}
			if !selfupdate2.ShouldCheck() {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			result, err := selfupdate2.CheckForUpdate(ctx, version)
			if err != nil {
				return
			}

			if result.NewVersionAvailable {
				updateNoticeMu.Lock()
				updateNotice = fmt.Sprintf(
					"A new version of plane is available (v%s). Run \"plane update\" to upgrade.",
					result.LatestVersion,
				)
				updateNoticeMu.Unlock()
			}
		}()
	})
}

// printUpdateNotice waits for the background check to finish (up to timeout)
// and prints the notice if one is available.
func printUpdateNotice(timeout time.Duration) {
	timer := time.After(timeout)
	select {
	case <-updateCheckDone:
	case <-timer:
		return
	}

	updateNoticeMu.Lock()
	notice := updateNotice
	updateNoticeMu.Unlock()

	if notice != "" {
		fmt.Fprintln(os.Stderr, notice)
	}
}
