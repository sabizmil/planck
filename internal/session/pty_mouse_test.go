//go:build !race

package session

import (
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
)

// Note: These tests are excluded from race detection because vt.Emulator
// has an internal race between Close() and Read() on its pipe reader field.
// The race is benign (just cleanup ordering) but triggers the race detector.

func TestSendMouseWheel_WithoutMouseMode(t *testing.T) {
	// Create emulator WITHOUT enabling mouse mode
	emu := vt.NewEmulator(80, 24)

	// Start reader FIRST (io.Pipe is unbuffered, so writes block until read)
	done := make(chan int, 1)
	go func() {
		buf := make([]byte, 256)
		n, _ := emu.Read(buf)
		done <- n
	}()

	// Try sending a mouse wheel event - should be a no-op (no mouse mode)
	emu.SendMouse(vt.MouseWheel{
		X:      10,
		Y:      5,
		Button: vt.MouseWheelUp,
	})

	select {
	case n := <-done:
		emu.Close()
		if n > 0 {
			t.Errorf("expected no bytes (no mouse mode), got %d bytes", n)
		}
	case <-time.After(200 * time.Millisecond):
		// Expected: Read blocks because SendMouse wrote nothing (no mouse mode)
		t.Log("CONFIRMED: SendMouse produces NO output when mouse mode is not enabled")
		emu.Close()
		<-done // wait for goroutine to exit
	}
}

func TestSendMouseWheel_WithMouseMode(t *testing.T) {
	// Create emulator and enable mouse modes (simulating what Bubble Tea does)
	emu := vt.NewEmulator(80, 24)

	// Start reader FIRST since io.Pipe is unbuffered
	results := make(chan []byte, 10)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for {
			buf := make([]byte, 256)
			n, err := emu.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				results <- data
			}
			if err != nil {
				return
			}
		}
	}()

	// Simulate the child process enabling mouse tracking by writing the
	// mode-enable sequences to the emulator (as if the child wrote them
	// to its stdout, which we'd read from PTY master and feed to emulator).
	emu.Write([]byte("\x1b[?1000h")) // Enable normal mouse tracking
	emu.Write([]byte("\x1b[?1002h")) // Enable button event tracking
	emu.Write([]byte("\x1b[?1006h")) // Enable SGR encoding

	// Now send a mouse wheel event
	emu.SendMouse(vt.MouseWheel{
		X:      10,
		Y:      5,
		Button: vt.MouseWheelUp,
	})

	// Collect all data from the pipe
	var allData []byte
	timeout := time.After(2 * time.Second)
	for {
		select {
		case data := <-results:
			allData = append(allData, data...)
			// Check if we got our mouse event (look for SGR mouse sequence)
			s := string(allData)
			if len(s) > 0 && containsSGRMouse(s) {
				t.Logf("SUCCESS: Got SGR mouse data: %q (hex: %x)", s, allData)
				emu.Close()
				<-readerDone
				return
			}
		case <-timeout:
			emu.Close()
			<-readerDone
			if len(allData) > 0 {
				t.Logf("Got data but no mouse event: %q (hex: %x)", string(allData), allData)
			}
			t.Fatal("timeout: no mouse event output after enabling mouse mode")
		}
	}
}

func TestSendMouseWheel_X10Encoding(t *testing.T) {
	// Test with only X10 mode (no SGR)
	emu := vt.NewEmulator(80, 24)

	// Start reader FIRST
	results := make(chan []byte, 10)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for {
			buf := make([]byte, 256)
			n, err := emu.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				results <- data
			}
			if err != nil {
				return
			}
		}
	}()

	emu.Write([]byte("\x1b[?1000h")) // Enable normal mouse only (no SGR)

	emu.SendMouse(vt.MouseWheel{
		X:      10,
		Y:      5,
		Button: vt.MouseWheelUp,
	})

	timeout := time.After(2 * time.Second)
	var allData []byte
	for {
		select {
		case data := <-results:
			allData = append(allData, data...)
			// X10 mouse events start with ESC[M
			if containsX10Mouse(allData) {
				t.Logf("SUCCESS: X10 encoding: %d bytes: %q (hex: %x)", len(allData), string(allData), allData)
				emu.Close()
				<-readerDone
				return
			}
		case <-timeout:
			emu.Close()
			<-readerDone
			if len(allData) > 0 {
				t.Logf("Got data but no X10 mouse event: %q (hex: %x)", string(allData), allData)
			}
			t.Fatal("timeout: no X10 mouse event output")
		}
	}
}

func containsSGRMouse(s string) bool {
	// SGR mouse: ESC[<...M or ESC[<...m
	for i := 0; i < len(s)-3; i++ {
		if s[i] == '\x1b' && s[i+1] == '[' && s[i+2] == '<' {
			return true
		}
	}
	return false
}

func containsX10Mouse(data []byte) bool {
	// X10 mouse: ESC[M followed by 3 bytes
	for i := 0; i < len(data)-5; i++ {
		if data[i] == '\x1b' && data[i+1] == '[' && data[i+2] == 'M' {
			return true
		}
	}
	return false
}
