package common

import "crypto/subtle"

func IsUIAuthEnabled() bool {
	return EnvConfig.UIUsername != "" && EnvConfig.UIPassword != ""
}

// CheckUICredentials compares in constant time so the response time does not leak a partial match.
func CheckUICredentials(user, password string) bool {
	userOK := subtle.ConstantTimeCompare([]byte(user), []byte(EnvConfig.UIUsername)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(EnvConfig.UIPassword)) == 1
	return userOK && passOK
}
