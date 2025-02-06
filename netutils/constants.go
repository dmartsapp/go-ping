package netutils

const (
	_DEFAULT_TTL                = 1000
	_DEFAULT_MIN_PING_COUNT     = 4
	_DEFAULT_MAX_PING_COUNT     = _DEFAULT_MIN_PING_COUNT * 250
	_DEFAULT_PAYLOAD_SIZE       = 4
	_DEFAULT_MTU                = 1500
	_DEFAULT_MAX_PAYLOAD_SIZE   = _DEFAULT_MTU - 16 - 20 - 16 // MTU - 16 bytes_of_src_datetime - 20 bytes_of_icmp_header - 16 bytes_of_checksum
	_DEFAULT_NETWORK            = "ip4"
	_DEFAULT_RESOLVE_TIMEOUT_MS = 5000
	_DEFAULT_PING_DELAY_MS      = 1000
	_DEFAULT_MAX_DELAY          = 10000
	_DEFAULT_LISTEN_ADDRESS     = "0.0.0.0"
)
