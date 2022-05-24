package service

import (
	"fmt"
	"github.com/golang-jwt/jwt"
	"sync"
)

// HermodAuthenticationConfig lets you set up and define parameters for Hermod's JWT-based authentication system. This is
// highly opinionated, and you don't have to use it! You can also use Hermod's authentication system outside of Hermod
// connections by using the public methods exposed by HermodAuthenticationConfig.
type HermodAuthenticationConfig[K any] struct {
	// SigningMethod must be defined. It's a function that returns true if the signing method of the token is what you
	// want it to be, and false if not.
	SigningMethod func(*jwt.Token) bool
	// either Secret or SecretProvider must be defined
	Secret []byte
	// SecretProvider doesn't need to verify the signing method, this is done automatically
	SecretProvider func(*jwt.Token) ([]byte, error)

	// TokenHydrator must be defined. It returns a custom type based on a map of pre-validated JWT claims.
	TokenHydrator func(jwt.MapClaims) (K, error)

	// UseCache determines whether to cache hydrated tokens. If false, only the raw JWT token will be saved for each
	// connection and must be rehydrated each time your code asks for it. If your hydrated value is unlikely to change during
	// a single connection, set this to true.
	UseCache bool
}

type authProvider struct {
	// goroutines created for endpoint handlers may need to access this concurrently
	sync.RWMutex

	// sessionToken must not be set unless the token is valid and matches the expected secret
	sessionToken  *jwt.Token
	hydratedToken any
	config        *HermodAuthenticationConfig
}

func newAuthProviderFromConfig(authConfig *HermodAuthenticationConfig) *authProvider {
	return &authProvider{
		config: authConfig,
	}
}

// ParseToken parses a JWT token, returning the parsed token object. It will use your specified secret and use the
// HermodAuthenticationConfig.SigningMethod function to determine whether the correct signing method is used.
// It does not check whether the token is valid. You can use jwt.Token.Valid to do this.
func (config *HermodAuthenticationConfig) ParseToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if config.SigningMethod(token) == false {
			return nil, fmt.Errorf("unexpected signing method %v", token.Header["alg"])
		}

		if config.SecretProvider != nil {
			return config.SecretProvider(token)
		}

		return config.Secret, nil
	})
}

// HydrateToken is a convenience method to parse and validate a JWT and to then 'hydrate' the token. It returns both
// the hydrated token and the raw parsed JWT. It will return an error if the JWT is not valid. The type of the hydrated
// token is inferred from the type parameter on HermodAuthenticationConfig.
func (config *HermodAuthenticationConfig[K]) HydrateToken(token string) (K, *jwt.Token, error) {
	parsedToken, err := config.ParseToken(token)
	if err != nil {
		return nil, nil, err
	}

	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {
		// we run this even if we're not going to cache the hydrated token, in case we need to validate the existence of
		// a user (e.g. in a database)
		hydratedToken, err := config.TokenHydrator(claims)
		return hydratedToken, parsedToken, err
	}

	return nil, parsedToken, fmt.Errorf("jwt was not valid")
}

func (provider *authProvider) verifyAndStoreToken(token string) error {
	provider.Lock()
	defer provider.Unlock()

	providerConfig := provider.config
	hydratedToken, parsedToken, err := providerConfig.HydrateToken(token)

	if err != nil {
		return err
	}

	if providerConfig.UseCache {
		provider.hydratedToken = hydratedToken
	}

	provider.sessionToken = parsedToken

	return nil
}

// getHydratedToken returns a hydrated token, either from cache or by calling HermodAuthenticationConfig.TokenHydrate
func (provider *authProvider) getHydratedToken() (any, error) {
	provider.RLock()
	defer provider.RUnlock()

	if provider.hydratedToken != nil && provider.config.UseCache {
		return provider.hydratedToken, nil
	}

	if provider.sessionToken == nil {
		return nil, fmt.Errorf("tried to get hydrated token without a session token being saved")
	}

	if claims, ok := provider.sessionToken.Claims.(jwt.MapClaims); ok {
		return provider.config.TokenHydrator(claims)
	} else {
		return nil, fmt.Errorf("session token could not be parsed as a map of claims")
	}
}

type AuthAPI struct {
	*authProvider
}

func (api *AuthAPI) GetHydratedToken() (any, error) {
	return api.getHydratedToken()
}

func (api *AuthAPI) UpdateToken(token string) error {
	return api.verifyAndStoreToken(token)
}

func setupRequestAuthorization(req *Request, token string, config *HermodConfig) error {
	if config.AuthenticationConfig == nil {
		return fmt.Errorf("authentication hasn't been configured")
	}

	if req.Auth != nil {
		err := req.Auth.UpdateToken(token)
		return err
	}

	auth := newAuthProviderFromConfig(config.AuthenticationConfig)
	err := auth.verifyAndStoreToken(token)
	if err != nil {
		return err
	}

	req.Auth = &AuthAPI{
		auth,
	}

	return nil
}
