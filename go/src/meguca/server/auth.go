package server

import (
	"database/sql"
	"net/http"
	"regexp"
	"strings"
	"time"

	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/db"

	"golang.org/x/crypto/bcrypt"
)

var (
	// Add "secure" flag to auth cookies.
	SecureCookie bool

	userIdRe = regexp.MustCompile(`^[-_ \p{Latin}\p{Cyrillic}\d]+$`)
)

type loginCreds struct {
	ID, Password string
	auth.Captcha
}

type passwordChangeRequest struct {
	Old, New string
	auth.Captcha
}

// Register a new user account
func register(w http.ResponseWriter, r *http.Request) {
	var req loginCreds
	isValid := decodeJSON(w, r, &req) &&
		trimLoginID(&req.ID) &&
		validateUserID(w, req.ID) &&
		checkPasswordAndCaptcha(w, r, req.Password, req.Captcha)
	if !isValid {
		return
	}

	hash, err := auth.BcryptHash(req.Password, 10)
	if err != nil {
		text500(w, r, err)
	}

	// Check for collision and write to DB
	switch err := db.RegisterAccount(req.ID, hash); err {
	case nil:
	case db.ErrUserNameTaken:
		text400(w, errUserIDTaken)
		return
	default:
		text500(w, r, err)
		return
	}

	commitLogin(w, r, req.ID)
}

// Separate function for easier chaining of validations
func validateUserID(w http.ResponseWriter, id string) bool {
	if id == "" || len(id) > common.MaxLenUserID || !userIdRe.MatchString(id) {
		text400(w, errInvalidUserID)
		return false
	}
	return true
}

// If login successful, generate a session token and commit to DB. Otherwise
// write error message to client.
func commitLogin(w http.ResponseWriter, r *http.Request, userID string) {
	token, err := auth.RandomID(128)
	if err != nil {
		text500(w, r, err)
		return
	}
	if err := db.WriteLoginSession(userID, token); err != nil {
		text500(w, r, err)
		return
	}

	// One hour less, so the cookie expires a bit before the DB session
	// gets deleted.
	expiry := time.Duration(common.SessionExpiry)*time.Hour*24 - time.Hour
	expires := time.Now().Add(expiry)
	sessionCookie := http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		Expires:  expires,
		Secure:   SecureCookie,
		HttpOnly: true,
	}
	setSameSiteCookie(w, &sessionCookie, SAMESITE_LAX_MODE)
}

// Log into a registered user account
func login(w http.ResponseWriter, r *http.Request) {
	var req loginCreds
	switch {
	case !decodeJSON(w, r, &req):
		return
	case !trimLoginID(&req.ID):
		return
	case !auth.AuthenticateCaptcha(req.Captcha):
		text403(w, errInvalidCaptcha)
		return
	}

	hash, err := db.GetPassword(req.ID)
	switch err {
	case nil:
	case sql.ErrNoRows:
		text403(w, common.ErrInvalidCreds)
		return
	default:
		text500(w, r, err)
		return
	}

	switch err := auth.BcryptCompare(req.Password, hash); err {
	case nil:
		commitLogin(w, r, req.ID)
	case bcrypt.ErrMismatchedHashAndPassword:
		text403(w, common.ErrInvalidCreds)
	default:
		text500(w, r, err)
	}
}

// Common part of both logout endpoints
func commitLogout(
	w http.ResponseWriter,
	r *http.Request,
	fn func(*auth.Session) error,
) {
	ss, ok := assertSession(w, r, "")
	if !ok {
		return
	}
	if err := fn(ss); err != nil {
		text500(w, r, err)
		return
	}

	expires := time.Unix(0, 0)
	sessionCookie := http.Cookie{
		Name:     "session",
		Value:    "0",
		Path:     "/",
		Expires:  expires,
		Secure:   SecureCookie,
		HttpOnly: true,
	}
	setSameSiteCookie(w, &sessionCookie, SAMESITE_LAX_MODE)
}

// Log out user from session and remove the session key from the database
func logout(w http.ResponseWriter, r *http.Request) {
	commitLogout(w, r, func(ss *auth.Session) error {
		return db.LogOut(ss.UserID, ss.Token)
	})
}

// Log out all sessions of the specific user
func logoutAll(w http.ResponseWriter, r *http.Request) {
	commitLogout(w, r, func(ss *auth.Session) error {
		return db.LogOutAll(ss.UserID)
	})
}

// Change the account password
func changePassword(w http.ResponseWriter, r *http.Request) {
	var msg passwordChangeRequest
	if !decodeJSON(w, r, &msg) {
		return
	}
	ss, ok := assertSession(w, r, "")
	if !ok || !checkPasswordAndCaptcha(w, r, msg.New, msg.Captcha) {
		return
	}

	// Get old hash
	hash, err := db.GetPassword(ss.UserID)
	if err != nil {
		text500(w, r, err)
		return
	}

	// Validate old password
	switch err := auth.BcryptCompare(msg.Old, hash); err {
	case nil:
	case bcrypt.ErrMismatchedHashAndPassword:
		text403(w, common.ErrInvalidCreds)
		return
	default:
		text500(w, r, err)
		return
	}

	// Old password matched, write new hash to DB
	hash, err = auth.BcryptHash(msg.New, 10)
	if err != nil {
		text500(w, r, err)
		return
	}
	if err := db.ChangePassword(ss.UserID, hash); err != nil {
		text500(w, r, err)
	}
}

// Check password length and authenticate captcha, if needed
func checkPasswordAndCaptcha(
	w http.ResponseWriter,
	r *http.Request,
	password string,
	captcha auth.Captcha,
) bool {
	switch {
	case password == "", len(password) > common.MaxLenPassword:
		text400(w, errInvalidPassword)
		return false
	case !auth.AuthenticateCaptcha(captcha):
		text403(w, errInvalidCaptcha)
		return false
	}
	return true
}

// Trim spaces from loginID. Chainable with other authenticators.
func trimLoginID(id *string) bool {
	*id = strings.TrimSpace(*id)
	return true
}

// Get request session data if any.
func getSession(r *http.Request, board string) (*auth.Session, error) {
	c, err := r.Cookie("session")
	if err != nil {
		return nil, common.ErrInvalidCreds
	}
	token := c.Value
	if len(token) != common.LenSession {
		return nil, common.ErrInvalidCreds
	}
	// Just in case, to avoid search for invalid board in DB.
	if board != "" && !config.IsServeBoard(board) {
		return nil, errInvalidBoard
	}
	// FIXME(Kagami): This might be affected to timing attack.
	return db.GetSession(board, token)
}

// Assert the user login session ID is valid.
func assertSession(w http.ResponseWriter, r *http.Request, b string) (ss *auth.Session, ok bool) {
	ss, err := getSession(r, b)
	switch err {
	case nil:
		ok = true
	case common.ErrInvalidCreds:
		text403(w, err)
	default:
		text500(w, r, err)
	}
	return
}

func setAccountSettings(w http.ResponseWriter, r *http.Request) {
}