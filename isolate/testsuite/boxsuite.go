package testsuite

import (
	"bytes"
	"io"
	"strings"

	"github.com/noxiouz/stout/isolate"

	"golang.org/x/net/context"
	check "gopkg.in/check.v1"
)

// RegisterSuite registers a new suite for a provided box
func RegisterSuite(boxConstructor BoxConstructor, opts isolate.Profile, skipCheck SkipCheck) {
	check.Suite(&BoxSuite{
		Constructor: boxConstructor,
		SkipCheck:   skipCheck,
		opts:        opts,
		ctx:         context.Background(),
	})
}

// SkipCheck returns a reason to skip the suite
type SkipCheck func() (reason string)

// BoxConstructor returns a Box to be tested
type BoxConstructor func(c *check.C) (isolate.Box, error)

// NeverSkip is predifened SkipCheck to mark a tester never skipped
var NeverSkip SkipCheck = func() string { return "" }

// BoxSuite is a suite with specification tests for various Box implementations
type BoxSuite struct {
	Constructor BoxConstructor
	SkipCheck
	isolate.Box
	opts isolate.Profile
	ctx  context.Context
}

// SetUpSuite sets up the gocheck test suite.
func (suite *BoxSuite) SetUpSuite(c *check.C) {
	if reason := suite.SkipCheck(); reason != "" {
		c.Skip(reason)
	}
	b, err := suite.Constructor(c)
	c.Assert(err, check.IsNil)
	suite.Box = b
}

// TestSpawn spool code, spawns special worker.sh to verify if env and args are set correctly and
// output is collected properly
func (suite *BoxSuite) TestSpawn(c *check.C) {
	var (
		ctx = context.Background()

		name       = "worker"
		executable = "worker.sh"
		args       = map[string]string{
			"--uuid":     "some_uuid",
			"--locator":  "127.0.0.1:10053",
			"--endpoint": "/var/run/cocaine.sock",
			"--app":      "appname",
		}
		env = map[string]string{
			"enva": "a",
			"envb": "b",
		}
	)

	err := suite.Box.Spool(ctx, name, suite.opts)
	c.Assert(err, check.IsNil)

	pr, err := suite.Box.Spawn(ctx, suite.opts, name, executable, args, env)
	c.Assert(err, check.IsNil)
	defer pr.Kill()

	// collect output
	first := true
	body := new(bytes.Buffer)
	for inc := range pr.Output() {
		c.Assert(inc.Err, check.IsNil)
		if first {
			// first chunk must be empty according to the spec
			// to notify cocaine-runtime that worker has been launched
			first = false
			c.Assert(inc.Data, check.HasLen, 0)
		}

		body.Write(inc.Data)
	}

	// verify args
	unsplittedArgs, err := body.ReadString('\n')
	c.Assert(err, check.IsNil)

	cargs := strings.Split(strings.Trim(unsplittedArgs, "\n"), " ")
	c.Assert(cargs, check.HasLen, len(args)*2)
	for i := 0; i < len(cargs); {
		c.Assert(args[cargs[i]], check.Equals, cargs[i+1])
		i += 2
	}

	// verify env
	cenv := make(map[string]string)
	for {
		envline, err := body.ReadString('\n')
		if err == io.EOF {
			break
		}
		envs := strings.Split(envline[:len(envline)-1], "=")
		c.Assert(envs, check.HasLen, 2)
		cenv[envs[0]] = envs[1]
	}

	for k, v := range env {
		c.Assert(cenv[k], check.Equals, v)
	}
}
