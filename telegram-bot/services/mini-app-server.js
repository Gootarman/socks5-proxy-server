import { readFile } from 'node:fs/promises'
import http from 'node:http'
import { join } from 'node:path'

import { dirname } from './utils.js'

export class MiniAppServer {
  #logger

  constructor (logger) {
    this.#logger = logger
  }

  run () {
    const useWebHooks = Number.parseInt(process.env.TELEGRAM_USE_WEB_HOOKS ?? process.env.TELEGRAM_USE_WEBHOOKS) === 1
    if (useWebHooks) {
      this.#logger.warn('Mini App server is disabled in webhooks mode. Use external MINI_APP_URL.')
      return
    }

    const port = Number.parseInt(process.env.BOT_APP_PORT)
    if (!Number.isFinite(port)) {
      this.#logger.warn('BOT_APP_PORT is not set. Mini App server is disabled.')
      return
    }

    const miniAppIndexPath = join(dirname(import.meta.url), '..', 'mini-app', 'index.html')

    const server = http.createServer(async (req, res) => {
      const pathname = new URL(req.url, 'http://127.0.0.1').pathname

      if (pathname === '/healthz') {
        res.writeHead(200, { 'Content-Type': 'text/plain; charset=utf-8' })
        res.end('ok')
        return
      }

      if (pathname === '/mini-app' || pathname === '/mini-app/') {
        try {
          const indexHtml = await readFile(miniAppIndexPath, 'utf8')

          res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' })
          res.end(indexHtml)
        } catch (err) {
          this.#logger.error(err)
          res.writeHead(500, { 'Content-Type': 'text/plain; charset=utf-8' })
          res.end('Internal Server Error')
        }

        return
      }

      res.writeHead(404, { 'Content-Type': 'text/plain; charset=utf-8' })
      res.end('Not Found')
    })

    server.listen(port, () => {
      this.#logger.info(`Mini App server is started on port ${port}`)
      this.#logger.info(`Mini App path is available at /mini-app`)
    })
  }
}
