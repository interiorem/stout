package process

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"context"

	"github.com/noxiouz/stout/isolate"
)

func BenchmarkSpawnSeq(b *testing.B) {
	spoolDir, err := ioutil.TempDir("", "example")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(spoolDir)

	const appName = "echo"

	executable, err := exec.LookPath(appName)
	if err != nil {
		b.Fatalf("LookPath(%s): %v", appName, err)
	}

	os.Mkdir(filepath.Join(spoolDir, appName), 0777)
	ctx := context.Background()
	box, err := NewBox(ctx, isolate.BoxConfig{"spool": spoolDir})
	if err != nil {
		b.Fatal("NewBox: ", err)
	}
	defer box.Close()

	opts, err := isolate.NewRawProfile(&Profile{})
	if err != nil {
		b.Fatalf("unable to prepare profile %v", err)
	}
	config := isolate.SpawnConfig{
		Opts:       opts,
		Name:       appName,
		Executable: executable,
		Args:       map[string]string{},
		Env:        nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p, err := box.Spawn(ctx, config, ioutil.Discard)
		if err != nil {
			b.Fatal("Spawn error: ", err)
		}
		p.Kill()
	}
}

func BenchmarkSpawnParallel(b *testing.B) {
	spoolDir, err := ioutil.TempDir("", "example")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(spoolDir)

	const appName = "echo"

	executable, err := exec.LookPath(appName)
	if err != nil {
		b.Fatalf("LookPath(%s): %v", appName, err)
	}

	os.Mkdir(filepath.Join(spoolDir, appName), 0777)
	ctx := context.Background()
	box, err := NewBox(ctx, isolate.BoxConfig{"spool": spoolDir})
	if err != nil {
		b.Fatal("NewBox: ", err)
	}
	defer box.Close()

	opts, err := isolate.NewRawProfile(&Profile{})
	if err != nil {
		b.Fatalf("unable to prepare profile %v", err)
	}
	config := isolate.SpawnConfig{
		Opts:       opts,
		Name:       appName,
		Executable: executable,
		Args:       map[string]string{},
		Env:        nil,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			p, err := box.Spawn(ctx, config, ioutil.Discard)
			if err != nil {
				b.Fatal("Spawn error: ", err)
			}
			p.Kill()
		}
	})
}
