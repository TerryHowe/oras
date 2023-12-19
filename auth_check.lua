-- This file contains the Lua code for the authentication check used by NGINX 
-- It sends a HTTP HEAD request to the upstream registry (NGX)
-- to validate user credentials before serving from the cache

-- Include the 'resty.http' module for HTTP requests
local http = require "resty.http"

-- Function to validate credentials against an upstream registry
local function validate_credentials(auth_header, upstream_registry)
    -- Create a new HTTP client instance
    local httpc = http.new()

    -- Perform a HEAD request to the upstream registry with the provided authorization header
    local res, err = httpc:request_uri(upstream_registry, {
        method = "HEAD",
        headers = {
            ["Authorization"] = auth_header
        }
    })

    -- Check if the request was not successful
    if not res then
        -- Log an error message with the failure reason
        ngx.log(ngx.ERR, "Failed to send request: ", err)
        -- Return false to indicate failed validation
        return false
    end

    -- Return true on successful validation
    return res.status == 200
end

-- Retrieve the 'Authorization' header from the incoming request
local auth_header = ngx.var.http_authorization
-- Define the URL of the upstream registry to validate against
local upstream_registry = "https://nvcr.io"  

-- Check if there's an 'Authorization' header and validate the credentials
if auth_header and validate_credentials(auth_header, upstream_registry) then
    -- Log a notice message if the credentials are successfully validated
    ngx.log(ngx.NOTICE, "Credentials validated successfully.")
else
    -- Send an HTTP 401 Unauthorized response if validation fails
    ngx.exit(ngx.HTTP_UNAUTHORIZED)
end
