# Todo

- Compression (request body compression, and response encodings beyond gzip)
- Multipart
- Smarter Dump middleware, which adjusts the output format based on the size
and type of the body.  For example, limiting the size of the body dumped,
dumping binary bodies as hex, and maybe auto-indenting xml or json bodies.  Also
could handle multi parts bodies in a smart way.