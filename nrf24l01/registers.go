package nrf24l01

const (
	// SPI Commands for the NRF24L01

	R_REGISTER    byte = 0b0000_0000
	W_REGISTER    byte = 0b0010_0000
	REGISTER_MASK byte = 0b0001_1111

	W_ACK_PAYLOAD      byte = 0b1010_1000
	R_RX_PAYLOAD       byte = 0b0110_0001
	W_TX_PAYLOAD       byte = 0b1010_0000
	FLUSH_TX           byte = 0b1110_0001
	FLUSH_RX           byte = 0b1110_0010
	REUSE_TX_PL        byte = 0b1110_0011
	ACTIVATE           byte = 0b0101_0000
	R_RX_PL_WID        byte = 0b0110_0000
	W_TX_PAYLOAD_NOACK byte = 0b1011_0000
	NOOP               byte = 0b1111_1111

	// Register map addresses and bits

	CONFIG      byte = 0x00        // Register address
	CONFIG_MASK byte = 0b0111_1111 // Register mask
	MASK_RX_DR       = 6           // register bits
	MASK_TX_DS       = 5
	MASK_MAX_RT      = 4
	EN_CRC           = 3
	CRCO             = 2
	PWR_UP           = 1
	PRIM_RX          = 0

	EN_AA   byte = 0x01
	ENAA_P5      = 5
	ENAA_P4      = 4
	ENAA_P3      = 3
	ENAA_P2      = 2
	ENAA_P1      = 1
	ENAA_P0      = 0

	EN_RXADDR byte = 0x02
	ERX_P5         = 5
	ERX_P4         = 4
	ERX_P3         = 3
	ERX_P2         = 2
	ERX_P1         = 1
	ERX_P0         = 0

	SETUP_AW byte = 0x03
	AW            = 0 // bits 1:0

	SETUP_RETR byte = 0x04
	ARD             = 4 // bits 7:4
	ARC             = 0 // bits 3:0

	RF_CH      byte = 0x05
	RF_CH_MASK      = 0b0111_1111

	RF_SETUP  byte = 0x06
	RF_DR_LOW      = 5
	PLL_LOCK       = 4
	RF_DR          = 3
	RF_PWR         = 1 // bits 2:1
	LNA_HCURR      = 0 // deprecated

	STATUS  byte = 0x07
	RX_DR        = 6
	TX_DS        = 5
	MAX_RT       = 4
	RX_P_NO      = 1 // bits 3:1
	TX_FULL      = 0

	OBSERVE_TX byte = 0x08
	PLOS_CNT        = 4 // bits 7:4
	ARC_CNT         = 0 // bits 3:0

	RPD byte = 0x09
	CD  byte = 0x09 // old name

	RX_ADDR_P0 byte = 0x0A
	RX_ADDR_P1 byte = 0x0B
	RX_ADDR_P2 byte = 0x0C
	RX_ADDR_P3 byte = 0x0D
	RX_ADDR_P4 byte = 0x0E
	RX_ADDR_P5 byte = 0x0F

	TX_ADDR byte = 0x10

	RX_PW_P0   byte = 0x11
	RX_PW_P1   byte = 0x12
	RX_PW_P2   byte = 0x13
	RX_PW_P3   byte = 0x14
	RX_PW_P4   byte = 0x15
	RX_PW_P5   byte = 0x16
	RX_PW_MASK byte = 0b0011_1111

	FIFO_STATUS   byte = 0x17
	TX_REUSE           = 6
	TX_FIFO_FULL       = 5
	TX_FIFO_EMPTY      = 4
	RX_FIFO_FULL       = 1
	RX_FIFO_EMPTY      = 0

	DYNPD  byte = 0x1C
	DPL_P5      = 5
	DPL_P4      = 4
	DPL_P3      = 3
	DPL_P2      = 2
	DPL_P1      = 1
	DPL_P0      = 0

	FEATURE    byte = 0x1D
	EN_DPL          = 2
	EN_ACK_PAY      = 1
	EN_DYN_ACK      = 0
)