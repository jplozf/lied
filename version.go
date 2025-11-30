package main

var (
	InternalVersionMajor = "0"
	CommitCount          = "unknown"
	CommitHash           = "unknown"
	Version              string
)

// *********************************************************
// init()
// *********************************************************
func init() {
	Version = InternalVersionMajor + "." + CommitCount + "-" + CommitHash
}
