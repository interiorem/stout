package testsuite

// ScriptWorkerSh is a script that sends sys.argv and environ
// back to Box to test Boxes
const ScriptWorkerSh = `#!/usr/bin/env bash
# we need this to give an isolation system the gap to attach
sleep 5
echo $@
printenv
`
