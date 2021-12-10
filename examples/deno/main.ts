import { readLines } from "https://deno.land/std@0.117.0/io/mod.ts";

export class RtCvClient {
    private proc: Deno.Process;
    private textEncoder = new TextEncoder();
    private firstLineReader: Promise<string>;
    private linePromiseResolveFn: (value: string) => void = () => { };

    constructor() {
        this.proc = Deno.run({
            cmd: ["rtcv_scraper_client"],
            stdin: "piped",
            stdout: "piped",
        });

        this.panicOnExit();
        this.awaitProcStdOut();
        this.firstLineReader = this.readLine();
    }

    async authenticate(credentials: {
        serverLocation?: string;
        keyId?: string;
        key?: string;
        mock?: boolean;
        mockSecrets?: { [key: string]: unknown };
    }) {
        // await the ready message
        await this.firstLineReader;

        try {
            await this.rwLine(
                "set_credentials",
                credentials.mock
                    ? {
                        mock: { secrets: credentials.mockSecrets },
                    }
                    : {
                        server_location: credentials.serverLocation,
                        api_key_id: credentials.keyId,
                        api_key: credentials.key,
                    },
            );
        } catch (e) {
            console.log(e);
            console.log(
                "Hint: did you set the envourment variables using your shell or the .env?",
            );
            Deno.exit(1);
        }
    }

    // This is the main way of communicating with the client
    async rwLine<In, Out>(type: string, content: In): Promise<Out> {
        const lineReader = this.readLine<Out>();
        await this.writeLine(type, content);
        return await lineReader;
    }

    private async panicOnExit() {
        await this.proc.status();
        console.log("rtcv scraper client stopped unexpectedly");
        Deno.exit(1);
    }

    private async awaitProcStdOut() {
        for await (
            const line of readLines(this.proc.stdout as (Deno.Reader & Deno.Closer))
        ) {
            this.linePromiseResolveFn(line);
        }
    }

    private async readLine<T>(): Promise<T> {
        const line = await new Promise<string>((res) =>
            this.linePromiseResolveFn = res
        );
        const parsedLine: { type: string; content: T } = JSON.parse(line);
        if (parsedLine.type === "error") {
            throw `request send to rtcv_scraper_client returned an error: ${parsedLine.content}`;
        }
        return parsedLine.content;
    }

    private async writeLine<T>(type: string, content: T): Promise<void> {
        const input = this.textEncoder.encode(
            JSON.stringify({ type, content }) + "\n",
        );
        await this.proc.stdin?.write(input);
    }
}

const rtcvClient = new RtCvClient();
await rtcvClient.authenticate({
    mock: true,
    mockSecrets: { user: { username: "foo", password: "bar" } },
});
// await rtcvClient.authenticate({
//     key: 'abc',
//     keyId: '1',
//     serverLocation: 'http://localhost:4000',
// })
console.log("authenticated to RTCV");

const siteLoginCredentials = await rtcvClient.rwLine(
    "get_user_secret",
    {
        "encryption_key": "my-very-secret-encryption-key",
        "key": "user",
    },
);
console.log(siteLoginCredentials);
Deno.exit(0);
