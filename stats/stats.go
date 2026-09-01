package stats

import (
	"fmt"
	"tg-rss/external/db"
	"time"
)

// Bot run time
var startUpTime time.Time

func GetStartUpTime() time.Time {
	return startUpTime
}

func SetStartUpTime(t time.Time) {
	startUpTime = t
}

func GetRunTime() time.Duration {
	return time.Since(startUpTime)
}

// Feed pull (all) time
var avgFeedsPullDuration time.Duration = 0
var timesLooped uint = 0

func GetAvgFeedsPullDuration() time.Duration {
	return avgFeedsPullDuration
}

func RecordFeedsPullDuration(d time.Duration) {
	// Ongoing average with the last value added
	if timesLooped == 0 {
		avgFeedsPullDuration = d
		timesLooped = 1
	}

	wOld := (float64(timesLooped) / float64(timesLooped+1)) * float64(avgFeedsPullDuration.Nanoseconds())
	wNew := (1.0 / float64(timesLooped+1)) * float64(d)

	avgFeedsPullDuration = time.Duration(wOld) + time.Duration(wNew)
	timesLooped++
}

func FormatStats() (stats string) {
	nUsers, _ := db.GetCount("users")
	nFeeds, _ := db.GetCount("feeds")
	stats = fmt.Sprintf(
		`Statistics:
• Uptime: %s
• Average feed pull time: %.3f µs
• Users: %d
• Feeds: %d`,
		time.Since(startUpTime).Round(time.Second),
		float64(avgFeedsPullDuration.Nanoseconds())/1000.0,
		nUsers,
		nFeeds,
	)

	return
}
