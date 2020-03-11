import { baseApi } from './index'

export function fetchDepviz(url, params) {
  return baseApi.get(`${url}`, params)
}
