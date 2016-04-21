package isolate

const (
	isolateErrCategory = 42
)

const (
	codeBadMsg = 1 + iota
	codeBadProfile
	codeUnknownIsolate
	codeSpoolingFailed
	codeSpawningFailed
	codeOutputError
)

var (
	errBadMsg         = [2]int{isolateErrCategory, codeBadMsg}
	errBadProfile     = [2]int{isolateErrCategory, codeBadProfile}
	errUnknownIsolate = [2]int{isolateErrCategory, codeUnknownIsolate}
	errSpoolingFailed = [2]int{isolateErrCategory, codeSpoolingFailed}
	errSpawningFailed = [2]int{isolateErrCategory, codeSpawningFailed}
	errOutputError    = [2]int{isolateErrCategory, codeOutputError}
)
