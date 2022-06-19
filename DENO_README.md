# RT-CV scraper client

A [Deno](https://deno.land) client that can be used by a scraper to ease communication with [RT-CV](https://github.com/script-development/RT-CV)

## Get started

1. Create a `env.json` file with the following content (this file can also be obtained using the RTCV dashboard)
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

2. Add the boilerplate code to your scraper
```ts
import { RTCVScraperClient, LoginUsersRestriction, newCvToScan } from 'https://deno.land/x/rtcv_scraper_client/client.ts'

// Create a new rtcv scraper client and set the users restriction in the env.json file to have a single user
const rtcvClient = await new RTCVScraperClient(LoginUsersRestriction.One).authenticate()

const cv = newCvToScan('reference-nr-here')
// Fillin the remaining information now ..

// Send the cv to rtcv
await rtcvClient.sendCV(cv)
```

*Hint: it's best to*

## Exposed Methods / Vars

### `(RTCVScraperClient).sendCV(cv: CVToScan)`

This method sends a CV to RTCV

### `(RTCVScraperClient).loginUser : {username: string, password: string}`

Obtain a login user for the site you are scraping

You should only call this method if you have initialized `RTCVScraperClient` with `LoginUsersRestriction.One`

### `(RTCVScraperClient).loginUsers : Array<{username: string, password: string}>`

Obtain login users for the site you are scraping

You should only call this method if you have initialized `RTCVScraperClient` with `LoginUsersRestriction.OneOrMore`

### `(RTCVScraperClient).setCachedReference(referenceNr: string | number, options)`

Label a reference number as cached

This method is also called by `.sendCV(cv)`

### `(RTCVScraperClient).hasCachedReference(referenceNr: string | number): boolean`

Check if a reference number is cached

This can be used if you fetch a overview page and want to check for wich CVs you already have checked and thus can skip to fetch for aditional information as they will ignored by RTCV

### `LoginUsersRestriction`

```ts
export enum LoginUsersRestriction {
    None, // The website you are scraping doesn't require authentication
    One, // The website you are scraping has one user to login with
    OneOrMore, // The website you are scraping has one or more users to login with
}
```

### `newCvToScan(referenceNr: string): CvToScan`

Creates a new CV that can be used to send to RTCV using the `(RTCVScraperClient).sendCV(..)` method

This is mainly a helper function you can also create a CV yourself

## Dummy / Mock mode

The client can be used in dummy mode, this is useful for testing and debugging.

```ts
// the second argument for RTCVScraperClient tells if we want to use the dummy mode
new RTCVScraperClient(LoginUsersRestriction.One, true)
```
