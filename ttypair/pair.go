/*
	Manages a tty pair.
*/

package ttypair

import (
	"bytes"
	"code.google.com/p/goplan9/plan9/acme"
	"github.com/rjkroege/winmux/acmebufs"
	"io"
	"log"
)

//type Ttyfd interface {
//	Write(b []byte) error
//	// TODO(rjkroege): Write me
//	// Isecho() bool
//}

type Tty struct {
	acmebufs.Winslice
	cook     bool
	password bool
	fd       io.Writer
	echo *Echo
}

// Creates a Tty object
func New(fd io.Writer, e *Echo) *Tty {
	tty := &Tty{cook: true, password: false, fd: nil}
	tty.fd = fd
	tty.echo = e
	return tty
}

// Returns true if t needs to be treated as a raw tty.
func (t *Tty) Israw() bool {
	// TODO(rjkroege): Pull in isecho.
	return (!t.cook || t.password) /* && !isecho(t.fd0) */
}

// Ships n backspaces to the child.
func (t *Tty) Sendbs(n int) {
	log.Printf("Sendbs %d\n", n)
}

func (t *Tty) Setcook(b bool) {
	t.cook = b
	log.Printf("Setcook to %b\n", b)
}

// Writes the provided buffer to the associated file descriptor.
// Either a single delete character to stop the remote or a single
// command line for the remote shell to execute.
// TODO(rjkroege): Send the provided buffer off to the child process.
//func (t *Tty) Write(b []byte) error {
//	log.Printf("Write: <%s>\n", string(b))
//	return nil
//}

// Adds typing to the buffer associated with this pair at position p0.
func (t *Tty) addtype(typing []byte, p0 int, fromkeyboard bool) {
	// log.Println("Tty.addtype")
	if fromkeyboard && bytes.IndexAny(typing, "\003\007") != -1 {
		log.Println("Tty.addtype: resetting")
		t.Reset()
		return
	}
	t.Addtyping(typing, p0)
}

// Add typing to the buffer or do a bypass write as necessary
// TODO(rjkroege): This is not in the right place.
func (t *Tty) Type(e *acme.Event) {
	if e.Nr > 0 {
		// TODO(rjkroege): Conceivably, I am not shifting the offset enough.
		t.addtype(e.Text, e.Q0, e.C1 == 'K' /* Verify this test. */)
	} else {
		log.Fatal("you've not handled the case where you need to read from acme\n")
		// TODO(rjkroege): Write the acme fetcher...
	}

	if t.Israw() {
		// This deletes the character typed if we have set israw so that
		// raw mode works properly.
		log.Fatal("unsupported raw mode\n")
		//		n = sprint(buf, "#%d,#%d", e->q0, e->q1);
		//		fswrite(afd, buf, n);
		//		fswrite(dfd, "", 0);
		//		q.p -= e->q1 - e->q0;
	}
	t.Sendtype()
	if len(e.Text) > 0 && e.Text[len(e.Text)-1] == '\n' {
		// Not really clear to me what this is for.
		t.cook = true
	}
}

// This is sendtype !raw.
// TODO(rjkroege): Write sendtype_raw or modify this function to do raw mode.
// TODO(rjkroege): this is buffer-oriented. maybe move into winslice?
func (t *Tty) Sendtype() {
	// raw and cooked mode are interleaved. Write cooked mode
	// aside: we should be removing the typed characters in acme right
	// because otherwise the echo will insert them twice... (this block of code)

	ty := t.Typing
	mutated := false
	for p := bytes.IndexAny(ty, "\n\004"); p >= 0; p = bytes.IndexAny(ty, "\n\004") {
		s := ty[0 : p+1]
		t.echo.echoed(s)
		t.fd.Write(s) // Send to the child program
		t.Move(len(s))
		mutated = true
		ty = ty[p+1:]
	}

	// Copy the remaining text to a new slice so that the old backing can
	// get garbage collected.
	if mutated {
		t.Typing = make([]byte, len(ty))
		copy(t.Typing, ty)
	}
}


