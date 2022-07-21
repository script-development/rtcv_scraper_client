# RT-CV scraper client

A helper program that aims to ease the communication between a scraper and [RT-CV](https://github.com/script-development/RT-CV)

## How does this work?

This scraper client handles authentication and communication with RT-CV and beside that also has a cache for the already fetched reference numbers.

This scraper works like this:

1. You run `rtcv_scraper_client` in your terminal
2. The scraper client reads `env.json` and authenticates with RT-CV
3. The scraper client spawns the program you have defined as args for the `rtcv_scraper_client`, for example with an npm scraper it would be something like `rtcv_scraper_client npm run start`
4. Your scraper can now easially talk with `RT-CV` via `rtcv_scraper_client` using http requests where the http server address is defiend by a shell variable set by the scraper client `$SCRAPER_ADDRESS`

## Why this client?

Every scraper needs to communicate with RT-CV and the amound of code that require is quite a lot.

If we have the same code for communicating with RT-CV we only have a single point of failure and updating / adding features is easy.

## Example

A Deno example

```ts
// denoexample.ts
const req = await fetch(Deno.env.get('SCRAPER_ADDRESS') + '/users')
const users = await req.json()
console.log(users)
```

```sh
# rtcv_scraper_client deno run -A denoexample.ts
credentials set
testing connections..
connected to RTCV
running scraper..
Check file:///.../denoexample.ts
[ { username: "username here", password: "password here" } ]
```

## Setup & Run

### *1.* Install the helper

```sh
# Install latest version
go install github.com/script-development/rtcv_scraper_client/v2@latest
# Install a specific version
go install github.com/script-development/rtcv_scraper_client/v2@v2.3.0
```

### *2.* Obtain a `env.json`

Create a `env.json` file with the following content **(this file can also be obtained from the RTCV dashboard, tough note that you might need to add login_users yourself)**
```js
{
    "primary_server": {
        "server_location": "http://localhost:4000",
        "api_key_id": "aa",
        "api_key": "bbb"
    },
    "alternative_servers": [
        // If you want to send CVs to multiple servers you can add additional servers here
    ],
    "login_users": [
        {"username": "scraping-site-username", "password": "scraping-site-password"}
    ],
    // For production, set mock_mode to false
    // "mock_mode": false
}
```

### *3.* Develop / Deploy a scraper using `rtcv_scraper_client`

You can now prefix your scraper's run command with `rtcv_scraper_client` and the scraper client program will run a webserver as long as your scraper runs where via you can communicate with RT-CV.

By default the program will run in `mock_mode`, for production you'll have to explicitly turn it off by setting `"mock_mode": false` in your env.json

If you have for a NodeJS project you can run your program like this:

```sh
rtcv_scraper_client npm run start
```

## Routes available

Notes:
- The http method can be anything
- If errors occur the response will be the error message with a 400 or higher status code

### `$SCRAPER_ADDRESS/send_cv`

Sends a cv to rtcv and remembers the reference number

- Body: In JSON the cv send to RT-CV
- Resp: **true** / **false** if the cv was sent to RT-CV

### `$SCRAPER_ADDRESS/users`

Returns the login users from the `env.json`

- Body: None
- Resp: **true** / **false** the login users from env.json

### `$SCRAPER_ADDRESS/set_cached_reference`

Manually add another cached reference with the default ttl (3 days)

Note that this is also done by the send_cv route

- Body: The reference number
- Resp: **true**

### `$SCRAPER_ADDRESS/set_short_cached_reference`

manually add another cached reference with a short ttl (12 hours)

- Body: The reference number
- Resp: **true**

### `$SCRAPER_ADDRESS/get_cached_reference`

Check if a reference number is in the cache

- Body: The reference number
- Resp: **true** / **false**

## `env.json` as env variable?

If you are using containers you might want to use the contents of your `env.json` as env variable instaid of binding it to the container as volume.

You can do that with the following shell variable:

```sh
export RTCV_SCRAPER_CLIENT_ENV='{}'
```

## Deno client docs

[click here](./DENO_README.md)
