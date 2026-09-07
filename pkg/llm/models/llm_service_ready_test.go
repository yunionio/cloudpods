package models

import (
	"testing"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
)

func TestCrashLoopEpisodeAfter(t *testing.T) {
	cases := []struct {
		name         string
		episodes     int
		inCrashLoop  bool
		status       string
		wantEpisodes int
		wantInCrash  bool
		wantFailed   bool
	}{
		{
			name:         "enter container crash_loop",
			episodes:     0,
			inCrashLoop:  false,
			status:       computeapi.CONTAINER_STATUS_CRASH_LOOP_BACK_OFF,
			wantEpisodes: 1,
			wantInCrash:  true,
			wantFailed:   false,
		},
		{
			name:         "stay in container crash_loop does not increment",
			episodes:     1,
			inCrashLoop:  true,
			status:       computeapi.CONTAINER_STATUS_CRASH_LOOP_BACK_OFF,
			wantEpisodes: 1,
			wantInCrash:  true,
			wantFailed:   false,
		},
		{
			name:         "pod crash_loop while already in episode does not increment",
			episodes:     1,
			inCrashLoop:  true,
			status:       computeapi.POD_STATUS_CRASH_LOOP_BACK_OFF,
			wantEpisodes: 1,
			wantInCrash:  true,
			wantFailed:   false,
		},
		{
			name:         "running leaves crash_loop and keeps count",
			episodes:     1,
			inCrashLoop:  true,
			status:       computeapi.CONTAINER_STATUS_RUNNING,
			wantEpisodes: 1,
			wantInCrash:  false,
			wantFailed:   false,
		},
		{
			name:         "probing leaves crash_loop and keeps count",
			episodes:     2,
			inCrashLoop:  true,
			status:       computeapi.CONTAINER_STATUS_PROBING,
			wantEpisodes: 2,
			wantInCrash:  false,
			wantFailed:   false,
		},
		{
			name:         "third episode fails",
			episodes:     2,
			inCrashLoop:  false,
			status:       computeapi.CONTAINER_STATUS_CRASH_LOOP_BACK_OFF,
			wantEpisodes: 3,
			wantInCrash:  true,
			wantFailed:   true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotEpisodes, gotInCrash, gotFailed := crashLoopEpisodeAfter(tc.episodes, tc.inCrashLoop, tc.status)
			if gotEpisodes != tc.wantEpisodes || gotInCrash != tc.wantInCrash || gotFailed != tc.wantFailed {
				t.Fatalf("crashLoopEpisodeAfter(%d, %v, %q) = (%d, %v, %v), want (%d, %v, %v)",
					tc.episodes, tc.inCrashLoop, tc.status,
					gotEpisodes, gotInCrash, gotFailed,
					tc.wantEpisodes, tc.wantInCrash, tc.wantFailed)
			}
		})
	}
}

func TestCrashLoopEpisodeAfterSingleCrashThenRunning(t *testing.T) {
	episodes, inCrash, failed := crashLoopEpisodeAfter(0, false, computeapi.POD_STATUS_CRASH_LOOP_BACK_OFF)
	if failed || episodes != 1 || !inCrash {
		t.Fatalf("server crash_loop: episodes=%d inCrash=%v failed=%v", episodes, inCrash, failed)
	}
	episodes, inCrash, failed = crashLoopEpisodeAfter(episodes, inCrash, computeapi.CONTAINER_STATUS_CRASH_LOOP_BACK_OFF)
	if failed || episodes != 1 || !inCrash {
		t.Fatalf("same crash still in loop: episodes=%d inCrash=%v failed=%v", episodes, inCrash, failed)
	}
	episodes, inCrash, failed = crashLoopEpisodeAfter(episodes, inCrash, computeapi.CONTAINER_STATUS_CRASH_LOOP_BACK_OFF)
	if failed || episodes != 1 {
		t.Fatalf("poll again should not fail: episodes=%d failed=%v", episodes, failed)
	}
	episodes, inCrash, failed = crashLoopEpisodeAfter(episodes, inCrash, computeapi.CONTAINER_STATUS_RUNNING)
	if failed || episodes != 1 || inCrash {
		t.Fatalf("recover to running: episodes=%d inCrash=%v failed=%v", episodes, inCrash, failed)
	}
}

func TestCrashLoopEpisodeAfterThreeCyclesFail(t *testing.T) {
	episodes, inCrash, failed := 0, false, false
	for cycle := 1; cycle <= llmCrashLoopFailThreshold; cycle++ {
		episodes, inCrash, failed = crashLoopEpisodeAfter(episodes, inCrash, computeapi.CONTAINER_STATUS_CRASH_LOOP_BACK_OFF)
		if cycle < llmCrashLoopFailThreshold && failed {
			t.Fatalf("cycle %d should not fail: episodes=%d", cycle, episodes)
		}
		if cycle < llmCrashLoopFailThreshold {
			episodes, inCrash, failed = crashLoopEpisodeAfter(episodes, inCrash, computeapi.CONTAINER_STATUS_PROBING)
			if failed || inCrash {
				t.Fatalf("recover after cycle %d: episodes=%d inCrash=%v failed=%v", cycle, episodes, inCrash, failed)
			}
		}
	}
	if !failed || episodes != llmCrashLoopFailThreshold {
		t.Fatalf("third crash cycle: episodes=%d failed=%v", episodes, failed)
	}
}
