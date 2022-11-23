package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

/** Interface for User Delegation Credential (udc) Providers */
type UdcProvider interface {
	/** Get a udc for a given storage account, that is valid for time.Duration */
	UserDelegationCredential(
		string,
		time.Duration,
		bool,
	) (service.UserDelegationCredential, error)
}

type cacheEntry struct {
	value  service.UserDelegationCredential
	expiry time.Time
}

/** A caching User Delegation Credential (udc) provider
 *
 * This provider implements the UdcProvider interface. It provides user
 * delegation credentials for any storage account that the user has read access
 * and the "Storage Blob Delegator" role on. The user in this case is a service
 * principal.
 *
 * Fetching new user delegation credential can be quite time-consuming as it
 * involves a request to azure. Hence credentials are cached between 
 * invocations. Credentials are cached for ttl (time-to-live) time before a 
 * fresh one is fetched from azure.
 */
type udcCachingProvider struct {
	credential azcore.TokenCredential
	mutex      sync.Mutex
	ttl        time.Duration
	cache      map[string]cacheEntry
}

/** Fetch a fresh udc from azure
 *
 * The new credential is valid for either *minValidity* or ttl, whichever one
 * is the biggest. The returned cacheEntry, on the other hand, will always be
 * valid for u.ttl time.
 */ 
func (u *udcCachingProvider) getNewUserDelegationCredential(
	storageAccount string,
	minValidity    time.Duration,
) (cacheEntry, error) {
	client, err := service.NewClient(
		fmt.Sprintf("https://%s.blob.core.windows.net/", storageAccount),
		u.credential,
		&service.ClientOptions{},
	)
	if err != nil {
		return cacheEntry{}, err
	}

	keyExpiry := time.Now().UTC()
	if minValidity > u.ttl {
		keyExpiry = keyExpiry.Add(minValidity)
	} else {
		keyExpiry = keyExpiry.Add(u.ttl)
	}

	keyStart  := time.Now().UTC().Add(-10 * time.Second)
	keyInfo := service.KeyInfo{
		Start:  to.Ptr( keyStart.Format(sas.TimeFormat)),
		Expiry: to.Ptr(keyExpiry.Format(sas.TimeFormat)),
	}

	udc, err := client.GetUserDelegationCredential(
		context.Background(),
		keyInfo,
		nil,
	)
	if err != nil {
		return cacheEntry{}, err
	}
	
	cacheExpiry := time.Now().UTC().Add(u.ttl)
	return cacheEntry{ value: *udc, expiry: cacheExpiry }, nil
}

func (u *udcCachingProvider) UserDelegationCredential(
	account     string,
	minValidity time.Duration,
	useCachedServicePrincipalKey bool,
) (service.UserDelegationCredential, error) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	udcShouldBeValidTo := time.Now().Add(minValidity).Add(10 * time.Second)
	if useCachedServicePrincipalKey {
		if cached, ok := u.cache[account]; ok {
			if udcShouldBeValidTo.Before(cached.expiry) {
				return cached.value, nil
			}
		}
	}

	start := time.Now()

	cacheEntry, err := u.getNewUserDelegationCredential(account, minValidity)

	duration := time.Since(start)
	fmt.Println("Retrieving UDK with service principal", duration)

	if err != nil {
		return service.UserDelegationCredential{}, err
	}

	u.cache[account] = cacheEntry
	return cacheEntry.value, nil
}

func NewUdcCachingProvider(
	credential azcore.TokenCredential,
	ttl        time.Duration,
) *udcCachingProvider {
	return &udcCachingProvider{
		credential: credential,
		ttl:        ttl,
		cache:      map[string]cacheEntry{},
	}
}

/** Interface for SAS token providers */
type SasTokenProvider interface {
	ContainerSas(string, string, time.Duration, bool, bool, string) (string, error)
}

/** A user delegation SAS token provider
 *
 * This implementation of the SasTokenProvider-interface, provides
 * user delegation sas that is valid for up to *maxDuration* amount
 * of time.
 */
type userDelegationSasProvider struct {
	udcProvider UdcProvider
	maxDuration time.Duration
	tenantID string
	clientID string
	clientSecret string
}

func (u userDelegationSasProvider) ContainerSas(
	account   string,
	container string,
	duration  time.Duration,
	useServicePrincipal bool,
	useCachedServicePrincipalKey bool,
	userAccessToken string,
) (string, error) {
	start := time.Now()

	if duration > u.maxDuration {
		msg := fmt.Sprintf(
			"Maximum sas-token duration is %s, (requested %s)",
			u.maxDuration.String(),
			duration.String(),
		)
		return "", errors.New(msg)
	}

	startTime := time.Now().UTC().Add(time.Second * -10)
	expiryTime := time.Now().UTC().Add(duration)

	// user probably must choose between speed of access and azure-assured-security
	// we might cache service principal delegation keys or not. We might cache user ones

	if !useServicePrincipal {
		
		oboToken, err := getStorageAccountOBOToken(userAccessToken, u.tenantID, u.clientID, u.clientSecret)
		if err != nil {
			return "", err
		}
	
		keyExpiry := time.Now().UTC().Add(duration)
		keyStart  := time.Now().UTC().Add(-10 * time.Second)
		keyInfo := service.KeyInfo{
			Start:  to.Ptr( keyStart.Format(sas.TimeFormat)),
			Expiry: to.Ptr(keyExpiry.Format(sas.TimeFormat)),
		}
		key, err := getUserDelegationKey(keyInfo, oboToken, account)
		if err != nil {
			return "", err
		}

		sas, err := signWithUserDelegation(key, account, container, startTime, expiryTime)
		if err != nil {
			return "", err
		}

		requestDuration := time.Since(start)
		fmt.Println("Creating SAS with user flow: full time", requestDuration)
		return sas, nil

	} else {
		udc, err := u.udcProvider.UserDelegationCredential(account, duration, useCachedServicePrincipalKey)
		if err != nil {
			return "", err
		}
	
		/* The azure sdk is super fragile when it comes to signing sas tokens.
		 * As of version 0.5.1 the only sas version that is correctly signed
		 * is 2020-10-02, but this keeps changing from version to version.
		 * E.g. in v0.5.0 only 2021-06-08 works correctly. Keep this in mind
		 * in case sas signing breaks in the future.
		 */
		values, err := sas.BlobSignatureValues{
			Version:       "2020-10-02",
			Protocol:      sas.ProtocolHTTPS,
			StartTime:     startTime,
			ExpiryTime:    expiryTime,
			Permissions:   to.Ptr(sas.ContainerPermissions{ Read: true }).String(),
			ContainerName: container,
		}.SignWithUserDelegation(&udc)
		if err != nil {
			return "", err
		}

		requestDuration := time.Since(start)
		fmt.Println("Creating SAS with service principal flow: full time", requestDuration)
		
		return values.Encode(), nil
	}
}

func NewUserDelegationSasProvider(
	udcProvider UdcProvider,
	maxDuration time.Duration,
	tenantId string,
	clientID string,
	clientSecret string,
) userDelegationSasProvider {
	return userDelegationSasProvider{
		udcProvider: udcProvider,
		maxDuration: maxDuration,
		tenantID: tenantId,
		clientID: clientID,
		clientSecret: clientSecret,
	}
}
