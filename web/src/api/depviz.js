import baseApi from './index'

const fetchDepviz = (url, params) => baseApi.get(`${url}`, params)

export default fetchDepviz
