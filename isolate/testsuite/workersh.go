package testsuite

// ScriptWorkerSh is a script that sends sys.argv and environ
// back to Box to test Boxes
const ScriptWorkerSh = `#!/usr/bin/env bash
echo $@
printenv
`
