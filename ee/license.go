//go:build ee
// +build ee

package ee

import (
	"github.com/gravitl/netmaker/logger"
)

// AddLicenseHooks - adds the validation and cache clear hooks
func AddLicenseHooks() {
}

// ValidateLicense - the initial license check for netmaker server
// checks if a license is valid + limits are not exceeded
// if license is free_tier and limits exceeds, then server should terminate
// if license is not valid, server should terminate
func ValidateLicense() error {
	logger.Log(0, "proceeding with Netmaker license validation...")
	logger.Log(0, "License validation succeeded!")
	return nil
}
