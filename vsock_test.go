package vsock

import (
	"bytes"
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

func TestAddr_fileName(t *testing.T) {
	tests := []struct {
		cid  uint32
		port uint32
		s    string
	}{
		{
			cid:  Hypervisor,
			port: 10,
			s:    "vsock:hypervisor(0):10",
		},
		{
			cid:  loopback,
			port: 20,
			s:    "vsock:loopback(1):20",
		},
		{
			cid:  Host,
			port: 30,
			s:    "vsock:host(2):30",
		},
		{
			cid:  3,
			port: 40,
			s:    "vsock:vm(3):40",
		},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			addr := &Addr{
				ContextID: tt.cid,
				Port:      tt.port,
			}

			if want, got := tt.s, addr.fileName(); want != got {
				t.Fatalf("unexpected file name:\n- want: %q\n-  got: %q",
					want, got)
			}
		})
	}
}

func TestListenLocal(t *testing.T) {
	const listenPort = 1025

	var	testData = make([]byte, 1024)
	_, err := rand.Read(testData)
	if err != nil {
		t.Fatalf("unable to prepare test data: %v", err)
	}

	// server must listen before client dial
	listener, err := ListenLocal(listenPort)
	if err != nil {
		t.Fatalf("unable to listen on local CID: %v", err)
	}
	defer listener.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	errChan := make(chan string, 2)
	go func() {
		defer wg.Done()

		conn, err := listener.Accept()
		if err != nil {
			errChan <- fmt.Sprintf("unable to accept on local CID: %v",  err)
			return
		}
		defer conn.Close()

		data := make([]byte, 1024)
		_, err = conn.Read(data)
		if err != nil {
			errChan <- fmt.Sprintf("unable to read on local CID: %v",  err)
			return
		}

		if !bytes.Equal(data, testData) {
			errChan <- fmt.Sprintf("read corrupted on local CID: %v",  err)
			return
		}
	}()

	go func() {
		defer wg.Done()

		conn, err := Dial(loopback, listenPort)
		if err != nil {
			errChan <- fmt.Sprintf("unable to dial local CID(%d): %v", loopback, err)
			return
		}
		defer conn.Close()

		_, err = conn.Write(testData)
		if err != nil {
			errChan <- fmt.Sprintf("unable to write to local CID: %v",  err)
			return
		}
	}()

	wg.Wait()

	select {
	case err := <- errChan:
		t.Fatal(err)
	default:
	}

}