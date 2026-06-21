package handlers

// CombinedHandler embeds UserHandler and AuthHandler to satisfy the full ServerInterface.
type CombinedHandler struct {
	*UserHandler
	*AuthHandler
}

func NewCombinedHandler(user *UserHandler, auth *AuthHandler) *CombinedHandler {
	return &CombinedHandler{UserHandler: user, AuthHandler: auth}
}
