## 1.0.0
This marks the API as stable.

### Added
- Range(): Sets the range header
- AppendPath(): Adds path elements to the URL path, using slightly different
  rules than RelativeURL().
- Contributing guidelines
### Changed
- The Makefile no longer auto-installs dep.  It's available via package managers like brew now.

## 0.3.0
### Added
- Requester.MustWith(): a version of With() which panics on errors
- DumpToStderr()
- Revised docs and more examples
### Removed
- clientserver package.  Use 0.2.x to transition to the httptestutil package

## 0.2.3
### Fixed
- Using httptestutil.Inspect() and Dump() with an httptest.Server that has a nil Handler would
  cause a panic.

## 0.2.2
### Changed
- Breaking change: BodyMarshaler and BodyUnmarshaler interfaces changed names to just Marshaler
  and Unmarshaler
- Breaking change: Marshaler and Unmarshaler options changed names to WithMarshaler and WithUnmarshaler
- Breaking change: DumpToStandardOut() is renamed to DumpToStout()
### Removed
- Non2xxCode has been replaced with ExpectCode and ExpectSuccessCode
### Deprecated
- clientserver package has been deprecated and will be removed.  Replace with httptestutil
### Added
- Inspect(): A utility which can be installed in a Requester, which captures the most recent
  outgoing request, request body, incoming response, and response body.
- requester "mocks": Added MockRequest(), MockDoer(), ChannelDoer(), MockHandler(), and ChannelHandler().
  These are convenience methods for writing tests which create http.Requests, mock Doers, and mock http.Handlers.
- DoerFunc, MiddlewareFunc, MarshalerFunc, and UnmarshalerFunc are now Options: they can passed directly to Requester
  as options, rather than wrapping them in the WithDoer, WithMarshaler, etc.
- ExpectCode and ExpectSuccessCode options: these convert responses which don't have the required StatusCodes into
  errors.
- httptestutil package: contains an Inspector utility which intercepts and captures traffic to and from
  an httptest.Server.
- httptestutil.Dump(), httptestutil.DumpToLog(), httptestutil.DumpToStdout(): Functions which intercept traffic
  to and from an httptest.Server, and dump the requests and responses to various outputs.
- More examples.
### Fixed
- Receive() and ReceiveContext() would fail to apply marshaling options correctly.  This is fixed.
- If middleware turned a real server response into an error, the response body might never be read,
  which could lead to leaks.  Receive now always reads and returns the body of the response whenever possible,
  even if Do() returned an error.




