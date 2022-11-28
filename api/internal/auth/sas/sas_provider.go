package sas

import (
	"errors"
	"fmt"
	"time"

	"github.com/equinor/seismic-catalogue/api/internal/auth"
)

/** Interface for SAS token providers */
type SasTokenProvider interface {
	ContainerSas(string, string, time.Duration, auth.User) (string, error)
}

/** A user delegation SAS token provider
 *
 * This implementation of the SasTokenProvider-interface, provides
 * user delegation sas that is valid for up to *maxDuration* amount
 * of time.
 */
type userDelegationSasProvider struct {
	udkProvider UdkProvider
	maxDuration time.Duration
}

func (u userDelegationSasProvider) ContainerSas(
	account string,
	container string,
	duration time.Duration,
	user auth.User,
) (string, error) {

	if duration > u.maxDuration {
		msg := fmt.Sprintf(
			"Maximum sas-token duration is %s, (requested %s)",
			u.maxDuration.String(),
			duration.String(),
		)
		return "", errors.New(msg)
	}

	resource := newResource(account, container)

	key, err := u.udkProvider.UserDelegationKey(user, resource, duration)
	if err != nil {
		return "", err
	}

	startTime := time.Now().UTC().Add(time.Second * -10)
	expiryTime := time.Now().UTC().Add(duration)

	sas, err := resource.signWithUserDelegation(key, startTime, expiryTime)
	if err != nil {
		return "", err
	}

	return sas, nil
}

func NewUserDelegationSasProvider(
	udkProvider UdkProvider,
	maxDuration time.Duration,
) userDelegationSasProvider {
	return userDelegationSasProvider{
		udkProvider: udkProvider,
		maxDuration: maxDuration,
	}
}
