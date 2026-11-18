import { describe, expect, test } from 'vitest'
import { SESSION_CHANGED, runChatInit } from './chatInit.js'

const required = () => {
  const error = new Error('turnstile')
  error.response = { data: { data: { code: 'turnstile_required' } } }
  return error
}

describe('runChatInit', () => {
  test('sends a visitor token before a normal init', async () => {
    const posts = []
    const response = await runChatInit({
      data: { message: 'hi' },
      getSession: () => 'visitor',
      isVisitor: () => true,
      getSiteKey: () => 'site',
      requestToken: async () => 'token-1',
      refreshSiteKey: async () => {},
      post: async (payload) => {
        posts.push(payload)
        return { ok: true }
      }
    })
    expect(response).toEqual({ ok: true })
    expect(posts).toEqual([{ message: 'hi', turnstile_token: 'token-1' }])
  })

  test('does not ask a signed-in user for a token', async () => {
    const posts = []
    await runChatInit({
      data: { message: 'hi' },
      getSession: () => 'user',
      isVisitor: () => false,
      getSiteKey: () => 'site',
      requestToken: async () => {
        throw new Error('token requested')
      },
      refreshSiteKey: async () => {},
      post: async (payload) => {
        posts.push(payload)
        return { ok: true }
      }
    })
    expect(posts).toEqual([{ message: 'hi' }])
  })

  test('posts a campaign reply once when the server accepts it', async () => {
    const posts = []
    await runChatInit({
      data: { message: 'hi', delivery_id: 'delivery' },
      getSession: () => 'visitor',
      isVisitor: () => true,
      getSiteKey: () => 'site',
      requestToken: async () => {
        throw new Error('token requested')
      },
      refreshSiteKey: async () => {},
      post: async (payload) => {
        posts.push(payload)
        return { ok: true }
      }
    })
    expect(posts).toEqual([{ message: 'hi', delivery_id: 'delivery' }])
  })

  test('fetches a token when a campaign reply would create a conversation', async () => {
    const posts = []
    let calls = 0
    await runChatInit({
      data: { message: 'hi', delivery_id: 'delivery' },
      getSession: () => 'visitor',
      isVisitor: () => true,
      getSiteKey: () => 'site',
      requestToken: async () => 'token-1',
      refreshSiteKey: async () => {},
      post: async (payload) => {
        calls += 1
        posts.push(payload)
        if (calls === 1) throw required()
        if (calls > 2) throw new Error('retried more than once')
        return { ok: true }
      }
    })
    expect(posts[0]).toEqual({ message: 'hi', delivery_id: 'delivery' })
    expect(posts[1]).toEqual({ message: 'hi', delivery_id: 'delivery', turnstile_token: 'token-1' })
  })

  test('refreshes a missing site key and retries the same draft once', async () => {
    const posts = []
    let siteKey = ''
    await runChatInit({
      data: { message: 'hi', form_data: { email: 'a@b.c' } },
      getSession: () => 'visitor',
      isVisitor: () => true,
      getSiteKey: () => siteKey,
      requestToken: async (key) => `token-for-${key}`,
      refreshSiteKey: async () => {
        siteKey = 'site'
      },
      post: async (payload) => {
        posts.push(payload)
        if (!payload.turnstile_token) throw required()
        return { ok: true }
      }
    })
    expect(posts).toEqual([
      { message: 'hi', form_data: { email: 'a@b.c' } },
      { message: 'hi', form_data: { email: 'a@b.c' }, turnstile_token: 'token-for-site' }
    ])
  })

  test('does not retry a rejected token when the site key is unchanged', async () => {
    const posts = []
    await expect(
      runChatInit({
        data: { message: 'hi' },
        getSession: () => 'visitor',
        isVisitor: () => true,
        getSiteKey: () => 'site',
        requestToken: async () => 'token-1',
        refreshSiteKey: async () => {},
        post: async (payload) => {
          posts.push(payload)
          throw required()
        }
      })
    ).rejects.toBeInstanceOf(Error)
    expect(posts).toHaveLength(1)
  })

  test('retries once when the site key changed after the token was rejected', async () => {
    const posts = []
    let siteKey = 'old'
    await runChatInit({
      data: { message: 'hi' },
      getSession: () => 'visitor',
      isVisitor: () => true,
      getSiteKey: () => siteKey,
      requestToken: async (key) => `token-for-${key}`,
      refreshSiteKey: async () => {
        siteKey = 'new'
      },
      post: async (payload) => {
        posts.push(payload)
        if (payload.turnstile_token === 'token-for-old') throw required()
        return { ok: true }
      }
    })
    expect(posts.map((payload) => payload.turnstile_token)).toEqual(['token-for-old', 'token-for-new'])
  })

  test('does not post when the session changes while the token is pending', async () => {
    let session = 'visitor'
    const posts = []
    await expect(
      runChatInit({
        data: { message: 'hi', form_data: { email: 'a@b.c' } },
        getSession: () => session,
        isVisitor: () => session === 'visitor',
        getSiteKey: () => 'site',
        requestToken: async () => {
          session = 'signed-in'
          return 'token-1'
        },
        refreshSiteKey: async () => {},
        post: async (payload) => {
          posts.push(payload)
          return { ok: true }
        }
      })
    ).rejects.toMatchObject({ code: SESSION_CHANGED })
    expect(posts).toEqual([])
  })

  test('discards a response that arrives after the session changes', async () => {
    let session = 'visitor'
    await expect(
      runChatInit({
        data: { message: 'hi' },
        getSession: () => session,
        isVisitor: () => true,
        getSiteKey: () => '',
        requestToken: async () => 'token-1',
        refreshSiteKey: async () => {},
        post: async () => {
          session = 'signed-in'
          return { ok: true, conversation: { uuid: 'stale' } }
        }
      })
    ).rejects.toMatchObject({ code: SESSION_CHANGED })
  })
})
