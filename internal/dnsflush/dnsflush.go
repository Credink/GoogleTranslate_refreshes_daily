// Package dnsflush clears the OS DNS resolver cache after hosts file changes.
package dnsflush

import "log"

// AfterHostsWrite runs a best-effort DNS cache flush for the current OS.
// Failures are logged and ignored — hosts writes must not fail because of flush.
func AfterHostsWrite() {
	if err := flush(); err != nil {
		log.Printf("dnsflush: %v", err)
		return
	}
	log.Printf("dnsflush: ok")
}
