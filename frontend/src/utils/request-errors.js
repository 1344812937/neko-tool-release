function stringifyError(errorLike) {
  if (!errorLike) {
    return ''
  }
  if (typeof errorLike === 'string') {
    return errorLike.trim()
  }
  if (typeof errorLike.message === 'string') {
    return errorLike.message.trim()
  }
  return String(errorLike).trim()
}

function stripRequestPrefix(message) {
  return message
    .replace(/^(Get|Post|Put|Delete|Patch|Head)\s+"[^"]+":\s*/i, '')
    .replace(/^request failed:\s*/i, '')
    .trim()
}

function normalizeWhitespace(message) {
  return message.replace(/\s+/g, ' ').trim()
}

function extractHttpStatus(message) {
  const match = message.match(/HTTP\s+(\d{3})/i)
  if (!match) {
    return 0
  }
  return Number(match[1])
}

function formatHttpStatusMessage(status) {
  switch (status) {
    case 400:
      return '请求参数不正确，请检查节点地址或请求内容。'
    case 401:
      return '远程节点鉴权失败，请检查共享令牌是否与对端配置一致。'
    case 403:
      return '远程节点拒绝访问，请检查对端授权配置。'
    case 404:
      return '远程节点接口不存在，请确认对端服务版本和地址是否正确。'
    case 408:
      return '远程节点响应超时，请稍后重试。'
    case 429:
      return '远程节点请求过于频繁，请稍后再试。'
    case 500:
      return '远程节点内部处理失败，请检查对端服务日志。'
    case 502:
    case 503:
    case 504:
      return '远程节点当前不可用，可能是服务未启动、网关异常或暂时过载。'
    default:
      return ''
  }
}

function formatUnexpectedMessage(message) {
  const cleaned = normalizeWhitespace(stripRequestPrefix(message))
  if (!cleaned) {
    return '未知错误'
  }
  return cleaned
    .replace(/^dial tcp\s+/i, 'TCP 连接失败：')
    .replace(/^read tcp\s+/i, '读取响应失败：')
    .replace(/^write tcp\s+/i, '发送请求失败：')
}

export function formatRequestError(errorLike, fallback = '请求失败') {
  const rawMessage = stringifyError(errorLike)
  if (!rawMessage) {
    return fallback
  }

  const message = normalizeWhitespace(rawMessage)
  const simplified = stripRequestPrefix(message)
  const lower = simplified.toLowerCase()

  if (lower.includes('返回了 html 页面') || lower.includes('returned html page')) {
    return '节点地址返回了页面内容，请填写服务根地址，例如 http://127.0.0.1:8888，不要带 /static 或页面路径。'
  }

  if (lower.includes('内部节点鉴权失败') || lower.includes('unauthorized') || lower.includes('forbidden')) {
    return '远程节点鉴权失败，请检查共享令牌、节点启用状态和双方时间是否一致。'
  }

  if (lower.includes('connection refused')) {
    return '目标地址拒绝连接，请确认远程服务已启动，且地址和端口填写正确。'
  }

  if (lower.includes('no such host') || lower.includes('server misbehaving') || lower.includes('name or service not known')) {
    return '节点地址无法解析，请检查域名、主机名或 DNS 配置。'
  }

  if (lower.includes('i/o timeout') || lower.includes('context deadline exceeded') || lower.includes('client.timeout exceeded') || lower.includes('timeout awaiting response headers')) {
    return '连接远程节点超时，请检查网络连通性、防火墙或服务响应时间。'
  }

  if (lower.includes('no route to host') || lower.includes('network is unreachable')) {
    return '当前网络无法到达该节点，请检查路由、防火墙、代理或 VPN 配置。'
  }

  if (lower.includes('certificate signed by unknown authority') || lower.includes('x509:')) {
    return '远程节点证书校验失败，请检查 HTTPS 证书是否有效或改用正确的访问方式。'
  }

  if (lower.includes('server gave http response to https client') || lower.includes('first record does not look like a tls handshake')) {
    return '节点地址的协议与服务不匹配，请确认应该使用 http 还是 https。'
  }

  if (lower === 'eof' || lower.includes('unexpected eof')) {
    return '远程连接在响应完成前被中断，请检查对端服务、反向代理或网络稳定性。'
  }

  if (lower.includes('远程节点返回空响应')) {
    return '远程节点返回了空响应，请检查对端服务是否正常。'
  }

  const status = extractHttpStatus(simplified)
  if (status) {
    const formattedStatusMessage = formatHttpStatusMessage(status)
    if (formattedStatusMessage) {
      return formattedStatusMessage
    }
  }

  return `${fallback}：${formatUnexpectedMessage(message)}`
}