package handlers

// CombinedHandler embeds UserHandler, AuthHandler, and ClassHandler to satisfy the full ServerInterface.
type CombinedHandler struct {
	*UserHandler
	*AuthHandler
	*ClassHandler
}

func NewCombinedHandler(user *UserHandler, auth *AuthHandler, class *ClassHandler) *CombinedHandler {
	return &CombinedHandler{UserHandler: user, AuthHandler: auth, ClassHandler: class}
}
