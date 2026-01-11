package auth

// Token represents an access token with its expiry.
type Token struct {
	AccessToken string
	ExpiresAt   int64
}
