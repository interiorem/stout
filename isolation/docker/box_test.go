package docker

import (
	"os"
	"testing"

	"github.com/noxiouz/stout/isolation"

	"golang.org/x/net/context"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

func init() {
	Suite(&dockerBoxSuite{})
}

type dockerBoxSuite struct{}

func (s *dockerBoxSuite) TestSpool(c *C) {
	var (
		ctx     = context.Background()
		appname = "alpine"
	)
	b, err := NewBox(nil)
	c.Assert(err, IsNil)

	c.Assert(b.Spool(ctx, appname, isolation.Profile{"endpoint": os.Getenv("DOCKER_HOST")}), IsNil)
}
