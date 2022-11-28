package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func GetStorageAccountOBOToken(accessToken string, tenantID string, clientID string, clientSecret string) (string, error) {
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

	client := &http.Client{}
	response, err := client.Do(request)

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
		return "", errors.New(fmt.Sprintln("error while retrieving obo token", response.Status))
	}

	if access_token, ok := target["access_token"].(string); ok {
		return access_token, nil
	} else {
		return "", errors.New("access token is not found in token body")
	}

}
