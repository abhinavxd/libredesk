// TODO: This should support recursive ZodEffects but TypeScript doesn't allow circular type definitions.

/**
 * Beautify a camelCase string.
 * e.g. "myString" -> "My String"
 */
export function beautifyObjectName(string) {
  // Remove bracketed indices
  // if numbers only return the string
  let output = string.replace(/\[\d+\]/g, '').replace(/([A-Z])/g, ' $1')
  output = output.charAt(0).toUpperCase() + output.slice(1)
  return output
}

/**
 * Parse string and extract the index
 * @param string
 * @returns index or undefined
 */
export function getIndexIfArray(string) {
  const indexRegex = /\[(\d+)\]/
  // Match the index
  const match = string.match(indexRegex)
  // Extract the index (number)
  const index = match ? Number.parseInt(match[1]) : undefined
  return index
}

/**
 * Get the lowest level Zod type.
 * This will unpack optionals, refinements, etc.
 */
export function getBaseSchema(schema) {
  if (!schema) return null
  if ('innerType' in schema._def) return getBaseSchema(schema._def.innerType)

  if ('schema' in schema._def) return getBaseSchema(schema._def.schema)

  return schema
}

/**
 * Get the type name of the lowest level Zod type.
 * This will unpack optionals, refinements, etc.
 */
export function getBaseType(schema) {
  const baseSchema = getBaseSchema(schema)
  return baseSchema ? baseSchema._def.typeName : ''
}

/**
 * Search for a "ZodDefault" in the Zod stack and return its value.
 */
export function getDefaultValueInZodStack(schema) {
  const typedSchema = schema

  if (typedSchema._def.typeName === 'ZodDefault') return typedSchema._def.defaultValue()

  if ('innerType' in typedSchema._def) {
    return getDefaultValueInZodStack(typedSchema._def.innerType)
  }
  if ('schema' in typedSchema._def) {
    return getDefaultValueInZodStack(typedSchema._def.schema)
  }

  return undefined
}

export function getObjectFormSchema(schema) {
  if (schema?._def.typeName === 'ZodEffects') {
    const typedSchema = schema
    return getObjectFormSchema(typedSchema._def.schema)
  }
  return schema
}

function isIndex(value) {
  return Number(value) >= 0
}
/**
 * Constructs a path with dot paths for arrays to use brackets to be compatible with vee-validate path syntax
 */
export function normalizeFormPath(path) {
  const pathArr = path.split('.')
  if (!pathArr.length) return ''

  let fullPath = String(pathArr[0])
  for (let i = 1; i < pathArr.length; i++) {
    if (isIndex(pathArr[i])) {
      fullPath += `[${pathArr[i]}]`
      continue
    }

    fullPath += `.${pathArr[i]}`
  }

  return fullPath
}

/**
 * Checks if the path opted out of nested fields using `[fieldName]` syntax
 */
export function isNotNestedPath(path) {
  return /^\[.+\]$/.test(path)
}
function isObject(obj) {
  return obj !== null && !!obj && typeof obj === 'object' && !Array.isArray(obj)
}
function isContainerValue(value) {
  return isObject(value) || Array.isArray(value)
}
function cleanupNonNestedPath(path) {
  if (isNotNestedPath(path)) return path.replace(/\[|\]/g, '')

  return path
}

/**
 * Gets a nested property value from an object
 */
export function getFromPath(object, path, fallback) {
  if (!object) return fallback

  if (isNotNestedPath(path)) return object[cleanupNonNestedPath(path)]

  const resolvedValue = (path || '')
    .split(/\.|\[(\d+)\]/)
    .filter(Boolean)
    .reduce((acc, propKey) => {
      if (isContainerValue(acc) && propKey in acc) return acc[propKey]

      return fallback
    }, object)

  return resolvedValue
}
