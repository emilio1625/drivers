// Package nrf24l01+ implements a driver for the NRF24L01+ transceiver
//
// Datasheet: https://infocenter.nordicsemi.com/pdf/nRF24L01P_PS_v1.0.pdf
//
package nrf24l01

import (
	"errors"
	"machine"
	"time"
)

var (
	ErrNRF24InvalidConfig = errors.New("NRF24 Invalid configuration")
	ErrNRF24InvalidPipe = errors.New("NRF24 Invalid pipe, valid pipes go from 0 to 5")
	ErrNRF24InvalidSliceLength = errors.New("NRF24 Invalid slice length")
)

// Device wraps the nrf24l01
type Device struct {
	// pins to use
	bus machine.SPI
	ce  machine.Pin
	csn machine.Pin
	irq machine.Pin
	addressWidth byte
	address [5]byte
	channel byte
	payloadLength byte
	compatMode bool
	featuresEnabled bool
	dinamicPayloadsEnabled bool
	autoAckEnabled bool
}

type Config struct {
	Bus      machine.SPI
	CEPin    machine.Pin
	CSNPin   machine.Pin
	IRQPin   machine.Pin
	Address  []byte
	DataRate byte
	Channel  byte
	Power    byte
}

// New creates a new nrf24 instance, the spi bus and pins must be already configured
// the operation mode of SPI must be Mode0, MSBit first, 8Mhz max
func New(bus machine.SPI, csn, ce machine.Pin) Device {
	return Device{bus: bus, csn: csn, ce:  ce}
}

// Configure sets sane defaults for the NRF24 for maximum transmission compatibility
func (d *Device) Configure(cfg Config) error {
	d.address := make([]byte, min(len(cfg.Address), 5), 5)
	copy(d.address, cfg.Address)
	d.channel = cfg.Channel

	// TODO: set data rate to 1Mbps
	// TODO: set a channel to not get interference from bluetooth or wifi
	// TODO: set the power of transmission
	// TODO: set the LNA gain
	// TODO: config as receiver
	// TODO: disable EnhancedShockBurst maybe?
	// TODO: make sure features are enabled of disabled
	d.ce.Low()
	d.csn.High()
}

// Status returns the contents of the status register
func (d *Device) Status() byte {
	d.csn.Low()
	status, _ := d.bus.Transfer(NOOP)
	d.csn.High()
	return status
}

// ~~~~~~~~~~ 6. Radio Control ~~~~~~~~~~~~~~~~

// PowerDown puts the device to sleep, the current consumption is minimal.
// This cancels any automatic retransmission of a packet due to a
// missing acknoledgement
func (d *Device) PowerDown() {
	d.ce.Low()
	d.clearRegisterBit(CONFIG, PWR_UP)
}

// PowerUp wakes the device from sleep, the device could take up to 5 ms
// to wake up, so this function sleeps for 5 ms.
func (d *Device) PowerUp() {
	d.setRegisterBit(CONFIG, PWR_UP)
	time.Sleep(5 * time.Millisecond)
}

// SetDataRate sets the data rate of transmission.
// The argument can be 0 or 1 for 1Mbps or 2Mbps respectively. 1Mbps gives 3dB 
// better receiver sensitivity compared to 2Mbps. Higher data rate means lower
// average current consumption and reduced probability of on-air collisions.
// For compatibility with older radios the data rate should be set to 1Mbps.
func (d *Device) SetDataRate(rate byte) {
	// TODO: change this to comply with the nrf24l01+ product specification
	rate = min(rate, 1)
	d.WriteRegisterBit(RF_SETUP, RF_DR, rate)
}

func (d *Device) DataRate() {
	// TODO: change this to comply with the nrf24l01+ product specification
	return (d.ReadRegisterByte(RF_SETUP) >> RF_DR) & 1
}

// SetChannel sets the channel frequency of transmission (max 125). A transmitter
// and a receiver must be programmed with the same channel frequency to be able to
// communicate with each other.
// WiFi and Bluetooth overlap up to channel 83, so chosing a higher channel is 
// recommended.
func (d *Device) SetChannel(channel byte) {
	channel = min(channel, 125)
	d.WriteRegisterByte(RF_CH, channel)
}

// Channel returns the channel currently configured in the radio
func (d *Device) Channel() byte {
	return d.ReadRegisterByte(RF_CH)
}

// SetTXPower sets the output power for the nRF24L01 power amplifier.
// The argument can be a number for 0 to 3, for the lowest and maximum
// power respectively
// ? maybe we sould set some constants for this. nah
func (d *Device) SetTXPower(power byte) {
	power = min(power, 3)
	d.UpdateRegister(RF_SETUP, power << RF_PWR, 0b11 << RF_PWR)
}

func (d *Device) TXPower() {
	return (d.ReadRegisterByte(RF_SETUP) >> RF_PWR) & 0b11
}

//! Deprecated for nrf24l01+
// // EnableLNAGain enables the low noise amplifier (enabled by default)
// func (d *Device) EnableLNAGain() {
// 	d.setRegisterBit(RF_SETUP, LNA_HCURR)
// }

//! Deprecated for nrf24l01+
// // DisableLNAGain disables the low noise amplifier. The LNA gain makes it
// // possible to reduce the current consumption in RX mode by 0.8mA at the cost
// // of 1.5dB reduction in receiver sensitivity
// func (d *Device) DisableLNAGain() {
// 	// ? this feature is not well documented, it is not clear to me whether
// 	// ? setting this bit reduces or increases the gain. This is impplemented
// 	// ? taking the RF24 library as reference
// 	d.clearRegisterBit(RF_SETUP, LNA_HCURR)
// }

// ~~~~~~~~~~~ 7. Enhanced Shockburst ~~~~~~~~~~
// For a general of how the protocol works see section 7.5 of the product specification

// Address
// see section 7.3.2 & 7.6 of the product specification

// SetRXAddress sets the addresses of the devices from which packets will be received.
// Pipes 0 and 1 can have unique addresses, but pipes 2 to 5 share the most 
// significant bytes with pipe 1 and only change the last byte. So if you are setting
// the adresses for pipes 2 to 5 make sure you have already set a known address for
// pipe 1
func (d *Device) SetRXAddress(pipe byte, address []byte) error {
	// TODO: write to the RX_ADDR_P{0,5} registers
	if pipe > 5 {
		return ErrNRF24InvalidPipe
	}
	reg := RX_ADDR_P0 + pipe
	if pipe <= 1 {
		if len(address) < d.addressWidth {
			return ErrNRF24InvalidSliceLength
		}
		d.WriteRegister(reg, address[0:d.addressWidth])
	} else {
		d.WriteRegisterByte(reg, address[0])
	}
}

// Set the width of the address for all the receivers
func (d *Device) SetAddressWidth(width byte) {
	// width beetween 3 and 5 bytes
	d.addressWidth = min(max(width, 3), 5) // ? maybe not needed
	d.WriteRegisterByte(SETUP_AW, d.addressWidth - 2)
}

// Sets the address of the device to which a packet will be transmitted. If
// AutoAck is enabled this will also change the address of pipe 0.
// For more info see sections  & 7.6 of the product specification
func (d *Device) SetTXAddress(address []byte) error {
	// TODO: write to the TX_ADDR register
	if len(address) < d.addressWidth {
		return ErrNRF24InvalidSliceLength
	}
	d.WriteRegister(TX_ADDR, address)
	if d.autoAckEnabled {
		d.WriteRegister(RX_ADDR_P0, address)
	}
}

// Packet Control Fields
// For more info see section 7.3.3 of the product specification

// Payload length

// EnableDynamicPayloadLength allows a transmitter to send payloads of variable 
// length to the receiver
// For more info see sections 7.3.4 & 7.4.1 of the product specification
func (d *Device) EnableDynamicPayloadLength() {
	// TODO: set the EN_DPL bit on the FEATURE register
	if !d.featuresEnabled {
		d.EnableFeatures()
	}
	d.setRegisterBit(FEATURE, EN_DPL)
}

func (d *Device) DisableDynamicPayloadLength() {
	// TODO: clear the EN_DPL bit on the FEATURE register
	// TODO: Also disable ack payloads (section 7.4.1)
	d.clearRegisterBit(FEATURE, EN_DPL)
}

// SetPayloadLength sets the expected payload length on the receiver side.
// On the trasmitter side the payload length is set by the length of the slice
// pass to WritePayload.
// If the second argument is zero, the pipe is configured to receive payloads of
// variable length
// For more info see section 7.3.4 of the product specification
func (d *Device) SetPayloadLength(pipe, len byte) {
	// TODO: write to the RX_PW_P{0,5} registers
	// TODO: write to the DYNPD register
}

// PayloadLength returns the length of the received payload, if the payload is
// invalid discards the packet and returns 0
func (d *Device) PayloadLength() byte {
	if !d.dinamicPayloadsEnabled {
		return d.payloadLength
	}
	d.csn.Low()
	d.bus.Transfer(R_RX_PAYLOAD)
	len, _ := d.bus.Transfer(NOOP)
	d.csn.High()
	if len > 32 {
		d.FlushRX()
		len = 0
	}
	return len
}

// CRC

// SetCRCLength set the length of the CRC, it can be 0 (only compat mode), 1 or 2 bytes
// For more info see section 7.3.5 of the product specification
func (d *Device) SetCRCLength(len byte) error {
	mask := 1 << EN_CRC | 1 << CRCO
	if len < 1 {
		// CRC length can only be 0 in compat mode
		if !d.compatMode {
			return ErrNRF24InvalidConfig
		}
		d.clearRegisterBit(CONFIG, EN_CRC)
	} else if len == 1 {
		d.UpdateRegister(CONFIG, 1 << EN_CRC, mask)
	} else {
		d.UpdateRegister(CONFIG, 1 << EN_CRC | 1 << CRCO, mask)
	}

}

// CRCLength returns the configured length of the CRC
// For more info see section 7.3.5 of the product specification
func (d *Device) CRCLength() {
	rval = d.ReadRegisterByte(CONFIG)
	if rval & (1 << EN_CRC) > 0 { // if CRC enabled
		if rval & (1 << CRCO) > 0 {
			return 2
		}
		return 1
	}
	return 0
}

// Payload

// For more info see section 7.3.4 of the product specification
func (d *Device) WriteAckPayload(pipe byte, data []byte) error {
	if pipe > 5 {
		return ErrNRF24InvalidPipe
	}
	command := W_ACK_PAYLOAD | pipe
	d.SendCommand(command, data, nil)
}

func (d *Device) WritePayload(data []byte, noAck bool) error {
	if noAck {
		command := W_TX_PAYLOAD_NOACK
	} else {
		command := W_TX_PAYLOAD
	}
	d.SendCommand(command, data, nil)
}

// ReadPayload reads a payload to a slice of bytes and returns the length of the payload
func (d *Device) ReadPayload(into []byte) byte {
	len := d.PayloadLength()
	if len > 0 { // 0 means an invalid payload
		d.SendCommand(R_RX_PAYLOAD, nil, into[0:len])
	}
	return len
}

func (d *Device) FlushTX() {
	d.SendCommand(FLUSH_TX, nil, nil) //? this works?
}

func (d *Device) FlushRX() {
	d.SendCommand(FLUSH_RX, nil, nil) //? this works?
}

// Automatic packet handling

func (d *Device) EnableFeatures() {
	d.SendCommand(ACTIVATE, []byte{0x73}, nil)
}

// Acknowledgment

// SetAuto
// For more info see sections 7.4.2 & 7.3.3.3
func (d *Device) SetAutoAck(pipe byte, enable bool) error {
	if d.compatMode || pipe > 5 {
		return ErrNRF24InvalidConfig
	}
	if (enable) {
		d.setRegisterBit(EN_AA, pipe)
	} else {
		d.clearRegisterBit(EN_AA, pipe)
	}
}

// 
// For more info see sections 7.4.1
func (d *Device) EnableAckPayload(pipe byte) {
	// TODO: implement
	if !d.featuresEnabled {
		d.EnableFeatures()
	}
	d.setRegisterBit(FEATURE, EN_ACK_PAY)
}

// SetMaxReries sets the number of automatic retransmissions on comunication fail.
// Up to 15 retransmisions AutoAcknoledgement must be enabled
// For more info see section 7.4.2 & 7.8 of the product specification
func (d *Device) SetMaxRetries(retries byte) {
	retries = min(retries, 15)
	d.UpdateRegister(SETUP_RETR, retries << ARC, 0xf << ARC)
}

// SetRetryDelay sets the time between the end of the last transmission and the
// start of the retransmission of a packet when the ACK packet is not received. 
// The delay is set in steps of 250us, from 0 (250us) to 15 (4000us).
// The minumum delay depends on the length of the payload, 500 us should be long
// enough for any payload length at 1 or 2 Mbps, 1500us is enough for any payload 
// at 250kbps.
// When multiple transmitters are sending to the same receiver, you'll probably
// want to increase this so that the transmitters don't block each other.
// For more info see sections 7.4.2, 7.6 & 7.8 of the product specification
func (d *Device) SetRetryDelay(delay byte) {
	delay = min(delay, 0xf) // ???
	d.UpdateRegister(SETUP_RETR, delay << ARD, 0xf << ARD)
}

// LostPackets returns the number of lost packets since the last reset of the counter
// For more info see section 7.4.2 of the product specification
func (d *Device) LostPackets() byte {
	return d.ReadRegisterByte(OBSERVE_TX) >> PLOS_CNT
}

// For more info see section 7.4.2 of the product specification
func (d *Device) ResetLostPacketsCounter() {
	d.SetChannel(d.channel)
}

// RetransmissionCount returns the number of retransmissions for the current packet send
func (d *Device) RetrasmissionCount() byte {
	return d.ReadRegisterByte(OBSERVE_TX) & 0x0f
}

// TODO: reread last paragraph of the section 7.5.2

func (d *Device) EnableCompatMode() {}

func (d *Device) DisableCompatMode() {}

// 8. Data & Control interface

// SendCommand sends a command and a slice of bytes to the spi bus and reads the 
// response to a slice of bytes, returns the status register, throws an error
// if len(data) != len(response)
func (d *Device) SendCommand(command byte, data, response []byte) byte, error {
	d.csn.Low()
	status, _ := d.bus.Transfer(command)
	err := d.bus.Tx(data, response)
	d.csn.High()
	return status, err
}

// ReadRegister reads a register into a slice, returns the status register
func (d *Device) ReadRegister(register byte, into []byte) byte {
	command := R_REGISTER | (REGISTER_MASK & register)
	status, _ := d.SendCommand(command, nil, into)
	return status
}

// ReadRegisterByte reads a single byte from a register
func (d *Device) ReadRegisterByte(register byte) byte {
	command := R_REGISTER | (REGISTER_MASK & register)
	d.csn.Low()
	d.bus.Transfer(command)
	rval, _ := d.bus.Transfer(NOOP)
	d.csn.Low()
	return rval
}

// WriteRegister writes a slice of bytes to a register, returns the status register
func (d *Device) WriteRegister(register byte, data []byte) byte {
	command := W_REGISTER | (REGISTER_MASK & register)
	status, _ := d.SendCommand(command, data, nil)
	return status
}

// WriteRegister writes a single byte to a register, returns the status register
func (d *Device) WriteRegisterByte(register, value byte) byte {
	command := W_REGISTER | (REGISTER_MASK & register)
	d.csn.Low()
	status, _ := d.bus.Transfer(command)
	d.bus.Transfer(value)
	d.csn.High()
	return status
}

// UpdateRegister modifies the value of a register acording to the bits set in 
// the mask, replacing them with the bits set in the value parameter, return the
// new value in the register
func (d *Device) UpdateRegister(register, value, mask byte) byte {
	rval := d.ReadRegisterByte(register)
	// clear the bits specified in the mask
	rval &^= mask
	// replace them with value
	rval |= value
	d.WriteRegisterByte(register, rval)
	// return the new value on the register
	return rval
}

// WriteRegisterBit modifies a register bit and returns the new value on the register
func (d *Device) WriteRegisterBit(register, bit, value byte) byte {
	// return the new value on the register
	return d.UpdateRegister(register, value << bit, 1 << bit)
}

// setRegisterBit sets a bit in a register and returns the new value in the register
func (d *Device) setRegisterBit(register byte, bit byte) byte {
	return d.WriteRegisterBit(register, bit, 1)
}

// clearRegisterBit clears a bit in a register and returns the new value in the register
func (d *Device) clearRegisterBit(register byte, bit byte) byte {
	return d.WriteRegisterBit(register, bit, 0)
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
