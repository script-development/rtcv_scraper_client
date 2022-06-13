export async function fetchWithRetry(input: string | Request, init?: RequestInit) {
    let attempt = 0
    let lastError: unknown
    while (attempt <= 4) {
        if (attempt > 0) {
            const seconds = attempt * 4
            console.warn(`fetch failed to RTCV, retrying after ${seconds} seconds`)
            await sleep(seconds * 1000)
        }
        try {
            return await fetch(input, init)
        } catch (e) {
            attempt++
            lastError = e
        }
    }
    throw `fetch failed to RTCV, retried 4 times. ${lastError}`
}

export const sleep = (timeout: number) => new Promise((res) => setTimeout(res, timeout))
