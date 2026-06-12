# pkg-token
> Generic JWT token library (ES256/RS256, ConnectRPC + HTTP transports) reusable across all future Go apps.

**Created**: 2026-05-14 | **Branch**: feat/pkg-token

## Steps
1. [backend] Bump golang-jwt v4 → v5 and scaffold pkg/token
   → go.mod requires jwt/v5; pkg/token directories created (empty); ./... compiles and go vet passes

2. [backend] KeyProvider with PEM loaders for ES256 and RS256
   → KeyProvider interface with kid lookup; static loader from PEM bytes/file; supports EC and RSA keys; unit tests cover both algos plus unknown kid rejection

3. [backend] Signer interface unifying ES256 and RS256
   → Single Signer interface; ES256 and RS256 implementations; round-trip Sign/Verify tested; tampered tokens and wrong-algo tokens rejected

4. [backend] Claims contract and RegisteredClaims helper
   → Claims constraint type embeds jwt.Claims; helper builds iss/sub/aud/iat/exp/nbf/jti from ttl; unit test verifies field population

5. [backend] Manager[T] Issue and Parse with strict validation
   → Generic Manager with Issue and Parse; validates iss, aud, exp, nbf, signature, kid header; config validation at construction; happy-path and rejection-cause tests

6. [backend] Typed sentinel errors for parse failures
   → ErrExpired, ErrInvalid, ErrWrongAudience, ErrWrongIssuer, ErrUnknownKID, ErrSignature exported; errors.Is works; tests verify each cause maps to the right sentinel

7. [backend] ConnectRPC transport with generic interceptor
   → NewInterceptor[T] with WithPublicProcedure option; typed ClaimsFromContext[T]; tests cover bearer absent, malformed, invalid, valid, and public-procedure flows for unary and streaming

8. [backend] Plain HTTP transport with predicate-based authorization
   → RequireAuth[T] and Require[T](predicate) middlewares; typed ClaimsFromContext[T]; httptest coverage of missing header, bad token, predicate pass/fail

9. [backend] Migrate internal auth service to pkg/token.Manager
   → Service constructor accepts a typed token manager instead of raw signing key and duration; Login delegates to Issue, VerifyToken delegates to Parse; existing handler tests stay green

10. [backend] Remove pkg/authz and wire new transport in server bootstrap
    → pkg/authz package deleted; imports redirected to pkg/token and pkg/token/transport/connect; server loads ES256 keypair from env or file; race-enabled full test suite green

11. [backend] End-to-end integration test across both transports
    → Server with ConnectRPC interceptor plus token manager; client calls succeed with valid token, fail with 401 on missing/expired/wrong-audience; equivalent coverage via plain HTTP middleware

12. [backend] pkg/token README with bootstrap example and key generation
    → README documents public API, app-side bootstrap (claims, manager, interceptor wiring), openssl commands for ES256/RS256 keypair generation, explicit out-of-scope list (revocation, JWKS rotation, refresh orchestration)

## Execution
- step-01
- step-02 || step-03
- step-04
- step-05 || step-06
- step-07 || step-08
- step-09
- step-10
- step-11
- step-12
