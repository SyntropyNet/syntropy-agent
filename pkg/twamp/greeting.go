package twamp

import (
	"crypto/rand"
	"fmt"
	"net"
)

type ServerGreeting struct {
	Unused    [12]byte
	Modes     uint32
	Challenge [16]byte
	Salt      [16]byte
	Count     uint32
	MBZ       [12]byte
}

func sendServerGreeting(conn net.Conn) error {
	greeting, err := createServerGreeting(ModeUnauthenticated)
	if err != nil {
		return err
	}

	err = sendMessage(conn, greeting)
	if err != nil {
		return err
	}

	return nil
}

func createServerGreeting(modes uint32) (*ServerGreeting, error) {
	greeting := new(ServerGreeting)

	greeting.Modes = modes
	greeting.Count = 1024

	_, err := rand.Read(greeting.Challenge[:])
	if err != nil {
		return nil, err
	}

	_, err = rand.Read(greeting.Salt[:])
	if err != nil {
		return nil, err
	}

	return greeting, nil
}

func recvServerGreeting(conn net.Conn) error {
	// check for greeting message from TWAMP server
	greeting := new(ServerGreeting)
	err := receiveMessage(conn, greeting)
	if err != nil {
		return err
	}

	// check greeting mode for errors
	switch greeting.Modes {
	case ModeUnauthenticated:
		// The only mode currently supported
		return nil
	case ModeUnspecified:
		return fmt.Errorf("TWAMP server is not interested in communicating with you")
	case ModeAuthenticated:
		return fmt.Errorf("authentication is not currently supported")
	case ModeEncypted:
		return fmt.Errorf("encyption is not currently supported")
	default:
		return fmt.Errorf("unsupported mode 0x%x", greeting.Modes)
	}
}
