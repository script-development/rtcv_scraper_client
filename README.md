# EXPERIMENT

This project is just an experiment to try a different approach to communicating with [RT-CV](https://github.com/script-development/RT-CV) from a custom scraper

# RT-CV scraper client

A client that handles communication with rt-cv

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

## What this should do?

- Handle authentication
- Handle secrets
- Remember the reference numbers of the scraped data
