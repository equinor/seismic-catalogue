package sas

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

type UserDelegationKey struct {
	SignedOID     string `xml:"SignedOid"`
	SignedTID     string `xml:"SignedTid"`
	SignedStart   string `xml:"SignedStart"`
	SignedExpiry  string `xml:"SignedExpiry"`
	SignedService string `xml:"SignedService"`
	SignedVersion string `xml:"SignedVersion"`
	Value         string `xml:"Value"`
}

func (key *UserDelegationKey) sign(stringToSign string) (string, error) {
	bytes, _ := base64.StdEncoding.DecodeString(key.Value)

	mac := hmac.New(sha256.New, []byte(bytes))
	_, err := mac.Write([]byte(stringToSign))
	signedMAC := mac.Sum(nil)

	return base64.StdEncoding.EncodeToString(signedMAC), err
}
