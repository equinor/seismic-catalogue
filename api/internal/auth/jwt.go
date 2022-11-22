package auth


import (
	"context"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/auth0/go-jwt-middleware/v2"
	jwkeyset "github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

func NewJwksProvider(issuer string) *jwkeyset.CachingProvider {
	issuerURL, err := url.Parse(issuer)
	if err != nil {
		log.Fatalf("failed to parse the issuer url: %v", err)
	}
	return jwkeyset.NewCachingProvider(issuerURL, 60*time.Minute)
}


/** Custom claims expected to be present in the JWT token */
type rolesClaim struct {
	Roles []string `json:"roles"`
}

/** Custom validation of roles claim
 * 
 * Nothing to validate. There are no hard requirements on which roles should be
 * present. The purpose of the roles claim is to filter queries to the
 * backend, which is dealt with elsewhere.
 */
func (r *rolesClaim) Validate(ctx context.Context) error {
	return nil
}

/** Authentication middleware
 *
 * Check for- and validate access token in the authorization header on
 * incoming requests. If present, the role claim in the token will be extracted
 * and added to the gin context as "jwtRolesClaim" as a []string. If "roles" is
 * not present "jwtRolesClaim" will be set to []string{}
 */
func NewJwtTokenValidator(
	issuer   string,
	audience string,
	keyFunc  func(context.Context) (interface{}, error),
) gin.HandlerFunc {
	rClaim := func() validator.CustomClaims {
		return &rolesClaim{}
	}

	jwtValidator, err := validator.New(
		keyFunc,
		validator.RS256,
		issuer,
		[]string{audience},
		validator.WithCustomClaims(rClaim),
	)

	if err != nil {
		log.Fatalf("failed to setup JWT validator: %v", err)
	}

	return func (ctx *gin.Context) {
		tokenString, err := jwtmiddleware.AuthHeaderTokenExtractor(ctx.Request)
		if err != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if tokenString == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		token, err := jwtValidator.ValidateToken(ctx.Request.Context(), tokenString)
		if err != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims := token.(*validator.ValidatedClaims)
		roles := claims.CustomClaims.(*rolesClaim)
		ctx.Set("jwtRolesClaim", roles.Roles)
		ctx.Set("jwtAccessToken", tokenString)
	}
}
