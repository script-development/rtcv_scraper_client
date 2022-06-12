import vs from 'https://deno.land/x/value_schema@v4.0.0-rc.1/mod.ts'

const server = vs.object({
    schemaObject: {
        server_location: vs.string({ minLength: 1 }),
        api_key_id: vs.string({ minLength: 1 }),
        api_key: vs.string({ minLength: 1 }),
    }
})

export type serverT = ReturnType<typeof server.applyTo>

const loginUser = vs.object({
    schemaObject: {
        username: vs.string({ minLength: 1 }),
        password: vs.string({ minLength: 1 }),
    }
})

const envValidationSchema = vs.object({
    schemaObject: {
        login_users: vs.array({
            ifUndefined: undefined,
            each: loginUser,
        }),
        primary_server: server,
        alternative_servers: vs.array({
            ifUndefined: undefined,
            each: server,
        }),
    },
})

export type envT = ReturnType<typeof envValidationSchema.applyTo>

export function readEnv(): envT {
    try {
        const envContents = Deno.readTextFileSync('env.json')
        const env = JSON.parse(envContents)
        const validatedEnv = envValidationSchema.applyTo(env)
        return validatedEnv
    } catch (e) {
        throw [
            `\nUnable to read ./env.json in ${Deno.cwd()}`,
            `hint: you can create a env.json on the RTCV dashboard for this scraper?`,
            `error: ${e}`,
        ].join('\n')
    }
}
