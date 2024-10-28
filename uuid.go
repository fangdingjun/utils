package utils

import (
	"crypto/rand"
	"fmt"
	"io"
)

// GenerateUUID generate a version 4 UUID string
func GenerateUUID() string {
	tmp := make([]byte, 16)
	io.ReadFull(rand.Reader, tmp)
	//log.Infof("%032x", tmp)
	tmp[6] = (tmp[6] & 0x0f) | 0x40
	tmp[8] = (tmp[8] & 0x3f) | 0x80
	s := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", tmp[:4], tmp[4:6], tmp[6:8], tmp[8:10], tmp[10:])
	return s
}
