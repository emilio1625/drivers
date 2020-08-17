// Package nrf24l01 implements a driver for the NRF24L01 transceiver
//
// Datasheet: https://cdn.sparkfun.com/datasheets/Wireless/Nordic/nRF24L01_Product_Specification_v2_0.pdf
//
package nrf24l01

import (
	"errors"
	"machine"
	"time"
)

var (
	ErrNRF24InvalidDataRate       = errors.New("NRF24 Invalid Data Rate")
	ErrNRF24InvalidRegisterConfig = errors.New("NRF24 Invalid register configuration")
)

const (
	NRF24_1Mbps = 0
	NRF24_2Mbps = 1
)

// Device wraps the nrf24l01
type Device struct {
	// pins to use
	bus machine.SPI
	ce  machine.Pin
	csn machine.Pin
	//
}

// New creates a new nrf24 instance, the spi bus and pins must be already configured
// the operation mode of SPI must be Mode0, MSBit first, 8Mhz max
func New(bus machine.SPI, ce, csn machine.Pin) *Device {
	ce.Low()
	csn.High()
	return &Device{
		bus: bus,
		csn: csn,
		ce:  ce,
		// channel:
	}
}

// Configure sets sane defaults for the NRF24 for maximum transmission compatibility
func (d *Device) Configure() {
	// TODO: set data rate to 1Mbps
	// TODO: set a channel to not get interference from bluetooth or wifi
	// TODO: set the power of transmission
	// TODO: set the LNA gain
	// TODO: config as receiver
	// TODO: disable EnhancedShockBurst maybe?
}

// Status returns the contents of the status register
func (d *Device) Status() byte {
	d.csn.Low()
	status, _ := d.bus.Transfer(NOOP)
	d.csn.High()
	return status
}

// PowerDown puts the device to sleep, the current consumption is minimal.
func (d *Device) PowerDown() {
	d.clearRegisterBit(CONFIG, PWR_UP)
}

// PowerUp puts the device in Standby-I mode, the device could take up to 1.5ms
// to wake up, this time decreases to 150us if a external oscillator is used
func (d *Device) PowerUp() {
	d.setRegisterBit(CONFIG, PWR_UP)
	time.Sleep((3 / 2) * time.Millisecond)
}

// SetDataRate sets the data rate of transmission.
// The data rate can be 1Mbps or 2Mbps. The 1Mbps data rate gives 3dB better
// receiver sensitivity compared to 2Mbps. High air data rate means lower
// average current consumption and reduced probability of on-air collisions.
// For compatibility the air data rate should be set to 1Mbps.
func (d *Device) SetDataRate(mbps byte) error {
	if mbps > NRF24_2Mbps {
		return ErrNRF24InvalidDataRate
	}
	d.WriteRegisterBit(RF_SETUP, RF_DR, mbps)
	return nil
}

// SetRFChannel sets the channel frequency of transmission. A transmitter and a
// receiver must be programmed with the same channel frequency to be able to
// communicate with each other.
// ? maybe suggest some good frequencies?
func (d *Device) SetRFChannel(channel byte) {
	channel = (RF_CH_MASK & channel)
	d.WriteRegister(RF_CH, []byte{channel})
}

func (d *Device) SetTXPower() {
	// TODO: write to the RF_PWR bits from the RF_SETUP register
}

func (d *Device) EnableLNAGain() {
	//TODO: set the LNA_HCURR bit from the RF_SETUP register
}

func (d *Device) DisableLNAGain() {
	//TODO: clear the LNA_HCURR bit from the RF_SETUP register
}

// SetMaxReries sets the number of automatic retransmissions on comunication fail
func (d *Device) SetMaxRetries() {
	// TODO: set the number of retries using the SETUP_RETR register
}

func (d *Device) SetRetryDelay() {
	// TODO: set the delay in us between retries
}

// LostPackages returns the number of lost packages since the last reset of the counter
func (d *Device) LostPackages() {
	// TODO: implement reading from the PLOS_CNT bits from the OBSERVE_TX register
}

func (d *Device) ResetLostPackagesCounter() {
	// TODO: write to the RF_CH register to reset PLOS_CNT in OBSERVETX register (pages 65 and 74)
}

func (d *Device) SetRXAddress(address, pipe byte) {
	// TODO: write to the RX_ADDR_P{0,5} registers
}

func (d *Device) SetTXAddress(address byte) {
	// TODO: write to the TX_ADDR register
}

func (d *Device) SetPayloadSize(size, pipe byte) {
	// TODO: write to the RX_PW_P{0,5} registers
}

func (d *Device) EnableDynamicPayload(pipe byte) {

}

// readRegister reads a register
func (d *Device) ReadRegister(register byte, into []byte) byte {
	command := R_REGISTER | (REGISTER_MASK & register)
	d.csn.Low()
	status, _ := d.bus.Transfer(command)
	d.bus.Tx(nil, into)
	d.csn.High()
	return status
}

// WriteRegister writes a slice of bytes to a register
func (d *Device) WriteRegister(register byte, value []byte) byte {
	command := (W_REGISTER | (REGISTER_MASK & register))
	d.csn.Low()
	status, _ := d.bus.Transfer(command)
	d.bus.Tx(value, nil)
	d.csn.High()
	return status
}

// WriteRegisterBit modifies a register bit and returns the new value on the register
func (d *Device) WriteRegisterBit(register, bit, value byte) byte {
	// if register > FEATURE || bit > 7 || value > 1 {
	// 	// ? should we test this if this is an internal function?
	// 	return ErrNRF24InvalidRegisterConfig
	// }
	// read the current value in the register
	rval := []byte{0}
	d.ReadRegister(register, rval)
	// modify the register value
	if value == 1 { // set the bit
		rval[0] |= (1 << bit)
	} else { // clear the bit
		rval[0] &^= (1 << bit)
	}
	// store the new value
	d.WriteRegister(register, rval)
	// return the new value on the register
	return rval[0]
}

// setRegisterBit sets a bit in a register and returns the new value in the register
func (d *Device) setRegisterBit(register byte, bit byte) byte {
	return d.WriteRegisterBit(register, bit, 1)
}

// setRegisterBit clears a bit in a register and returns the new value in the register
func (d *Device) clearRegisterBit(register byte, bit byte) byte {
	return d.WriteRegisterBit(register, bit, 0)
}
