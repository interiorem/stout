package version

// All variables are set by Makefile

// Version is by the latest version in debian/changelog
var Version string

// GitHash is commit hash of the source
var GitHash string

// Build is timestamp of building
var Build string

// GitTag is git tag or git with a distance
var GitTag string
