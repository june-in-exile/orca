package handler

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/anthropics/paylock/internal/model"
	"github.com/anthropics/paylock/internal/suiauth"
)

const (
	headerWalletAddress   = "X-Wallet-Address"
	headerWalletSig       = "X-Wallet-Sig"
	headerWalletTimestamp  = "X-Wallet-Timestamp"
	authWindowSeconds     = 60
	maxResourceIDLen      = 128
)

// SigVerifier verifies a Sui wallet signature and returns the derived address.
type SigVerifier interface {
	Verify(message, serializedSig string) (address string, err error)
}

// authResult holds the outcome of wallet auth extraction.
type authResult struct {
	address string
	status  int
	err     string
}

// extractAndVerifyWalletAuth reads the three auth headers, validates the
// timestamp window, builds the canonical message, verifies the signature,
// and returns the verified Sui address.
func extractAndVerifyWalletAuth(r *http.Request, v SigVerifier, clock suiauth.Clock, action, resourceID string) authResult {
	addr := r.Header.Get(headerWalletAddress)
	sig := r.Header.Get(headerWalletSig)
	tsStr := r.Header.Get(headerWalletTimestamp)

	if addr == "" || sig == "" || tsStr == "" {
		return authResult{status: http.StatusUnauthorized, err: "missing authentication headers"}
	}

	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return authResult{status: http.StatusBadRequest, err: "invalid timestamp"}
	}

	now := clock.Now()
	if math.Abs(float64(now-ts)) > authWindowSeconds {
		return authResult{status: http.StatusUnauthorized, err: "request timestamp expired"}
	}

	if len(resourceID) > maxResourceIDLen {
		return authResult{status: http.StatusBadRequest, err: "resource id too long"}
	}

	message := suiauth.MessageForAction(action, resourceID, ts)

	verified, err := v.Verify(message, sig)
	if err != nil {
		return authResult{status: http.StatusUnauthorized, err: "invalid wallet signature"}
	}

	if !strings.EqualFold(verified, addr) {
		return authResult{status: http.StatusUnauthorized, err: fmt.Sprintf("signature does not match claimed address")}
	}

	return authResult{address: verified}
}

// verifyOwnership checks the verified address matches the video's creator.
// If the video has no creator set, always returns true for backwards compatibility.
func verifyOwnership(video *model.Video, verifiedAddr string) bool {
	if video.Creator == "" {
		return true
	}
	return strings.EqualFold(verifiedAddr, video.Creator)
}
