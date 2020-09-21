package nrf24l01

import (
	"errors"
	"machine"
)

// BBSPI is a bit-bang implementation of SPI protocol for the nrf24l01.
type BBSPI struct {
	SCK   machine.Pin
	SDI   machine.Pin
	SDO   machine.Pin
	Delay uint32
}

var (
	ErrTxInvalidSliceSize = errors.New("SPI write and read slices must be same size")
)

// Configure sets SCK and SDO low, pins must be already configured
func (s *BBSPI) Configure() {
	s.SCK.Low()
	s.SDO.Low()
	if s.Delay == 0 {
		s.Delay = 1
	}
}

// delay ensures that the output pins have changed state
func (s *BBSPI) delay() {
	for i := uint32(0); i < s.Delay; {
		i++
	}
}

// Transfer matches signature of machine.SPI.Transfer() and is used to send and
// receive a single byte.
func (s *BBSPI) Transfer(b byte) (byte, error) {
	var r byte = 0
	s.delay() // small delay after chip select
	for i := uint8(0); i < 8; i++ {
		// write the value to SDO (MSBit first)
		if b&(1<<(7-i)) == 0 {
			s.SDO.Low()
		} else {
			s.SDO.High()
		}
		s.delay()

		// half clock cycle high to start
		s.SCK.High()

		s.delay()

		// read the value from SIO (MSBit First)
		if s.SDI.Get() {
			r |= 1 << (7 - i)
		}

		// half clock cycle low
		s.SCK.Low()
	}

	return r, nil
}

// Tx handles read/write operation for SPI interface. Since SPI is a syncronous write/read
// interface, there must always be the same number of bytes written as bytes read.
// The Tx method knows about this, and offers a few different ways of calling it.
//
// This form sends the bytes in tx buffer, putting the resulting bytes read into the rx buffer.
// Note that the tx and rx buffers must be the same size:
//
// 		spi.Tx(tx, rx)
//
// This form sends the tx buffer, ignoring the result. Useful for sending "commands" that return zeros
// until all the bytes in the command packet have been received:
//
// 		spi.Tx(tx, nil)
//
// This form sends zeros, putting the result into the rx buffer. Good for reading a "result packet":
//
// 		spi.Tx(nil, rx)
//
func (s *BBSPI) Tx(w, r []byte) error {
	switch {
	case w == nil:
		// read only, so write zero and read a result.
		for i := range r {
			r[i], _ = s.Transfer(0)
		}

	case r == nil:
		// write only
		for _, b := range w {
			s.Transfer(b)
		}

	default:
		// write/read
		if len(w) != len(r) {
			return ErrTxInvalidSliceSize
		}

		for i, b := range w {
			r[i], _ = s.Transfer(b)
		}
	}

	return nil
}
