package main

var (
	InternalVersionMajor = "0"
	CommitCount          = "unknown"
	CommitHash           = "unknown"
	Version              string
)

func init() {
	Version = InternalVersionMajor + "." + CommitCount + "-" + CommitHash
}