package testsuite

import (
	"bufio"
	"io"
	"strings"
	"time"

	"github.com/noxiouz/stout/isolate"

	"golang.org/x/net/context"
	check "gopkg.in/check.v1"
)

// RegisterSuite registers a new suite for a provided box
func RegisterSuite(boxConstructor BoxConstructor, newprofile ProfileFactory, skipCheck SkipCheck) {
	check.Suite(&BoxSuite{
		Constructor: boxConstructor,
		SkipCheck:   skipCheck,
		newprofile:  newprofile,
		ctx:         context.Background(),
	})
}

// ProfileFactory returns new RawProfile for the box
type ProfileFactory func(c *check.C) isolate.RawProfile

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
	newprofile ProfileFactory
	ctx        context.Context
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

// TearDownSuite closes the Box
func (suite *BoxSuite) TearDownSuite(c *check.C) {
	if suite.Box != nil {
		suite.Box.Close()
	}
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

	err := suite.Box.Spool(ctx, name, suite.newprofile(c))
	c.Assert(err, check.IsNil)

	config := isolate.SpawnConfig{
		Opts:       suite.newprofile(c),
		Name:       name,
		Executable: executable,
		Args:       args,
		Env:        env,
	}

	rd, wr := io.Pipe()
	go func() {
		defer wr.CloseWithError(io.EOF)
		pr, err := suite.Box.Spawn(ctx, config, wr)
		data, err := suite.Box.Inspect(ctx, "some_uuid")
		c.Check(err, check.IsNil)
		c.Check(data, check.Not(check.HasLen), 0)
		c.Assert(err, check.IsNil)
		time.Sleep(10 * time.Second)
		pr.Kill()
	}()

	br := bufio.NewReader(rd)
	// verify args
	unsplittedArgs, err := br.ReadString('\n')
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
		envline, err := br.ReadString('\n')
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
