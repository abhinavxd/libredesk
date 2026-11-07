/* eslint-env node */
import { spawn, execFile } from 'node:child_process'
import { randomBytes } from 'node:crypto'
import { once } from 'node:events'
import { createServer } from 'node:net'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'
import { setTimeout as delay } from 'node:timers/promises'

const run = promisify(execFile)
const root = fileURLToPath(new URL('../../../', import.meta.url))
let server
let database
let databaseURL

export async function startSetupServer() {
  await stopSetupServer()
  databaseURL = new URL(process.env.SETUP_TEST_DB_DSN || 'postgres://libredesk:libredesk@127.0.0.1:5432/libredesk?sslmode=disable')
  database = `libredesk_setup_e2e_${randomBytes(6).toString('hex')}`
  await run('psql', [databaseURL.href, '-v', 'ON_ERROR_STOP=1', '-c', `CREATE DATABASE "${database}"`])
  const listener = createServer()
  listener.listen(0, '127.0.0.1')
  await once(listener, 'listening')
  const port = listener.address().port
  await new Promise((resolve) => listener.close(resolve))
  const origin = `http://localhost:${port}`
  const binary = process.env.SETUP_TEST_BINARY || `${root}/libredesk`
  const args = ['--config', `${root}/config.toml`]
  const env = {
    ...process.env,
    LIBREDESK_SYSTEM_USER_PASSWORD: '',
    LIBREDESK_DB__HOST: databaseURL.hostname,
    LIBREDESK_DB__PORT: databaseURL.port || '5432',
    LIBREDESK_DB__USER: decodeURIComponent(databaseURL.username),
    LIBREDESK_DB__PASSWORD: decodeURIComponent(databaseURL.password),
    LIBREDESK_DB__DATABASE: database,
    LIBREDESK_DB__SSL_MODE: databaseURL.searchParams.get('sslmode') || 'disable',
    LIBREDESK_REDIS__DB: '14',
    LIBREDESK_APP__SERVER__ADDRESS: `127.0.0.1:${port}`,
    LIBREDESK_APP__SERVER__DISABLE_SECURE_COOKIES: 'true',
    LIBREDESK_APP__ROOT_URL: origin,
    LIBREDESK_APP__CHECK_UPDATES: 'false'
  }
  try {
    await run(binary, [...args, '--install', '--yes'], { cwd: root, env })
    await run(binary, [...args, '--upgrade', '--yes'], { cwd: root, env })
    let output = ''
    server = spawn(binary, args, { cwd: root, env, stdio: ['ignore', 'pipe', 'pipe'] })
    server.stdout.on('data', (chunk) => { output += chunk })
    server.stderr.on('data', (chunk) => { output += chunk })
    for (let attempt = 0; attempt < 100; attempt++) {
      if (server.exitCode !== null) throw new Error('The setup test server exited before becoming ready.')
      const code = output.match(/expires in 1 hour\): ([a-zA-Z0-9]+)/)?.[1]
      if (code) {
        const response = await fetch(`${origin}/health`).catch(() => null)
        if (response?.ok) return { origin, code }
      }
      await delay(100)
    }
    throw new Error('The setup test server did not become ready within 10 seconds.')
  } catch (error) {
    await stopSetupServer()
    throw error
  }
}

export async function stopSetupServer() {
  if (server && server.exitCode === null) {
    const stopped = once(server, 'exit')
    server.kill('SIGTERM')
    await stopped
  }
  server = null
  if (database) {
    await run('psql', [databaseURL.href, '-v', 'ON_ERROR_STOP=1', '-c', `DROP DATABASE IF EXISTS "${database}" WITH (FORCE)`])
    database = null
  }
  return null
}
