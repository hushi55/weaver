package main

import (
	"debug/dwarf"
	"debug/elf"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

const (
	DefaultOpt = "-gcflags="
	NoOpt      = "-gcflags=-l -N"
	OptInl4    = "-gcflags=-l=4"
	OptAllInl4 = "-gcflags=all=-l=4"
)

type builtFile struct {
	*elf.File
	path string
}

func gobuild(t *testing.T, dir string, testfile string, gcflags string) *builtFile {
	src := filepath.Join(dir, "test.go")
	dst := filepath.Join(dir, "out.exe")

	if err := ioutil.WriteFile(src, []byte(testfile), 0666); err != nil {
		t.Fatal(err)
	}

	osenv := os.Getenv("GOOS")
	os.Setenv("GOOS", "linux")
	cmd := exec.Command("/usr/local/go/bin/go", "build", gcflags, "-o", dst, src)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Logf("build: %s\n", b)
		t.Fatalf("build error: %v", err)
	}
	os.Setenv("GOOS", osenv)

	elfFile, err := elf.Open(dst)
	if err != nil {
		t.Fatal(err)
	}
	return &builtFile{elfFile, dst}
}

func TestFieldOverlap(t *testing.T) {
	if runtime.GOOS == "plan9" {
		t.Skip("skipping on plan9; no DWARF symbol table in executables")
	}
	t.Parallel()

	// This test grew out of issue 21094, where specific sudog<T> DWARF types
	// had elem fields set to values instead of pointers.
	const prog = `
package main

var c chan string

func main() {
	c <- "foo"
}
`
	dir, err := ioutil.TempDir("", "TestFieldOverlap")
	if err != nil {
		t.Fatalf("could not create directory: %v", err)
	}
	defer os.RemoveAll(dir)

	f := gobuild(t, dir, prog, NoOpt)
	defer f.Close()

	d, err := f.DWARF()
	if err != nil {
		t.Fatalf("error reading DWARF: %v", err)
	}

	rdr := d.Reader()
	for entry, err := rdr.Next(); entry != nil; entry, err = rdr.Next() {
		if err != nil {
			t.Fatalf("error reading DWARF: %v", err)
		}
		if entry.Tag != dwarf.TagStructType {
			continue
		}
		typ, err := d.Type(entry.Offset)
		if err != nil {
			t.Fatalf("can't read type: %v", err)
		}
		s := typ.(*dwarf.StructType)
		for i := 0; i < len(s.Field); i++ {
			end := s.Field[i].ByteOffset + s.Field[i].Type.Size()
			var limit int64
			if i == len(s.Field)-1 {
				limit = s.Size()
			} else {
				limit = s.Field[i+1].ByteOffset
			}
			if end > limit {
				name := entry.Val(dwarf.AttrName).(string)
				t.Fatalf("field %s.%s overlaps next field", name, s.Field[i].Name)
			}
		}
	}
}
