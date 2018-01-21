# Todo

- RelativeURL("a", "b", "c") doesn't really work intuitively.  User probably means
RelativeURL("a/", "b/", "c/").  Can't simply append a path sep either.  The args are just
URLs, which might be fragments or query params
- Option/Middleware to convert non-2xx responses to an error
- Middleware for logging
- Retries
- Compression
- Multipart
- some of the method param and receiver naming is inconsistent