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
	// TODO maybe set the adress of this device
	address []byte
}

type Configuration struct {
	Bus machine.SPI
	CEPin machine.Pin
	CSNPin machine.Pin
	IRQPin machine.Pin
	Address []byte
	DataRate byte
	Channel byte
	Power byte
}

// New creates a new nrf24 instance, the spi bus and pins must be already configured
// the operation mode of SPI must be Mode0, MSBit first, 8Mhz max
func New(cfg Configuration) (*Device, error) {
	ce.Low()
	csn.High()
	if (len(cfg.Address) > 5) {
		return ErrNR24InvalidAddressLength
	}
	address := make([]byte, len(cfg.Address), 5)
	d := &Device{
		bus: bus,
		csn: csn,
		ce:  ce,
		// max address length: 5 bytes
		address: copy(address, cfg.Address),
	}
	d.SetDataRate(cfg.DataRate)
	d.SetRFChannel(cfg.)

	return d
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

// ~~~~~~~~~~ 6. RadioControl ~~~~~~~~~~~~~~~~

// PowerDown puts the device to sleep, the current consumption is minimal.
// This cancels any automatic retransmission of a package due to a
// missing acknoledgement
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
		mbps = NRF24_2Mbps
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

// SetTXPower sets the output power for the nRF24L01 power amplifier.
// The power argument could be a number for 0 to 3, for the lowest and maximum
// power respectively
// ? maybe we sould set some constants for this
func (d *Device) SetTXPower(power byte) {
	// check if valid power mode
	if power > 3 {
		power = 3
	}
	// read the current register value
	rval := []byte{0}
	d.ReadRegister(RF_SETUP, rval)
	// clear the current value in those bits
	rval[0] &^= 3 << RF_PWR
	// set the new value
	rval[0] |= power << RF_PWR
	rval[0] &= RF_SETUP_MASK
	// write the new value
	d.WriteRegister(RF_SETUP, rval)
}

// EnableLNAGain enables the low noise amplifier (enabled by default)
func (d *Device) EnableLNAGain() {
	d.setRegisterBit(RF_SETUP, LNA_HCURR)
}

// DisableLNAGain disables the low noise amplifier. The LNA gain makes it
// possible to reduce the current consumption in RX mode by 0.8mA at the cost
// of 1.5dB reduction in receiver sensitivity
func (d *Device) DisableLNAGain() {
	// ? this feature is not well documented, it is not clear to me whether
	// ? setting this bit reduces or increases the gain. This is impplemented
	// ? taking the RF24 library
	d.clearRegisterBit(RF_SETUP, LNA_HCURR)
}

// ~~~~~~~~~~~ 7. Enhanced Shockburst ~~~~~~~~~~

// section 7.3.2 7.4.22
func (d *Device) SetAddressLength() {
	// TODO: configure the AW register
}

func (d *Device) SetRXAddress(address, pipe byte) {
	// TODO: write to the RX_ADDR_P{0,5} registers
}

func (d *Device) SetTXAddress(address byte) {
	// TODO: write to the TX_ADDR register
}

// sections 7.3.4 & 7.4.1

func (d *Device) EnableDynamicPayloadLength() {
	// TODO: set (or clear?) the EN_DPL bit on the FEATURE register
}

// SetPayloadSize sets the expected payload size on the receiver side.
// On the trasmitter the payload size is set by the size of the slice send
// to the TX_FIFO
// This setting is ignored when the Dynamic Payload Length is active
// TODO: update this to the name of the fuction used to send a payload
func (d *Device) SetPayloadSize(size, pipe byte) {
	// TODO: write to the RX_PW_P{0,5} registers
}

// sections 7.3.5 & 7.4.2.5
func (d *Device) SetCRCLength(length byte) {
	// TODO: change the CRCO bit in the CONFIG register
}

// section 7.4.2.3
func (d *Device) SetTXNoACK() {
	//? there is a feature that allows the transmitter to tell the receiver of a package
	//? to not send an ack for that package by setting the NO_ACK bit in the package before sending it
	//? is this the function to do that? i dont remember :D
	// TODO: modify the feature register (and/or?) send the W_TX_PAYLOAD_NOACK command
}

// sections 7.5.1
func (d *Device) EnableAutoAcknoledgement(usePayload bool) {
	// TODO: configure the EN_AA register

	// TODO: set the EN_ACK_PAY bit on the FEATURE register
}

// SetMaxReries sets the number of automatic retransmissions on comunication fail.
// AutoAcknoledgement must be enabled
func (d *Device) SetMaxRetries() {
	// TODO: set the number of retries using the SETUP_RETR register
}

// SetRetryDelay sets the time beetween the end of the last transmission and the
// retransmission of a package when the ACK is not received. The minumum delay depends
// on the length of the payload (see section 7.5.2 of the product specification),
// 500 us should be long enough for any payload length.
// When multiple transmitters are sending to the same receiver, you'll probably
// want to increase this so that the transmitters don't block each others
func (d *Device) SetRetryDelay(delay time.Duration) {
	// TODO: set the delay in us between retries
}

// LostPackages returns the number of lost packages since the last reset of the counter
func (d *Device) LostPackages() byte {
	// TODO: implement reading from the PLOS_CNT bits from the OBSERVE_TX register
	return 0
}

func (d *Device) ResetLostPackagesCounter() {
	// TODO: write to the RF_CH register to reset PLOS_CNT in OBSERVE_TX register (pages 65 and 74)
	// TODO: define a field in Device to store the current RF channel and keep it on a known state
}

// RetransmissionCount returns the number of retransmissions for the current package send
func (d *Device) RetrasmissionCount() byte {
	// TODO: read from the ARC_CNT bits from the OBSERVE_TX register
	return 0
}

// TODO: reread last paragraph of the section 7.5.2

func (d *Device) EnableCompatMode() {}

func (d *Device) DisableCompatMode() {}

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
