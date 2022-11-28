package sas

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

type resource struct {
	storageAccountName string
	containerName      string
}

func newResource(
	storageAccountName string,
	containerName string,
) *resource {
	return &resource{
		storageAccountName: storageAccountName,
		containerName:      containerName,
	}
}

func (s *resource) getUserDelegationKey(keyInfo service.KeyInfo, oboToken string) (UserDelegationKey, error) {
	body, err := xml.Marshal(keyInfo)
	if err != nil {
		return UserDelegationKey{}, err
	}

	url := fmt.Sprintf("https://%s.blob.core.windows.net/", s.storageAccountName)
	var bearer = "Bearer " + oboToken

	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return UserDelegationKey{}, err
	}

	q := request.URL.Query()
	q.Add("restype", "service")
	q.Add("comp", "userdelegationkey")
	request.URL.RawQuery = q.Encode()

	request.Header.Add("Content-Type", "application/xml; charset=utf-8")
	request.Header.Add("Authorization", bearer)
	request.Header.Add("x-ms-version", "2020-04-08")

	client := &http.Client{}
	response, err := client.Do(request)

	if err != nil {
		return UserDelegationKey{}, err
	}

	defer response.Body.Close()
	resBody, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return UserDelegationKey{}, err
	}

	if response.StatusCode != http.StatusOK {
		return UserDelegationKey{}, errors.New(fmt.Sprintln("error retrieving user delegation key", response.Status))
	}

	var udk UserDelegationKey
	err = xml.Unmarshal(resBody, &udk)
	if err != nil {
		return UserDelegationKey{}, err
	}

	return udk, nil
}

func (s *resource) signWithUserDelegation(udk UserDelegationKey, sasStartTime time.Time, sasExpiryTime time.Time) (string, error) {
	permissions := "r"      // "Read"
	resource := "c"         // "container" (see if we should sign directory instead!)
	version := "2020-02-10" // if changed, update string-to-sign accordingly
	canonicalResourceName := fmt.Sprintf("/blob/%s/%s", s.storageAccountName, s.containerName)
	startTime := sasStartTime.Format(time.RFC3339)
	expiryTime := sasExpiryTime.Format(time.RFC3339)
	protocol := "https"

	stringToSign := strings.Join([]string{
		permissions,
		startTime,
		expiryTime,
		canonicalResourceName,
		udk.SignedOID,
		udk.SignedTID,
		udk.SignedStart,
		udk.SignedExpiry,
		udk.SignedService,
		udk.SignedVersion,
		"",
		"",
		"",
		"",
		protocol,
		version,
		resource,
		"",
		"",
		"",
		"",
		"",
		""},
		"\n")

	signature, err := udk.sign(stringToSign)

	m := map[string]string{
		"sv":    version,
		"sr":    resource,
		"st":    startTime,
		"se":    expiryTime,
		"sp":    permissions,
		"spr":   protocol,
		"skoid": udk.SignedOID,
		"sktid": udk.SignedTID,
		"skt":   udk.SignedStart,
		"ske":   udk.SignedExpiry,
		"sks":   udk.SignedService,
		"skv":   udk.SignedVersion,
		"sig":   signature,
	}

	params := url.Values{}
	for k, v := range m {
		params.Add(k, v)
	}
	return params.Encode(), err
}
