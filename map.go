package sandboxfs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type mapRequest struct {
	Mapping  string
	Target   string
	Writable bool
}

func (sfs *Sandboxfs) mapHandler(requests <-chan []byte) {
	for {
		select {
		case mr := <-requests:
			// write mapping to the sandboxfs stdin
			fmt.Fprintf(
				sfs.stdin,
				`[{"Map": %s}]`,
				string(mr),
			)
			fmt.Fprint(sfs.stdin, "\n\n")
		case <-sfs.ctx.Done():
			return
		}
	}
}

func (sfs *Sandboxfs) stdoutHandler() {
	scanner := bufio.NewScanner(sfs.stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "Done" {
			// sandboxfs completed a map request
			sfs.outstandingRequests.Done()
		} else {
			// sandboxfs said something that we don't understand
			log.Println("[sandboxfs]", line)
		}
	}
}

// Map sends a mapping request to sandboxfs.
func (sfs *Sandboxfs) Map(mapping, target string, writable bool) error {
	buf, err := json.Marshal(&mapRequest{
		Mapping:  mapping,
		Target:   target,
		Writable: writable,
	})
	if err != nil {
		return fmt.Errorf("marshaling MapRequest: %v", err)
	}

	sfs.outstandingRequests.Add(1)
	sfs.requests <- buf

	return nil
}

// WaitUntilReady waits until all map requests have been processed.
func (sfs *Sandboxfs) WaitUntilReady(timeout time.Duration) error {
	ready := make(chan bool)
	go func() {
		sfs.outstandingRequests.Wait()
		close(ready)
	}()

	select {
	case <-ready:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timed out waiting for sandboxfs to become ready")
	}
}
