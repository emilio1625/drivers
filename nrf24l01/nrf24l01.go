// Package nrf24l01 implements a driver for the NRF24L01+ transceiver
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
	ErrNRF24InvalidConfig        = errors.New("NRF24 Invalid configuration")
	ErrNRF24InvalidPipe          = errors.New("NRF24 Invalid pipe, valid pipes go from 0 to 5")
	ErrNRF24InvalidAddressLength = errors.New("NRF24 Invalid address length")
)

type SPI interface {
	Tx(w, r []byte) error
	Transfer(b byte) (byte, error)
}

// Device wraps the nrf24l01
type Device struct {
	// pins to use
	bus                   SPI
	ce                    machine.Pin
	csn                   machine.Pin
	irq                   machine.Pin
	addressWidth          int
	address               [5]byte
	channel               byte
	payloadLength         byte
	compatModeEnabled     bool
	dinamicPayloadEnabled byte // bit represent if enabled in that pipe
	autoAckEnabled        byte // bit represent if enabled in that pipe
}

type Config struct {
	Bus      SPI
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
func New(bus SPI, csn, ce machine.Pin) Device {
	return Device{bus: bus, csn: csn, ce: ce}
}

// Configure sets sane defaults for the NRF24 for maximum transmission compatibility
func (d *Device) Configure(cfg Config) error {
	//!!! TODO: put ce Low and csn High
	d.ce.Low()
	d.csn.High()

	// TODO: Set address width
	d.addressWidth = copy(d.address[:], cfg.Address)
	// TODO: set a channel to not get interference from bluetooth or wifi
	d.channel = cfg.Channel

	// TODO: set data rate to 1Mbps
	// TODO: set the power of transmission
	// TODO: set the LNA gain
	// TODO: config as receiver
	// TODO: disable EnhancedShockBurst maybe?
	// TODO: make sure features are enabled of disabled
	return nil
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
// This cancels any automatic retransmission of a packet due to a missing
// acknoledgement.
func (d *Device) PowerDown() {
	d.ce.Low()
	d.ClearRegisterBit(CONFIG, PWR_UP)
}

// PowerUp wakes the device from sleep, the device could take up to 5 ms
// to wake up, so this function sleeps for 5 ms.
func (d *Device) PowerUp() {
	d.SetRegisterBit(CONFIG, PWR_UP)
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

func (d *Device) DataRate() byte {
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
// The argument can be a beetween 0 to 3, for the lowest and maximum
// power respectively
// ? maybe we sould set some constants for this. nah
func (d *Device) SetTXPower(power byte) {
	power = min(power, 3)
	d.UpdateRegister(RF_SETUP, power<<RF_PWR, 0b11<<RF_PWR)
}

func (d *Device) TXPower() byte {
	return (d.ReadRegisterByte(RF_SETUP) >> RF_PWR) & 0b11
}

// ~~~~~~~~~~~ 7. Enhanced Shockburst ~~~~~~~~~~
// For a general idea of how the protocol works see section 7.5 of the product specification

// Address
// see section 7.3.2 & 7.6 of the product specification

// SetRXAddress sets the addresses of the devices from which packets will be received.
// Pipes 0 and 1 can have unique addresses, pipes 2 to 5 share the four most
// significant bytes with pipe 1 and only change the least significant byte. So if
// you are setting the adresses for pipes 2 to 5 make sure you have already set
// a known address for pipe 1
// For more info see section 7.6
func (d *Device) SetRXAddress(pipe byte, address []byte) error {
	if pipe > 5 {
		return ErrNRF24InvalidPipe
	}
	reg := RX_ADDR_P0 + pipe
	if pipe <= 1 {
		if len(address) < d.addressWidth {
			return ErrNRF24InvalidAddressLength
		}
		d.WriteRegister(reg, address[0:d.addressWidth])
	} else {
		d.WriteRegisterByte(reg, address[0])
	}
	return nil
}

// SetAddressWidth sets the width of the address for all the pipes
func (d *Device) SetAddressWidth(width byte) {
	// width beetween 3 and 5 bytes
	width = min(max(width, 3), 5)
	d.addressWidth = int(width)
	// in the SETUP_AW register 1 means 3 bytes addresses and 3 means 5 bytes
	// addresses, therefore the substraction
	d.WriteRegisterByte(SETUP_AW, width-2)
}

// Sets the address of the device to which a packet will be transmitted.
// If AutoAck is enabled this will also change the address of pipe 0.
// For more info see sections  & 7.6 of the product specification
func (d *Device) SetTXAddress(address []byte) error {
	// TODO: write to the TX_ADDR register
	if len(address) < d.addressWidth {
		return ErrNRF24InvalidAddressLength
	}
	d.WriteRegister(TX_ADDR, address)
	// TODO: how to manage autoack enabled?
	// ? maybe we should manage this in other function
	// if d.autoAckEnabled {
	// 	d.WriteRegister(RX_ADDR_P0, address)
	// }
	return nil
}

// Packet Control Fields
// For more info see section 7.3.3 of the product specification

// Payload length

// TODO: we need a function that enables DPL on the sender side
// See section 7.3.4
// func (d *Device) something {
// TODO: A PTX that transmits to a PRX with DPL enabled must have the
// DPL_P0 bit in DYNPD set.
// }

// SetDynamicPayload enables or disables the receiver to accept payloads
// of variable length. Enabling this function also enables AutoAck
// on the corresponding pipe, however disabling this does not disables Auto Ack
// on the pipe.
// For more info see sections 7.3.4 & 7.4.1 of the product specification
func (d *Device) SetDynamicPayload(pipe byte, enable bool) error {
	if pipe > 5 {
		return ErrNRF24InvalidPipe
	}
	if enable {
		d.SetRegisterBit(FEATURE, EN_DPL)
		d.SetAutoAck(pipe, true)
		d.SetRegisterBit(DYNPD, pipe)
	} else {
		rval := d.ClearRegisterBit(DYNPD, pipe)
		if rval == 0 {
			d.ClearRegisterBit(FEATURE, EN_DPL)
		}
	}
	return nil
}

// SetPayloadLength sets the expected payload length on the receiver side.
// On the trasmitter side the payload length is set by the length of the slice
// pass to WritePayload. This is ignored if DynamicPayload is enabled
// in the pipe.
// If the second argument is zero, returns an error
// For more info see section 7.3.4 of the product specification
func (d *Device) SetPayloadLength(pipe, len byte) error {
	if pipe > 5 {
		return ErrNRF24InvalidPipe
	}
	// TODO: validate len
	register := RX_ADDR_P0 + pipe
	d.WriteRegisterByte(register, len)
}

// PayloadLength returns the length of the received payload, if the payload is
// invalid discards the packet and returns 0
// For more info see section
func (d *Device) PayloadLength() byte {
	if !d.dinamicPayloadsEnabled {
		return d.payloadLength
	}
	// TODO: fix error logic
	d.csn.Low()
	d.bus.Transfer(R_RX_PAYLOAD) // ? when is valid this command
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
// Returns an error when trying to set the length to zero while Dinamic Payloads
// or Auto Acknoledgments are active
// For more info see section 7.3.5 of the product specification
func (d *Device) SetCRCLength(len byte) error {
	// TODO: Fix error logic
	var mask byte = 1<<EN_CRC | 1<<CRCO
	if len < 1 {
		// CRC length can only be 0 in compat mode
		if !d.compatModeEnabled {
			return ErrNRF24InvalidConfig
		}
		d.ClearRegisterBit(CONFIG, EN_CRC)
	} else if len == 1 {
		d.UpdateRegister(CONFIG, 1<<EN_CRC, mask)
	} else {
		d.UpdateRegister(CONFIG, 1<<EN_CRC|1<<CRCO, mask)
	}
	return nil
}

// CRCLength returns the configured length of the CRC
// For more info see section 7.3.5 of the product specification
func (d *Device) CRCLength() int {
	rval := d.ReadRegisterByte(CONFIG)
	if rval&(1<<EN_CRC) > 0 { // if CRC enabled
		if rval&(1<<CRCO) > 0 {
			return 2
		}
		return 1
	}
	return 0
}

// Payload

func (d *Device) WritePayload(data []byte, noAck bool) {
	command := W_TX_PAYLOAD
	if noAck {
		command = W_TX_PAYLOAD_NOACK
	}
	d.SendCommand(command, data, nil)
}

// WriteAckPayload
// For more info see section 7.3.4 of the product specification
func (d *Device) WriteAckPayload(pipe byte, data []byte) error {
	// TODO: error when ack payloads or dinamic payloads are disabled
	if pipe > 5 {
		return ErrNRF24InvalidPipe
	}
	command := W_ACK_PAYLOAD | pipe
	d.SendCommand(command, data, nil)
	return nil
}

func ReusePayload() {
	// TODO: implement
}

// ReadPayload reads a payload to a slice of bytes and returns pipe and the length of the payload
func (d *Device) ReadPayload(into []byte) (pipe, n byte) {
	n = d.PayloadLength()
	if n == 0 { // 0 means an invalid payload
		return
	}
	status, _ := d.SendCommand(R_RX_PAYLOAD, nil, into[0:n])
	pipe = (status >> RX_P_NO) & 0b111
	return
}

func (d *Device) FlushTX() {
	d.SendCommand(FLUSH_TX, nil, nil) //? this works? Yas
}

func (d *Device) FlushRX() {
	d.SendCommand(FLUSH_RX, nil, nil) //? this works?
}

// Automatic packet handling

// SetAutoAck enables the receiver to automatically transmit an ack packet
// after a valid packet has arrived on a pipe. On the transmitter side the
// pipe 0 must be configured with the same address as the TXAddress and
// must have AutoAck enabled.
// For more info see sections 7.4.1 & 7.3.3.3 of the product specification
func (d *Device) SetAutoAck(pipe byte, enable bool) error {
	if d.compatModeEnabled {
		return ErrNRF24InvalidConfig
	}
	if pipe > 5 {
		return ErrNRF24InvalidPipe
	}
	if enable {
		d.SetRegisterBit(EN_AA, pipe)
		d.autoAckEnabled |= 1 << pipe
	} else {
		d.ClearRegisterBit(EN_AA, pipe)
		d.autoAckEnabled &^= (1 << pipe)
	}
	return nil
}

// SetAckPayload enables or disables the ability to send a variable length payload
// in the ack packet. This enables Dynamic Payload Length and Auto Acknoledgment,
// however disabling this does not disable neither DPL nor AutoAck
// For more info see section 7.4.1 and note (d) in section 9.1
func (d *Device) SetAckPayload(enable bool) {
	if enable {
		d.SetDynamicPayload(0, true)
		d.WriteRegisterBit(FEATURE, EN_ACK_PAY, 1)
	} else {
		d.WriteRegisterBit(FEATURE, EN_ACK_PAY, 0)
	}
}

// SetMaxRetries sets the number of automatic retransmissions on comunication fail.
// Up to 15 retransmisions can be set. AutoAcknoledgement must be enabled.
// For more info see section 7.4.2 & 7.8 of the product specification
func (d *Device) SetMaxRetries(retries byte) {
	retries = min(retries, 15)
	d.UpdateRegister(SETUP_RETR, retries<<ARC, 0b1111<<ARC)
}

// SetRetryDelay sets the time between the end of the last transmission and the
// start of the retransmission of a packet when the ACK packet is not received.
// The delay is set in steps of 250us, from 0 (250us) to 15 (4000us).
// The minimum delay depends on the length of the payload, 500 us should be long
// enough for any payload length at 1 or 2 Mbps, 1500us is enough for any payload
// at 250kbps.
// When multiple transmitters are sending to the same receiver, you'll probably
// want to increase this so that the transmitters don't block each other.
// AutoAcknoledgement must be enabled.
// For more info see sections 7.4.2, 7.6 & 7.8 of the product specification
func (d *Device) SetRetryDelay(delay byte) {
	delay = min(delay, 15)
	d.UpdateRegister(SETUP_RETR, delay<<ARD, 0b1111<<ARD)
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
	return d.ReadRegisterByte(OBSERVE_TX) & 0b1111
}

// TODO: reread last paragraph of the section 7.5.2

// EnableCompatMode
// This disables AutoAck and sets the retry delay and retransmissions tries to zero
func (d *Device) EnableCompatMode() {
	d.WriteRegisterByte(EN_AA, 0)
	d.WriteRegisterByte(SETUP_RETR, 0)
}

func (d *Device) DisableCompatMode() {}

// 8. Data & Control interface

// SendCommand sends a command and a slice of bytes to the spi bus and reads the
// response to a slice of bytes, returns the status register, returns an error
// if len(data) != len(response)
func (d *Device) SendCommand(command byte, data, response []byte) (byte, error) {
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
	d.csn.High()
	return rval
}

// WriteRegister writes a slice of bytes to a register, returns the status register
func (d *Device) WriteRegister(register byte, data []byte) (byte, error) {
	command := W_REGISTER | (REGISTER_MASK & register)
	status, err := d.SendCommand(command, data, nil)
	return status, err
}

// WriteRegisterByte writes a single byte to a register, returns the status register
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
	return d.UpdateRegister(register, value<<bit, 1<<bit)
}

// SetRegisterBit sets a bit in a register and returns the new value in the register
func (d *Device) SetRegisterBit(register byte, bit byte) byte {
	return d.WriteRegisterBit(register, bit, 1)
}

// ClearRegisterBit clears a bit in a register and returns the new value in the register
func (d *Device) ClearRegisterBit(register byte, bit byte) byte {
	return d.WriteRegisterBit(register, bit, 0)
}

func max(x, y byte) byte {
	if x > y {
		return x
	}
	return y
}

func min(x, y byte) byte {
	if x < y {
		return x
	}
	return y
}
