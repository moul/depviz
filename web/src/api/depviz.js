import baseApi from './index'

const fetchDepviz = (url, params) => baseApi.get(`/api${url}`, params)
const getToken = (url, params) => baseApi.post(`${url}`, params)

export {
  fetchDepviz,
  getToken
}
