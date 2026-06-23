export function collectOAuthCallbackParams(search: string, hash: string): URLSearchParams {
  const params = new URLSearchParams(search)
  const hashQueryIndex = hash.indexOf('?')
  if (hashQueryIndex < 0) {
    return params
  }

  const hashParams = new URLSearchParams(hash.substring(hashQueryIndex + 1))
  hashParams.forEach((value, key) => {
    params.set(key, value)
  })
  return params
}
