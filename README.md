# RT-CV scraper client

A client that can be spawned by a scraper to ease communication with [RT-CV](https://github.com/script-development/RT-CV)

## Communication flow

```
custom scraper <> rtcv_scraper_client (this project) <> RT-CV
```

1. The custom scraper spawns rtcv_scraper_client as child process
2. The custom scraper sends it's credentials to the child process via stdin
3. rtcv_scraper_client handles the authentication and reports if it went successfull
4. The custom scraper starts scraping and sends every scraped result to it's child process (rtcv_scraper_client)
5. rtcv_scraper_client sends the scraped data to rt-cv and reports if it was successfull
6. ...

## Methods

### Authentication

There are 3 authentication methods
- `set_credentials` if you have one RT-CV server
- `set_multiple_credentials` if you have multiple RT-CV servers
- `set_mock` if you want to debug a scraper, this mocks the RT-CV server

#### `set_credentials`

Set credentials and location of a RT-CV server

Example input

```json
{"type":"set_credentials","content":{"server_location":"http://localhost:4000","api_key_id":"111111111111111111111111","api_key":"ddd"}}
```

Ok Response

```json
{"type":"ok"}
```

#### `set_multiple_credentials`

Set credentials and location of multiple RT-CV servers

Note that we need to define which server the primary should be, the primary server is used to fetch secrets and recently scraped reference numbers

Example input

```json
{"type":"set_credentials","content":[{"primary":true,"server_location":"http://localhost:4000","api_key_id":"111111111111111111111111","api_key":"ddd"},{"server_location":"http://localhost:4000","api_key_id":"111111111111111111111111","api_key":"ddd"}]}
```

Ok Response

```json
{"type":"ok"}
```

#### `set_mock`

Enable mock mode

Example input:

```json
{"type":"set_mock","content":{}}
```

You can also provide mock secrets for the `*_secret` methods.

This object need to follow the following type convention `key (string) -> value (any)`

```json
{"type":"set_mock","content":{"secrets":{"users": [{"username":"foo","password":"bar"}],"user":{"username":"foo","password":"bar"}}}}
```

Ok Response

```json
{"type":"ok"}
```

### `send_cv`

Send a scraped CV to RT-CV

Example input

```json
{"type":"send_cv","content":{"reference_number":"abcd","..":".."}}
```

Ok Response

*The content represents if a match was made*

```json
{"type":"ok","content":true}
```

### `get_secret`

Get a user defined secret from the server

Example input

```json
{"type":"get_secret","content":{"encryption_key":"my-very-secret-encryption-key", "key":"key-of-value"}}
```

Ok Response

```jsonc
{"type":"ok","content":{/*Based on the content stored in the secret value*/}}
```

### `get_users_secret`

Get a secret from the server where the contents is a strictly defined list of users

Example input

```json
{"type":"get_users_secret","content":{"encryption_key":"my-very-secret-encryption-key", "key":"users"}}
```

Ok Response

```json
{"type":"ok","content":[{"username":"foo","password":"foo"},{"username":"bar","password":"bar"}]}
```

### `get_user_secret`

Get a secret from the server where the contents is a strictly defined user

Example input

```json
{"type":"get_user_secret","content":{"encryption_key":"my-very-secret-encryption-key", "key":"user"}}
```

Ok Response

```json
{"type":"ok","content":{"username":"foo","password":"bar"}}
```

### `set_cached_reference`

Save a reference number to the cache
This cache is used to avoid sending the same CV twice or scraping data that has already been scraped

*Note that "send_cv" also executes this function automatically*

Example input

```json
{"type":"set_cached_reference","content":"abcd"}
```

Ok Response

```json
{"type":"ok"}
```

### `has_cached_reference`

Is there a cache entry for a specific reference number?

Example input

```json
{"type":"has_cached_reference","content":"abcd"}
```

Ok Response

```json
{"type":"ok","content":true}
```

### `ping`

Send a ping message to the server and the server will respond with a pong

Example input

```json
{"type":"ping"}
```

Ok Response

```json
{"type":"pong"}
```

## How to develop

```sh
# After a change update the binary in your path using:
go install

# Then in your scraper project
cd ../some-scraper
go run .
```

Debug data send by scraper to this program
```sh
export LOG_SCRAPER_CLIENT_INPUT=true
# Now run your scraper
go run .

# Now all messages written to this program are also written to scraper_client_input.log
# You can follow the input by tailing the file:
tail -f scraper_client_input.log

# After collecting the input data you can also use rtcv_scraper_client to replay sending the data
# This is very handy for debugging
rtcv_scraper_client -replay scraper_client_input.log

# There are also additional options for the -replay command:
rtcv_scraper_client \
    -replay ../werk-scraper/scraper_client_input.log \
    -replaySkipCommands has_cached_reference,set_cached_reference,ping,get_users_secret
```

## How to ship?

Currently we don't have pre build binaries so you'll need to compile the binary yourself

```Dockerfile
FROM golang:alpine AS obtain-rtcv-client
RUN go install github.com/script-development/rtcv_scraper_client@latest

FROM denoland/deno:alpine AS runtime
COPY --from=obtain-rtcv-client /go/bin/rtcv_scraper_client /bin/rtcv_scraper_client
```
