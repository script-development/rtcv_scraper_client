const req = await fetch(Deno.env.get('SCRAPER_ADDRESS') + '/users')
const users = await req.json()
console.log(users)

