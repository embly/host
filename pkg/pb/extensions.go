package pb

import (
	"crypto/sha256"
	fmt "fmt"
)

func (p *Port) Protocol() string {
	if p.IsUDP {
		return "udp"
	}
	return "tcp"
}

func (p Port) ConsulName(serviceName, containerName string) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(serviceName))
	_, _ = hash.Write([]byte(containerName))
	_, _ = hash.Write([]byte(fmt.Sprint(p.Number)))
	_, _ = hash.Write([]byte(fmt.Sprint(p.Protocol())))
	return fmt.Sprintf("%x", hash.Sum(nil))[:32]
}
