package buildkit

func getRuntimeVariableDataDir() string {
	// Per Apple File System (APFS).
	return "/var/run"
}
