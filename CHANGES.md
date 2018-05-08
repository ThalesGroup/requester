# Sling Changelog

Notable changes between releases.

## 0.2.0
### Changed
- Breaking change: BodyMarshaler and BodyUnmarshaler interfaces changed names to just Marshaler
  and Unmarshaler
- Breaking change: Marshaler and Unmarshaler options changed names to WithMarshaler and WithUnmarshaler
- Breaking change: clientserver package has changed significantly.  Constructor "New" is now replaced
  with a set of constructors which mirror the httptest package (e.g. `NewServer(http.Handler)`, etc).
  Also, the Requester is no longer embedded, but is constructed on-demand with `Requester()`.
  The LastXXX fields, which held information captured from the last request/response exchange has been
  refactored into a requester.Inspector and a clientserver.Inspector, which are created and installed
  on-demand with `InspectClient()` and `InspectServer()`
- Breaking change: clientserver.ClientServer's is no longer an http.Handler
### Removed
- Non2xxCode has been replaced with ExpectCode and ExpectSuccessCode
### Added
- requester.Inspector: A utility which can be installed in a Requester, which captures the most recent
  outgoing request, request body, incoming response, and response body.
- requester "mocks": Added MockDoer(), ChannelDoer(), and MockRequest().  These are convenience methods
  for writing tests which create mock Doer's and mocked http.Requests.
- DoerFunc, MiddlewareFunc, MarshalerFunc, and UnmarshalerFunc are now Options: they can passed directly to Requester
  as options, rather than wrapping them in the WithDoer, WithMarshaler, etc.
- ExpectCode and ExpectSuccessCode options: these convert responses which don't have the required StatusCodes into
  errors.
- More examples.
### Fixed
- Receive() and ReceiveContext() would fail to apply marshaling options correctly.  This is fixed.
- If middleware turned a real server response into an error, the response body might never be read,
  which could lead to leaks.  Receive now always reads and returns the body of the response whenever possible,
  even if Do() returned an error.




