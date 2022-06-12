export type { CVToScan, CVToScanPersonalDetails, CVToScanEducation, CVToScanLanguage, CVToScanWorkExperience } from './cv.ts'
export { newCvToScan, LangLevel } from './cv.ts'

import { envT, readEnv, serverT } from './env.ts'
import { CVToScan } from './cv.ts'

import { Sha512 } from "https://deno.land/std@0.143.0/hash/sha512.ts"

export enum LoginUsersRestriction {
    None, // This scraper doesn't have any domain login users
    One, // This scraper expects one user to login with
    OneOrMore, // This scraper expects at least one user to login with but can have more
}

export class RTCVScraperClient {
    private env: envT
    private servers: Array<ServerConn>
    private referenceCache: { [reference: string | number]: Date } = {}
    dummyMode = false

    constructor(loginUsersResitrction: LoginUsersRestriction) {
        this.env = readEnv()

        const loginUsersCount = this.env.login_users?.length
        if (loginUsersResitrction == LoginUsersRestriction.One) {
            if (!loginUsersCount) throw 'Expected exactly one login user but got none'
            if (loginUsersCount != 1) throw `Expected exactly one login user but got ${loginUsersCount}`
        } else if (loginUsersResitrction == LoginUsersRestriction.OneOrMore) {
            if (!loginUsersCount) throw 'Expected exactly one ore more login users but got none'
        }

        this.servers = [
            new ServerConn(this.env.primary_server),
            ...(this.env.alternative_servers?.map(s => new ServerConn(s)) || []),
        ]
    }

    async authenticate(): Promise<this> {
        if (this.dummyMode) return this

        await Promise.all(this.servers.map(s => s.checkHasScraperRole()))
        return this
    }

    // Get the user that is used to login to the site we scrape
    get loginUser() {
        return this.loginUsers[0]!
    }

    // Get the users that are used to login to the site we scrape
    get loginUsers() {
        return this.env.login_users!
    }

    async sendCV(cv: CVToScan) {
        if (this.hasCachedReference(cv.referenceNumber)) return
        this.setCachedReference(cv.referenceNumber)

        if (this.dummyMode) return

        await Promise.all([
            // We only care if the cv was accepted by the primary server
            this.servers[0].sendCV(cv),
            // From the other servers we will ignore the errors
            ...this.servers.slice(1).map(s => s.sendCV(cv).catch(_ => {/* Ignore errors */ }))
        ])
    }

    setCachedReference(referenceNr: string | number, options?: { ttlHours?: 12 | 24 | 72 }) {
        if (referenceNr === '') throw 'Reference number cannot be empty'
        if (this.hasCachedReference(referenceNr)) return
        const expireDate = new Date()
        expireDate.setHours(expireDate.getHours() + (options?.ttlHours ?? 72))
        this.referenceCache[referenceNr] = expireDate
    }

    hasCachedReference(referenceNr: string | number): boolean {
        const expireDate = this.referenceCache[referenceNr]
        if (!expireDate) return false

        if (expireDate < new Date()) {
            // This reference number has been expired
            delete this.referenceCache[referenceNr]
            return false
        }

        return true
    }
}

class ServerConn {
    server_location: string
    authHeader: string

    constructor(data: serverT) {
        this.server_location = data.server_location

        const hashedApiKey = new Sha512().update(data.api_key).hex()
        this.authHeader = `Basic ${data.api_key_id}:${hashedApiKey}`
    }

    private async doRequest(method: 'GET' | 'POST', path: string, body?: unknown) {
        const url = this.server_location + path
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json',
                'Authorization': this.authHeader,
            },
            body: body ? JSON.stringify(body) : undefined,
        }

        const req = await fetch(url, options)
        // TODO retry if server is not available
        if (req.status >= 400) {
            const text = await req.text()
            throw `server responsed with [${req.status}] ${req.statusText}: ${text}`
        }
        return await req.json()
    }

    async sendCV(cv: CVToScan) {
        await this.doRequest('POST', '/api/v1/scraper/scanCV', { cv: cv })
    }

    async checkHasScraperRole() {
        const keyInfo: { roles: Array<{ role: number }> } = await this.doRequest('GET', '/api/v1/auth/keyinfo')
        const hasScraperRole = keyInfo.roles.some(r => r.role == 1)
        if (!hasScraperRole) throw `api key for ${this.server_location} does not have the scraper role`
    }
}
