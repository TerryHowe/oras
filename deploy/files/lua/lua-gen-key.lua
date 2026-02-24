-- generate a single private key in memory on app startup.
-- we don't need to generate more private keys, only public certs
-- so this can be static.
local openssl_pkey = require "resty.openssl.pkey"
local certs_key    = ngx.shared.certs_key

ngx.log(ngx.INFO, "generating new private key for MITM")

local key = openssl_pkey.new({ bits = 4096 })
local key_der = assert(key:tostring("private", "DER"))
assert(certs_key:set("STATIC_KEY", key_der))
