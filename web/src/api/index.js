import axios from 'axios'
import store from '../utils/store'

export const baseApi = axios.create({
  baseURL: process.env.API_URL,
})

// Add a response interceptor
baseApi.interceptors.request.use(
  (config) => {
    const auth = store.getItem('auth_token')
    config.headers.Authorization = `Basic ${btoa(`depviz:${auth}`)}`
    return config
  },
)

// Add a response interceptor
baseApi.interceptors.response.use((response) => response,
  (error) => {
    const status = error.response ? error.response.status : null

    if (status === 401) {
      const auth = process.env.AUTH_TOKEN
      store.setItem('auth_token', auth)
      error.config.headers.Authorization = `Basic ${btoa(`depviz:${auth}`)}`
      return baseApi.request(error.config)
    // });
    }
    console.error('failed', error, status, error)
    alert(`failed: ${error}`)
    return Promise.reject(error)
  })
