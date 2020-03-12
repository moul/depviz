export const updateBrowserHistory = (url) => {
  window.history.replaceState({}, 'DepViz - Dependecy Visualization', url)
}

export const generateUrl = (data) => {
  const {
    targets,
    withClosed,
    withIsolated,
    withPrs,
    withExternalDeps,
    layout,
    auth = 'd3pviz',
  } = data

  // construct url
  let url = '?'
  if (targets) {
    url += `${targets.split(',').map((target) => `targets=${target.trim()}`).join('&')}`
  }
  // url += `&withClosed=${withClosed}&withIsolated=${withIsolated}&withPrs=${withPrs}&withoutExternal-deps=${withExternalDeps}&layout=${layout}&auth=${auth}`
  url += `&withClosed=${withClosed}&withIsolated=${withIsolated}&withPrs=${withPrs}&withoutExternal-deps=${withExternalDeps}&layout=${layout}&auth=${auth}`

  return url
}
