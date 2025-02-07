package netutils

const (
	_DEFAULT_TTL                    int    = 1000
	_DEFAULT_MIN_PING_COUNT         int    = 4
	_DEFAULT_MAX_PING_COUNT         int    = _DEFAULT_MIN_PING_COUNT * 250
	_DEFAULT_PAYLOAD_SIZE           int    = 4
	_DEFAULT_MTU                    int    = 1500
	_DEFAULT_MAX_PAYLOAD_SIZE       int    = _DEFAULT_MTU - 16 - 20 - 16 // MTU - 16 bytes_of_src_datetime - 20 bytes_of_icmp_header - 16 bytes_of_checksum
	_DEFAULT_NETWORK                string = "ip4"
	_DEFAULT_RESOLVE_TIMEOUT_MS     int    = 5000
	_DEFAULT_PING_DELAY_MS          int    = 1000
	_DEFAULT_MAX_DELAY_MS           int    = _DEFAULT_PING_DELAY_MS * 10
	_DEFAULT_LISTEN_ADDRESS         string = "0.0.0.0"
	_DEFAULT_HTTP_CLIENT_USER_AGENT string = "dmarts.app-http-v0.1"
)
