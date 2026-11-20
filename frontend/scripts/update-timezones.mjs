import { readFile, writeFile } from 'node:fs/promises'

const source = process.argv[2] || '/usr/share/zoneinfo/zone.tab'
const destination = new URL('../shared-ui/constants/timezones.json', import.meta.url)
const existing = JSON.parse(await readFile(destination, 'utf8'))
const zoneTable = await readFile(source, 'utf8')
const zones = zoneTable
  .split('\n')
  .filter((line) => line.trim() && !line.startsWith('#'))
  .map((line) => line.split('\t')[2])

if (zones.length < 400 || zones.some((zone) => !zone)) {
  throw new Error('Expected an IANA zone.tab file containing the complete timezone list')
}

// Retain previously offered names when IANA renames a timezone.
const timezones = [...new Set(['UTC', ...existing, ...zones])].sort()
for (const timeZone of timezones) {
  new Intl.DateTimeFormat('en', { timeZone })
}

await writeFile(destination, JSON.stringify(timezones, null, 2) + '\n')
console.log(`Updated ${timezones.length} timezones from ${source}`)
