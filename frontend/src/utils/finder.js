// 统一把路径归一化成前端树结构使用的相对路径格式。
function normalizePath(path) {
  return (path || '').replace(/\\/g, '/').replace(/^\//, '').replace(/\/$/, '')
}

// 计算相对路径的层级深度。
function getPathDepth(path) {
  const normalizedPath = normalizePath(path)
  if (!normalizedPath) {
    return 0
  }
  return normalizedPath.split('/').length
}

// 判断某个路径是否与基路径相同或位于其子层级中。
function isSameOrDescendant(path, basePath) {
  const normalizedPath = normalizePath(path)
  const normalizedBasePath = normalizePath(basePath)
  if (!normalizedBasePath) {
    return true
  }
  return normalizedPath === normalizedBasePath || normalizedPath.startsWith(`${normalizedBasePath}/`)
}

// 计算路径相对于基路径的深度差，用于控制目录展开层级。
function relativeDepth(path, basePath) {
  const normalizedPath = normalizePath(path)
  const normalizedBasePath = normalizePath(basePath)
  if (!normalizedBasePath) {
    return getPathDepth(normalizedPath)
  }
  if (normalizedPath === normalizedBasePath) {
    return 0
  }
  if (!normalizedPath.startsWith(`${normalizedBasePath}/`)) {
    return -1
  }
  return normalizedPath.slice(normalizedBasePath.length + 1).split('/').length
}

// 从相对路径中取最后一级名称。
function pathName(path) {
  const normalizedPath = normalizePath(path)
  if (!normalizedPath) {
    return ''
  }
  const parts = normalizedPath.split('/')
  return parts[parts.length - 1]
}

// 创建 Finder/树视图通用节点对象。
function createNode(data) {
  return {
    name: data.name,
    path: data.path,
    parentPath: data.parentPath || '',
    aliases: data.aliases || (data.path ? [data.path] : []),
    entryType: data.entryType || 'directory',
    status: data.status || '',
    hash: data.hash || '',
    size: data.size || 0,
    modifyTime: data.modifyTime || 0,
    children: data.children || [],
    hasChildren: data.hasChildren ?? false,
    deleted: data.deleted ?? false,
    childrenLoaded: data.childrenLoaded ?? false,
    loading: data.loading ?? false,
    compacted: data.compacted ?? false,
  }
}

// 确保路径上的目录节点都已经在树结构中创建出来。
function ensureDirectory(rootNodes, nodeMap, path) {
  const normalizedPath = normalizePath(path)
  if (!normalizedPath) {
    return null
  }
  if (nodeMap.has(normalizedPath)) {
    return nodeMap.get(normalizedPath)
  }
  const parts = normalizedPath.split('/')
  const name = parts[parts.length - 1]
  const parentPath = parts.length > 1 ? parts.slice(0, -1).join('/') : ''
  const parentNode = ensureDirectory(rootNodes, nodeMap, parentPath)
  const node = createNode({
    name,
    path: normalizedPath,
    parentPath,
    entryType: 'directory',
    status: 'context',
    hasChildren: true,
  })
  nodeMap.set(normalizedPath, node)
  if (parentNode) {
    parentNode.children.push(node)
    parentNode.hasChildren = true
  } else {
    rootNodes.push(node)
  }
  return node
}

// 对同级节点做目录优先、名称排序。
function sortNodes(nodes) {
  nodes.sort((left, right) => {
    if (left.entryType !== right.entryType) {
      return left.entryType === 'directory' ? -1 : 1
    }
    return left.name.localeCompare(right.name, 'zh-Hans-CN')
  })
  nodes.forEach((node) => {
    if (node.children?.length) {
      sortNodes(node.children)
    }
  })
  return nodes
}

// 将接口返回的目录项标准化成前端树节点可消费的结构。
function normalizeManifestEntry(entry) {
  const relativePath = normalizePath(entry.relativePath || entry.path)
  if (!relativePath) {
    return null
  }
  return {
    relativePath,
    name: entry.name || pathName(relativePath),
    entryType: entry.entryType || 'directory',
    hash: entry.hash || '',
    size: entry.size || 0,
    modifyTime: entry.modifyTime || 0,
    hasChildren: Boolean(entry.hasChildren),
    deleted: Boolean(entry.deleted),
    childrenLoaded: Boolean(entry.childrenLoaded),
    loading: Boolean(entry.loading),
  }
}

// 根据当前 manifest 上下文补充 childrenLoaded 等派生状态。
function materializeManifestEntry(entry, manifest) {
  const normalized = normalizeManifestEntry(entry)
  if (!normalized) {
    return null
  }
  if (normalized.entryType === 'directory') {
    const currentRelativeDepth = relativeDepth(normalized.relativePath, manifest?.basePath)
    if (currentRelativeDepth >= 0 && currentRelativeDepth < (manifest?.depth || 3)) {
      normalized.childrenLoaded = true
    }
  } else {
    normalized.childrenLoaded = false
  }
  normalized.loading = false
  return normalized
}

// 用已有 entries 快速构造路径索引，便于后续合并。
function buildEntryMap(entries) {
  const entryMap = new Map()
  ;(entries || []).forEach((entry) => {
    const normalized = normalizeManifestEntry(entry)
    if (normalized) {
      entryMap.set(normalized.relativePath, normalized)
    }
  })
  return entryMap
}

export function mergeManifestEntries(existingEntries, manifest, options = {}) {
  const { replace = false } = options
  const nextMap = replace ? new Map() : buildEntryMap(existingEntries)
  const basePath = normalizePath(manifest?.basePath)
  if (!replace) {
    Array.from(nextMap.keys()).forEach((path) => {
      if (basePath && isSameOrDescendant(path, basePath)) {
        nextMap.delete(path)
      }
    })
  }
  ;(manifest?.entries || []).forEach((entry) => {
    const normalized = materializeManifestEntry(entry, manifest)
    if (!normalized) {
      return
    }
    const existing = nextMap.get(normalized.relativePath)
    nextMap.set(normalized.relativePath, {
      ...existing,
      ...normalized,
      childrenLoaded: Boolean(existing?.childrenLoaded || normalized.childrenLoaded),
      loading: false,
    })
  })
  return Array.from(nextMap.values()).sort((left, right) => left.relativePath.localeCompare(right.relativePath, 'zh-Hans-CN'))
}

export function updateManifestEntryState(entries, path, patch) {
  const normalizedPath = normalizePath(path)
  return (entries || []).map((entry) => {
    if (normalizePath(entry.relativePath) !== normalizedPath) {
      return entry
    }
    return { ...entry, ...patch }
  })
}

export function shouldLoadChildren(node) {
  return Boolean(node && node.entryType === 'directory' && node.hasChildren && !node.deleted && !node.childrenLoaded)
}

// 从紧凑目录链里挑第一个有效字段值作为聚合结果。
function firstMeaningfulValue(nodes, field) {
  for (const node of nodes) {
    if (node?.[field]) {
      return node[field]
    }
  }
  return ''
}

// 从紧凑目录链里挑一个最有代表性的状态。
function firstMeaningfulStatus(nodes) {
  for (const node of nodes) {
    if (node?.status && node.status !== 'context') {
      return node.status
    }
  }
  return nodes[nodes.length - 1]?.status || ''
}

// 合并紧凑目录链涉及到的所有别名路径。
function mergeAliases(nodes) {
  return Array.from(new Set(
    (nodes || []).flatMap((node) => {
      if (node?.aliases?.length) {
        return node.aliases
      }
      return node?.path ? [node.path] : []
    }),
  ))
}

// 将仅有单目录子级的链式目录压缩成 Finder 紧凑节点。
function compactNode(node) {
  const nextChildren = (node.children || []).map((child) => compactNode(child))
  const baseNode = {
    ...node,
    aliases: node.aliases?.length ? [...node.aliases] : (node.path ? [node.path] : []),
    children: nextChildren,
    hasChildren: node.entryType === 'directory' ? Boolean(node.hasChildren || nextChildren.length > 0) : false,
  }

  if (baseNode.entryType !== 'directory') {
    return baseNode
  }

  const chainNodes = [baseNode]
  let current = baseNode
  while (current.children?.length === 1 && current.children[0].entryType === 'directory') {
    current = current.children[0]
    chainNodes.push(current)
  }

  if (chainNodes.length === 1) {
    return baseNode
  }

  const compactedName = chainNodes.map((item) => item.name).join('.')
  const tailNode = chainNodes[chainNodes.length - 1]
  const aliases = mergeAliases(chainNodes)
  return {
    ...tailNode,
    name: compactedName,
    compactedName,
    compacted: true,
    aliases,
    compactedPaths: aliases,
    parentPath: baseNode.parentPath,
    status: firstMeaningfulStatus(chainNodes),
    hash: firstMeaningfulValue(chainNodes, 'hash') || tailNode.hash || '',
    leftHash: firstMeaningfulValue(chainNodes, 'leftHash') || tailNode.leftHash || '',
    rightHash: firstMeaningfulValue(chainNodes, 'rightHash') || tailNode.rightHash || '',
    children: tailNode.children || [],
    hasChildren: Boolean(tailNode.hasChildren || (tailNode.children || []).length > 0),
    childrenLoaded: Boolean(tailNode.childrenLoaded),
    loading: chainNodes.some((item) => item.loading),
  }
}

// 批量压缩整棵树中的链式目录结构。
function compactDirectoryChains(nodes) {
  return (nodes || []).map((node) => compactNode(node))
}

export function buildManifestTree(entries) {
  const rootNodes = []
  const nodeMap = new Map()
  const sortedEntries = [...(entries || [])].sort((left, right) => left.relativePath.localeCompare(right.relativePath, 'zh-Hans-CN'))
  sortedEntries.forEach((entry) => {
    const normalizedPath = normalizePath(entry.relativePath)
    if (!normalizedPath) {
      return
    }
    const parentPath = normalizedPath.includes('/') ? normalizedPath.split('/').slice(0, -1).join('/') : ''
    const parentNode = ensureDirectory(rootNodes, nodeMap, parentPath)
    const existing = nodeMap.get(normalizedPath)
    const node = existing || createNode({
      name: entry.name,
      path: normalizedPath,
      parentPath,
      children: [],
    })
    node.name = entry.name
    node.aliases = [normalizedPath]
    node.entryType = entry.entryType
    node.hash = entry.hash
    node.size = entry.size
    node.modifyTime = entry.modifyTime
    node.hasChildren = entry.entryType === 'directory' ? Boolean(entry.hasChildren) : false
    node.deleted = Boolean(entry.deleted)
    node.childrenLoaded = Boolean(entry.childrenLoaded)
    node.loading = Boolean(entry.loading)
    nodeMap.set(normalizedPath, node)
    if (!existing) {
      if (parentNode) {
        parentNode.children.push(node)
        parentNode.hasChildren = true
      } else {
        rootNodes.push(node)
      }
    }
  })
  return sortNodes(compactDirectoryChains(rootNodes))
}

export function buildCompareTree(items) {
  const rootNodes = []
  const nodeMap = new Map()
  ;(items || []).forEach((item) => {
    const normalizedPath = normalizePath(item.path)
    if (!normalizedPath) {
      return
    }
    const parentPath = normalizedPath.includes('/') ? normalizedPath.split('/').slice(0, -1).join('/') : ''
    const parentNode = ensureDirectory(rootNodes, nodeMap, parentPath)
    const existing = nodeMap.get(normalizedPath)
    const node = existing || createNode({
      name: item.name,
      path: normalizedPath,
      parentPath,
      children: [],
    })
    node.name = item.name
    node.aliases = [normalizedPath]
    node.entryType = item.entryType
    node.status = item.status
    node.hash = item.leftHash || item.rightHash || ''
    node.leftHash = item.leftHash || ''
    node.rightHash = item.rightHash || ''
    node.leftSize = item.leftSize || 0
    node.rightSize = item.rightSize || 0
    node.hasChildren = item.entryType === 'directory' && node.children.length > 0
    nodeMap.set(normalizedPath, node)
    if (!existing) {
      if (parentNode) {
        parentNode.children.push(node)
        parentNode.hasChildren = true
      } else {
        rootNodes.push(node)
      }
    }
  })
  return sortNodes(compactDirectoryChains(rootNodes))
}

export function indexFinderTree(nodes) {
  const map = new Map()
  function visit(list) {
    ;(list || []).forEach((node) => {
      map.set(node.path, node)
      ;(node.aliases || []).forEach((alias) => {
        map.set(alias, node)
      })
      if (node.children?.length) {
        visit(node.children)
      }
    })
  }
  visit(nodes)
  return map
}

export function buildFinderColumns(nodes, selectedPath) {
  const normalizedPath = normalizePath(selectedPath)
  const nodeMap = indexFinderTree(nodes)
  const columns = [{ key: 'root', title: '项目', items: nodes }]
  if (!normalizedPath) {
    return columns
  }
  const selectedNode = nodeMap.get(normalizedPath)
  if (!selectedNode) {
    return columns
  }
  const ancestors = []
  let current = selectedNode
  while (current?.parentPath) {
    ancestors.unshift(current.parentPath)
    current = nodeMap.get(current.parentPath)
  }
  ancestors.forEach((path) => {
    const node = nodeMap.get(path)
    if (node?.children?.length) {
      columns.push({ key: path, title: node.name, items: node.children })
    }
  })
  if (selectedNode?.entryType === 'directory' && selectedNode.children?.length) {
    columns.push({ key: `${selectedNode.path}:children`, title: selectedNode.name, items: selectedNode.children })
  }
  return columns
}

export function getNodeByPath(nodes, path) {
  const normalizedPath = normalizePath(path)
  if (!normalizedPath) {
    return null
  }
  return indexFinderTree(nodes).get(normalizedPath) || null
}