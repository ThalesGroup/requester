# Todo

- RelativeURL("a", "b", "c") doesn't really work intuitively.  User probably means
RelativeURL("a/", "b/", "c/").  Can't simply append a path sep either.  The args are just
URLs, which might be fragments or query params
- Retries
- Compression
- Multipart
- Smarter Dump middleware, which adjusts the output format based in the size
and type of the body.  For example, limiting the size of the body dumped,
dumping binary bodies as hex, and maybe auto-indenting xml or json bodies.  Also
could handle multi parts bodies in a smart way.