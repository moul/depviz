import axios from 'axios'
import store from '../utils/store'

const baseApi = axios.create({
  baseURL: process.env.API_URL,
})

// Add a response interceptor
baseApi.interceptors.request.use(
  (config) => {
    const newConfig = config
    const auth = store.getItem('auth_token')
    newConfig.headers.Authorization = `Basic ${btoa(`depviz:${auth}`)}`
    return newConfig
  },
)

// Add a response interceptor
baseApi.interceptors.response.use((response) => response,
  (error) => {
    const newError = error
    const status = newError.response ? newError.response.status : null

    if (status === 401) {
      const auth = process.env.AUTH_TOKEN
      store.setItem('auth_token', auth)
      newError.config.headers.Authorization = `Basic ${btoa(`depviz:${auth}`)}`
      return baseApi.request(newError.config)
    // });
    }
    console.error('failed', newError, status, newError)
    alert(`failed: ${newError}`)
    return Promise.reject(newError)
  })

export default baseApi
