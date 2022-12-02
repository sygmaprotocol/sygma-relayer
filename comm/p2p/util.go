// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package p2p

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ReadStream reads data from the given stream
func ReadStream(r *bufio.Reader) ([]byte, error) {
	msg, err := r.ReadString('\n')
	if err != nil {
		return []byte{}, err
	}

	if msg == "" {
		return []byte{}, fmt.Errorf("end of stream reached")
	}

	msg = strings.Trim(msg, "\n")
	return []byte(msg), nil
}

// WriteStream writes the message to stream
func WriteStream(msg []byte, w *bufio.Writer) error {
	_, err := w.WriteString(fmt.Sprintf("%s\n", string(msg[:])))
	if err != nil {
		return err
	}

	err = w.Flush()
	if err != nil {
		return fmt.Errorf("fail to flush stream: %w", err)
	}
	return nil
}

// SendError passes error to errChan if possible, if not drops error
func SendError(errChan chan error, err error, peer peer.ID) {
	err = &comm.CommunicationError{
		Err:  err,
		Peer: peer,
	}
	select {
	case errChan <- err:
		// error sent
	default:
		// error dropped
	}
}
