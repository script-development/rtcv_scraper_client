export class RtCvClient {
    private serverOrigin = Deno.env.get('SCRAPER_ADDRESS')
    private async fetch<T>(path: string, body?: BodyInit | null | undefined): Promise<T> {
        if (!this.serverOrigin) throw 'it seems like you are running this scraper outside of rtcv_scraper_client, make sure you are using rtcv_scraper_client to run this scraper'

        const req = await fetch(this.serverOrigin + path, { body: body })
        if (req.status >= 400) throw await req.text()
        return await req.json()
    }
    sendCV(cv: unknown): Promise<void> {
        return this.fetch("/send_cv", JSON.stringify(cv))
    }
    hasCachedReference(nr: string): Promise<boolean> {
        return this.fetch('/get_cached_reference', nr)
    }
    setCachedReference(nr: string): Promise<void> {
        return this.fetch('/set_cached_reference', nr)
    }
    getUsers(): Promise<Array<{ username: string, password: string }>> {
        return this.fetch('/users')
    }
}

const rtcvClient = new RtCvClient()
const siteLoginCredentials = await rtcvClient.getUsers()
console.log('siteLoginCredentials:', siteLoginCredentials)
