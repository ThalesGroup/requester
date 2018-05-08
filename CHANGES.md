# Sling Changelog

Notable changes between releases.

## 0.2.0
### Changed
- Breaking change: BodyMarshaler and BodyUnmarshaler interfaces changed names to just Marshaler
  and Unmarshaler
- Breaking change: Marshaler and Unmarshaler options changed names to WithMarshaler and WithUnmarshaler
### Removed
- Non2xxCode has been replaced with ExpectCode and ExpectSuccessCode
- clientserver package has been replaced with httptestutil
### Added
- requester.Inspector: A utility which can be installed in a Requester, which captures the most recent
  outgoing request, request body, incoming response, and response body.
- requester "mocks": Added MockRequest(), MockDoer(), ChannelDoer(), MockHandler(), and ChannelHandler().
  These are convenience methods for writing tests which create http.Requests, mock Doers, and mock http.Handlers.
- DoerFunc, MiddlewareFunc, MarshalerFunc, and UnmarshalerFunc are now Options: they can passed directly to Requester
  as options, rather than wrapping them in the WithDoer, WithMarshaler, etc.
- ExpectCode and ExpectSuccessCode options: these convert responses which don't have the required StatusCodes into
  errors.
- httptestutil package: contains an Inspector utility which intercepts and captures traffic to and from
  an httptest.Server.
- More examples.
### Fixed
- Receive() and ReceiveContext() would fail to apply marshaling options correctly.  This is fixed.
- If middleware turned a real server response into an error, the response body might never be read,
  which could lead to leaks.  Receive now always reads and returns the body of the response whenever possible,
  even if Do() returned an error.




