package sas

import (
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/equinor/seismic-catalogue/api/internal/auth"
)

/** Interface for User Delegation Key (udk) Providers */
type UdkProvider interface {
	/** Get a udk for a given storage account, that is valid for time.Duration */
	UserDelegationKey(
		auth.User,
		*resource,
		time.Duration,
	) (UserDelegationKey, error)
}

/* Cache entry for user's key to some storage account*/
type cacheStorageAccountKeyEntry struct {
	value  UserDelegationKey
	expiry time.Time
}

/* Cache entry for one user*/
type cacheUserEntry struct {
	/*storageAccountName: entry*/
	keys map[string]cacheStorageAccountKeyEntry
}

/** A caching User Delegation Key (udk) provider
 *
 * This provider implements the UdkProvider interface. It provides user
 * delegation keys for any storage account that the user has read access
 * and the "Storage Blob Delegator" role on.
 *
 * Fetching new user delegation keys can be quite time-consuming as it
 * involves several requests to azure. Hence keys are cached between
 * invocations. Key is cached for ttl (time-to-live) time before a
 * fresh one is fetched from azure.
 */
type udkCachingProvider struct {
	tenantID     string
	clientID     string
	clientSecret string
	mutex        sync.Mutex
	ttl          time.Duration
	/*userOID: entry*/
	userKeyCache map[string]cacheUserEntry
}

/** Fetch a fresh udk from azure
 *
 * The new key is valid for either *minValidity* or ttl, whichever one
 * is the biggest. The returned cacheEntry, on the other hand, will always be
 * valid for u.ttl time.
 */
func (u *udkCachingProvider) fetchNewUserDelegationKey(
	accessToken string,
	resource *resource,
	minValidity time.Duration,
) (cacheStorageAccountKeyEntry, error) {

	keyExpiry := time.Now().UTC()
	if minValidity > u.ttl {
		keyExpiry = keyExpiry.Add(minValidity)
	} else {
		keyExpiry = keyExpiry.Add(u.ttl)
	}

	keyStart := time.Now().UTC().Add(-10 * time.Second)
	keyInfo := service.KeyInfo{
		Start:  to.Ptr(keyStart.Format(sas.TimeFormat)),
		Expiry: to.Ptr(keyExpiry.Format(sas.TimeFormat)),
	}

	oboToken, err := auth.GetStorageAccountOBOToken(accessToken, u.tenantID, u.clientID, u.clientSecret)
	if err != nil {
		return cacheStorageAccountKeyEntry{}, err
	}

	udk, err := resource.getUserDelegationKey(keyInfo, oboToken)
	if err != nil {
		return cacheStorageAccountKeyEntry{}, err
	}

	cacheExpiry := time.Now().UTC().Add(u.ttl)
	return cacheStorageAccountKeyEntry{value: udk, expiry: cacheExpiry}, nil
}

func (u *udkCachingProvider) UserDelegationKey(
	user auth.User,
	resource *resource,
	minValidity time.Duration,
) (UserDelegationKey, error) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	udcShouldBeValidTo := time.Now().Add(minValidity).Add(10 * time.Second)

	cachedUser, userFound := u.userKeyCache[user.OID]
	if userFound {
		cachedStorageAccountKey, storageAccountFound := cachedUser.keys[resource.storageAccountName]
		if storageAccountFound {
			if udcShouldBeValidTo.Before(cachedStorageAccountKey.expiry) {
				return cachedStorageAccountKey.value, nil
			}
		}
	} else {
		u.userKeyCache[user.OID] = cacheUserEntry{
			keys: map[string]cacheStorageAccountKeyEntry{},
		}
	}

	cacheEntry, err := u.fetchNewUserDelegationKey(user.AccessToken, resource, minValidity)
	if err != nil {
		return UserDelegationKey{}, err
	}

	u.userKeyCache[user.OID].keys[resource.storageAccountName] = cacheEntry
	return cacheEntry.value, nil
}

func NewUdcCachingProvider(
	tenantID string,
	clientID string,
	clientSecret string,
	ttl time.Duration,
) *udkCachingProvider {
	return &udkCachingProvider{
		tenantID:     tenantID,
		clientID:     clientID,
		clientSecret: clientSecret,
		ttl:          ttl,
		userKeyCache: map[string]cacheUserEntry{},
	}
}
