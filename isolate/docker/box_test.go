package docker

import (
	"os"
	"testing"

	"github.com/noxiouz/stout/isolate"

	"github.com/docker/engine-api/client"
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
		ctx      = context.Background()
		appname  = "alpine"
		endpoint string
	)
	b, err := NewBox(nil)
	c.Assert(err, IsNil)

	if endpoint = os.Getenv("DOCKER_HOST"); endpoint == "" {
		endpoint = client.DefaultDockerHost
	}

	c.Assert(b.Spool(ctx, appname, isolate.Profile{"endpoint": endpoint}), IsNil)
	c.Assert(b.Spool(ctx, appname, isolate.Profile{"endpoint": "balbla"}), Not(IsNil))
}
