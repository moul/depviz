export const updateBrowserHistory = (url) => {
  window.history.replaceState({}, 'DepViz - Dependecy Visualization', url)
}

export const generateUrl = (data) => {
  // construct url
  let url = '?'
  Object.keys(data).map((key, index) => {
    if (data[key] && data[key] !== undefined) {
      if (key === 'targets') {
        url += `${data[key].split(',').map((target) => `targets=${target.trim()}`).join('&')}`
      } else {
        url += `${url.length === 1 ? '' : '&'}${key}=${data[key]}`
      }
    }
  })

  /* if (targets) {
    url += `${targets.split(',').map((target) => `targets=${target.trim()}`).join('&')}`
  }
  // url += `&withClosed=${withClosed}&withIsolated=${withIsolated}&withPrs=${withPrs}&withoutExternal-deps=${withExternalDeps}&layout=${layout}&auth=${auth}`
  url += `&withClosed=${withClosed}&withoutIsolated=${withoutIsolated}&withoutPrs=${withoutPrs}&withoutExternal-deps=${withoutExternalDeps}&layout=${layout}` */

  return url
}
