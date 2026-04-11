export const AUTH_STORAGE_KEY = 'neko_auth_key'
export const AUTH_ROUTE_PATH = '/auth'
export const USER_AUTH_HEADER = 'X-Neko-Auth-Key'

export function getStoredAuthKey() {
  return String(window.localStorage.getItem(AUTH_STORAGE_KEY) || '').trim()
}

export function setStoredAuthKey(authKey) {
  window.localStorage.setItem(AUTH_STORAGE_KEY, String(authKey || '').trim())
}

export function clearStoredAuthKey() {
  window.localStorage.removeItem(AUTH_STORAGE_KEY)
}

export function logoutAndRedirect(redirectPath = '/') {
  clearStoredAuthKey()
  redirectToAuth(redirectPath)
}

export function buildAuthRouteTarget(redirectPath = '/') {
  const normalizedRedirect = typeof redirectPath === 'string' && redirectPath ? redirectPath : '/'
  return `${AUTH_ROUTE_PATH}?redirect=${encodeURIComponent(normalizedRedirect)}`
}

export function currentRoutePath() {
  const basePath = '/static'
  const pathname = window.location.pathname.startsWith(basePath)
    ? window.location.pathname.slice(basePath.length) || '/'
    : window.location.pathname || '/'
  return `${pathname}${window.location.search}${window.location.hash}` || '/'
}

export function redirectToAuth(redirectPath = currentRoutePath()) {
  const target = `${'/static'}${buildAuthRouteTarget(redirectPath)}`
  if (window.location.pathname + window.location.search === target) {
    return
  }
  window.location.replace(target)
}

export function installAuthFetchInterceptor() {
  if (window.__nekoAuthFetchInstalled) {
    return
  }
  const originalFetch = window.fetch.bind(window)
  window.fetch = async (input, init) => {
    let request = input instanceof Request ? new Request(input, init) : new Request(input, init)
    const requestUrl = new URL(request.url, window.location.origin)
    const protectedApi = isProtectedApiRequest(requestUrl)
    if (protectedApi) {
      const authKey = getStoredAuthKey()
      if (authKey) {
        const headers = new Headers(request.headers)
        headers.set(USER_AUTH_HEADER, authKey)
        request = new Request(request, { headers })
      }
    }
    const response = await originalFetch(request)
    if (protectedApi && await isAuthFailureResponse(response)) {
      clearStoredAuthKey()
      redirectToAuth()
    }
    return response
  }
  window.__nekoAuthFetchInstalled = true
}

function isProtectedApiRequest(url) {
  if (url.origin !== window.location.origin) {
    return false
  }
  if (!url.pathname.startsWith('/api/')) {
    return false
  }
  if (url.pathname.startsWith('/api/internal/')) {
    return false
  }
  if (url.pathname.startsWith('/api/auth/login')) {
    return false
  }
  return true
}

async function isAuthFailureResponse(response) {
  if (response.status === 401) {
    return true
  }
  const contentType = response.headers.get('content-type') || ''
  if (!contentType.includes('application/json')) {
    return false
  }
  try {
    const body = await response.clone().json()
    return body?.code === 401
  } catch {
    return false
  }
}