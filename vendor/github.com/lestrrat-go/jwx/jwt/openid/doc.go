// Package openid provides a specialized token that provides utilities
// to work with OpenID JWT tokens.
//
// In order to use OpenID claims, you specify the token to use in the
// jwt.Parse method
//
//    jwt.Parse(data, jwt.WithOpenIDClaims())
package openid
