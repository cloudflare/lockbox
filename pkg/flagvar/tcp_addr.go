package flagvar

import "net"

// TCPAddr is a flag.Value for file paths. Returns any errors from net.ResolveTCPAddr.
type TCPAddr struct {
	Network string
	Value   *net.TCPAddr
	Text    string
}

// Help returns a string to include in the flag's help message.
func (t *TCPAddr) Help() string {
	return "TCP address in host:port format"
}

// Set implements flag.Value by parsing the provided address using net.ResolveTCPAddr.
// Any error return is returned by this function.
func (t *TCPAddr) Set(v string) error {
	network := "tcp"
	if t.Network != "" {
		network = t.Network
	}

	tcpAddr, err := net.ResolveTCPAddr(network, v)
	t.Text = v
	t.Value = tcpAddr

	return err
}

// String implements flag.Value by returning the current Text.
func (t *TCPAddr) String() string {
	if t == nil {
		return ""
	}

	return t.Text
}

// Type implements pflag.Value by noting our Value is net.TCPAddr typed.
func (t *TCPAddr) Type() string {
	return "net.TCPAddr"
}
