// +build go1.8

// This file is just for the dummy placeholder of serverCompat

package server

type serverCompat struct {
}

func init() {
	if false { // compiler wipes out this code
		_ = Server{}.serverCompat // Antiwarning: megacheck: field serverCompat is unused
	}
}
