package constant

import "errors"

type Error struct {
	Code    int64  `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

var (
	ErrServerError                    = errors.New("Server Error")
	ErrParseRequestBody               = errors.New("Failed to parse request body")
	ErrParseRequestParams             = errors.New("Failed to parse request params")
	ErrInvalidCredentials             = errors.New("Username or password is not valid")
	ErrAccountNotAvailable            = errors.New("Account is not available")
	ErrShopNotAvailable               = errors.New("Shop is not available")
	ErrAccountDisabled                = errors.New("Account is disabled")
	ErrInvalidAccount                 = errors.New("Invalid account")
	ErrUnknownActionType              = errors.New("Unknown action type")
	ErrUnknownAppCode                 = errors.New("Unknown app code")
	ErrUnsupportedDomain              = errors.New("Unsupported domain")
	ErrInvalidShop                    = errors.New("Invalid shop")
	ErrInvalidAppShop                 = errors.New("Invalid app_shop")
	ErrInvalidToken                   = errors.New("Token is not valid")
	ErrUnknownTokenType               = errors.New("Unknown token type")
	ErrRedisMissingTokenKeyPrefix     = errors.New("Missing key prefix for token")
	ErrRedisMustSetExpirationForToken = errors.New("Must set expiration time for token")
	ErrTokenExpired                   = errors.New("Token is expired")
	ErrDomainNotAvailable             = errors.New("Domain is not available")
	ErrPasswordIncorrect              = errors.New("Password is incorrect")
	ErrInvalidInput                   = errors.New("Invalid input")
	ErrPermissionDenied               = errors.New("Permission denied")
	ErrInvalidRequest                 = errors.New("Invalid request")
	ErrNotFound                       = errors.New("Not found")

	HttpErrorTokenExpired = errors.New("Token expired ")
	HttpErrorTokenMissing = errors.New("Missing authentication token ")
	HttpErrorForbidden    = errors.New("403 Forbidden – you don’t have permission to access on server ")
	HttpErrorTokenInvalid = errors.New("Token invalid ")
	HttpErrorNotFound     = errors.New("Not found! ")
	HttpErrorParseBody    = errors.New("error while parse request body")

	unknownErrCode = Error{
		Code:    999,
		Message: "Unknown error",
	}

	TokenExpiredCode = 456

	mapErrCodes = map[int64]string{
		100: "This shop is already registered with this account",
		101: "This shop is already registered with another account",
		110: "This email address has already been used",
		114: "Current password is incorrect",
	}
)
