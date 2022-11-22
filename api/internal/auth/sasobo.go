package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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

func getStorageAccountOBOToken(accessToken string, tenantID string, clientID string, clientSecret string) (string, error){
	data := url.Values{}
    data.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
    data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
    data.Set("assertion", accessToken)
	data.Set("scope", "https://storage.azure.com/user_impersonation")
    data.Set("requested_token_use", "on_behalf_of")

	url := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	request, err := http.NewRequest(http.MethodPost, url, strings.NewReader(data.Encode()))
    if err != nil {
        return "", err
    }

	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	start := time.Now()
	client := &http.Client{}
	response, err := client.Do(request)
	duration := time.Since(start)
	fmt.Println("Retrieving OBO", duration)

    if err != nil {
        return "", err
    }

	defer response.Body.Close()

	var target map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&target)
	if err != nil {
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		fmt.Println("HTTP status:", response.StatusCode)
		fmt.Println("error message:", target)
		return "", errors.New(fmt.Sprintln("obo token said something is wrong and returned", response.Status))
	}

	if access_token, ok := target["access_token"].(string); ok {
        return access_token, nil
    } else {
        return "", errors.New("no access token in body??")
    }
	
}


type UserDelegationKey struct {
	SignedOID string `xml:"SignedOid"`
	SignedTID string `xml:"SignedTid"`
	SignedStart string `xml:"SignedStart"`
	SignedExpiry string `xml:"SignedExpiry"`
	SignedService string `xml:"SignedService"`
	SignedVersion string `xml:"SignedVersion"`
	Value string `xml:"Value"`
}

func getUserDelegationKey(keyInfo service.KeyInfo, oboToken string, accountName string) (UserDelegationKey, error) {

	body, err := xml.Marshal(keyInfo)
    if err != nil {
        return UserDelegationKey{}, err
    }

	url := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
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

	start := time.Now()
	client := &http.Client{}
	response, err := client.Do(request)

	duration := time.Since(start)
	fmt.Println("Retrieving UDK with OBO", duration)

    if err != nil {
        return UserDelegationKey{}, err
    }

	defer response.Body.Close()
    resBody, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return UserDelegationKey{}, err
	}

	if response.StatusCode != http.StatusOK {
		fmt.Println(string(body))
		return UserDelegationKey{}, errors.New(fmt.Sprintln("udk said something is wrong and returned", response.Status))
	}

    var udk UserDelegationKey
    err = xml.Unmarshal(resBody, &udk)
	if err != nil {
		return UserDelegationKey{}, err
	}


	return udk, nil
}

func (key *UserDelegationKey) sign(stringToSign string) (string, error) {
	bytes, _ := base64.StdEncoding.DecodeString(key.Value)

	mac := hmac.New(sha256.New, []byte(bytes))
	_, err:= mac.Write([]byte(stringToSign))
	signedMAC := mac.Sum(nil)

	return base64.StdEncoding.EncodeToString(signedMAC), err
}


func signWithUserDelegation(udk UserDelegationKey, account string, container string, sasStartTime time.Time, sasExpiryTime time.Time) (string, error) {
	permissions := "r" // "Read"
	resource := "c" // "container" (see if we should sign directory instead)
	version := "2020-02-10" // if changed, update string-to-sign accordingly
	canonicalResourceName := fmt.Sprintf("/blob/%s/%s", account, container)  
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
        "sv": version,
        "sr": resource,
        "st": startTime,
        "se": expiryTime,
		"sp": permissions,
		"spr": protocol,
		"skoid": udk.SignedOID,
		"sktid": udk.SignedTID,
		"skt": udk.SignedStart,
		"ske": udk.SignedExpiry,
		"sks": udk.SignedService,
		"skv": udk.SignedVersion,
		"sig": signature,	
    }

	params := url.Values{}
    for k, v := range m {
        params.Add(k, v)
    }
    return params.Encode(), err
}


