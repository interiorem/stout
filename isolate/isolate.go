package isolate

import (
	"io"

	"golang.org/x/net/context"
)

//Isolation interface
type Isolation interface {
	Spool(ctx context.Context, image, tag string) error
	Create(ctx context.Context, profile Profile) (string, error)
	Start(ctx context.Context, container string) error
	Output(ctx context.Context, container string) (io.ReadCloser, error)
	Terminate(ctx context.Context, container string) error
}

//Profile contains basic parametrs for Isolation
type Profile struct {
	Command     string
	NetworkMode string
	Image       string
	WorkingDir  string
	Bind        string
	Env         string
}
