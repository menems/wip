// Package token provides a generic, asymmetric JWT issuing and verification
// library. Each token type is modelled by a typed [Manager], parameterised by
// the application's own [Claims] struct.
//
// Higher-level helpers for HTTP and ConnectRPC transports live under the
// transport sub-packages.
package token
