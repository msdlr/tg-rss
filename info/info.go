package info

var commit string = "HEAD"
var buildDate string = "BUILD_DATE"
var latestTag string = "LATEST_TAG"
var subVersion = "SUBVERSION"

func GetCommit() string {
	return commit
}

func GetDate() string {
	return buildDate
}

func GetTag() string {
	return latestTag
}

func GetSubversion() string {
	return subVersion
}
