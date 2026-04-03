package network

import (
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

// ForwardTunnel creates an SSH port forward: local:remote:host
// Example: "8080:80:example.com" via SSH
func ForwardTunnel(spec string) {
	parts := strings.Split(spec, ":")
	if len(parts) < 3 {
		fmt.Println("  usage: spore tunnel local_port:remote_port:host [ssh_target]")
		return
	}

	sshTarget := "localhost"
	if len(parts) > 3 {
		sshTarget = parts[3]
	}

	cmd := exec.Command("ssh",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-o", "StrictHostKeyChecking=no",
		"-N",
		"-L", fmt.Sprintf("%s:localhost:%s", parts[0], parts[1]),
		sshTarget,
	)

	fmt.Printf("  tunnel: localhost:%s → %s:%s (via %s)\n", parts[0], parts[2], parts[1], sshTarget)
	fmt.Println("  press ctrl+c to stop")

	err := cmd.Run()
	if err != nil {
		fmt.Printf("  tunnel error: %s\n", err)
	}
}

// ReverseTunnel creates a reverse SSH tunnel: remote:local:host
// Lets a remote server connect back to this device
func ReverseTunnel(spec string) {
	parts := strings.Split(spec, ":")
	if len(parts) < 2 {
		fmt.Println("  usage: spore tunnel reverse remote_port:local_port[:ssh_target]")
		return
	}

	sshTarget := "localhost"
	if len(parts) > 2 {
		sshTarget = parts[2]
	}

	cmd := exec.Command("ssh",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ExitOnForwardFailure=yes",
		"-N",
		"-R", fmt.Sprintf("%s:localhost:%s", parts[0], parts[1]),
		sshTarget,
	)

	fmt.Printf("  reverse tunnel: %s:%s → localhost:%s\n", sshTarget, parts[0], parts[1])
	fmt.Println("  press ctrl+c to stop")

	err := cmd.Run()
	if err != nil {
		fmt.Printf("  tunnel error: %s\n", err)
	}
}

// Scan performs a network scan on a target range
func Scan(target string) {
	fmt.Printf("  scanning %s...\n\n", target)

	// Check if nmap is available
	if _, err := exec.LookPath("nmap"); err == nil {
		// Use nmap for comprehensive scan
		cmd := exec.Command("nmap", "-sn", "-T4", target)
		output, err := cmd.CombinedOutput()
		if err == nil {
			fmt.Print(string(output))
			return
		}
	}

	// Fallback: pure Go ping sweep
	pingSweep(target)
}

func pingSweep(cidr string) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		// Try as single IP
		scanHost(cidr)
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var hosts []string

	// Generate all IPs in range
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		target := ip.String()
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			if isAlive(t) {
				mu.Lock()
				hosts = append(hosts, t)
				mu.Unlock()
			}
		}(target)
	}

	wg.Wait()

	sort.Strings(hosts)

	if len(hosts) == 0 {
		fmt.Println("  no hosts found")
		return
	}

	fmt.Printf("  found %d hosts:\n", len(hosts))
	for _, h := range hosts {
		name := reverseLookup(h)
		if name != "" {
			fmt.Printf("  \033[32m●\033[0m %s (%s)\n", h, name)
		} else {
			fmt.Printf("  \033[32m●\033[0m %s\n", h)
		}
	}
}

func scanHost(host string) {
	fmt.Printf("  scanning %s...\n", host)
	commonPorts := []int{22, 80, 443, 8080, 8443, 3000, 5000, 8000, 8422, 9002, 18789}

	for _, port := range commonPorts {
		addr := fmt.Sprintf("%s:%d", host, port)
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			fmt.Printf("  \033[32mopen\033[0m  %d\n", port)
		}
	}
}

func isAlive(host string) bool {
	// Try TCP connect on common ports
	for _, port := range []int{80, 22, 443, 8080} {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

func reverseLookup(ip string) string {
	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		return strings.TrimRight(names[0], ".")
	}
	return ""
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// SOCKSProxy starts a SOCKS5 proxy server
func SOCKSProxy(port string) {
	addr := "0.0.0.0:" + port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("  error: %s\n", err)
		return
	}
	defer listener.Close()

	fmt.Printf("  SOCKS5 proxy on %s\n", addr)
	fmt.Println("  press ctrl+c to stop")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleSOCKS(conn)
	}
}

func handleSOCKS(client net.Conn) {
	defer client.Close()

	// SOCKS5 handshake
	buf := make([]byte, 256)
	n, err := client.Read(buf)
	if err != nil || n < 2 || buf[0] != 0x05 {
		return
	}

	// No auth
	client.Write([]byte{0x05, 0x00})

	// Read request
	n, err = client.Read(buf)
	if err != nil || n < 7 {
		return
	}

	if buf[1] != 0x01 { // CONNECT only
		client.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	var targetAddr string
	switch buf[3] {
	case 0x01: // IPv4
		if n < 10 {
			return
		}
		targetAddr = fmt.Sprintf("%d.%d.%d.%d:%d", buf[4], buf[5], buf[6], buf[7],
			int(buf[8])<<8|int(buf[9]))
	case 0x03: // Domain
		domainLen := int(buf[4])
		if n < 5+domainLen+2 {
			return
		}
		domain := string(buf[5 : 5+domainLen])
		port := int(buf[5+domainLen])<<8 | int(buf[5+domainLen+1])
		targetAddr = fmt.Sprintf("%s:%d", domain, port)
	case 0x04: // IPv6
		if n < 22 {
			return
		}
		ip := net.IP(buf[4:20])
		port := int(buf[20])<<8 | int(buf[21])
		targetAddr = fmt.Sprintf("[%s]:%d", ip, port)
	default:
		return
	}

	// Connect to target
	remote, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		client.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer remote.Close()

	// Success
	client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	// Bidirectional relay
	done := make(chan struct{}, 2)
	relay := func(dst, src net.Conn) {
		buf := make([]byte, 32*1024)
		for {
			src.SetReadDeadline(time.Now().Add(120 * time.Second))
			n, err := src.Read(buf)
			if n > 0 {
				dst.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}

	go relay(remote, client)
	go relay(client, remote)
	<-done
}
