package testsuite

// ScriptWorkerSh is a script that sends sys.argv and environ
// back to Box to test Boxes
const ScriptWorkerSh = `#!/usr/bin/env python
import os
import sys

print(' '.join(sys.argv[1:]))
for k, v in os.environ.items():
    print("%s=%s" % (k, v))
`
