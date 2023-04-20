package main

import (
	"fmt"
	"io"

	"github.com/kevinburke/nacl"
	"sigs.k8s.io/yaml"
)

type kp struct {
	Private []byte `json:"private"`
	Public  []byte `json:"public"`
}

// KeyPairFromYAMLOrJSON loads a public/private NaCL keypair from a YAML or JSON file.
func KeyPairFromYAMLOrJSON(r io.Reader) (pub, pri nacl.Key, err error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return
	}

	keypair := kp{}
	err = yaml.Unmarshal(data, &keypair, yaml.DisallowUnknownFields)
	if err != nil {
		return
	}

	if len(keypair.Private) != 32 {
		err = fmt.Errorf("incorrect private key length: %d, should be 32", len(keypair.Private))
		return
	}
	if len(keypair.Public) != 32 {
		err = fmt.Errorf("incorrect public key length: %d, should be 32", len(keypair.Public))
		return
	}

	pub = new([nacl.KeySize]byte)
	pri = new([nacl.KeySize]byte)

	copy(pri[:], keypair.Private)
	copy(pub[:], keypair.Public)
	return
}
