import axios from 'axios'
import store from '../utils/store'

let retryCounter = 0

const baseApi = axios.create({
  baseURL: process.env.API_URL,
})

// Add a response interceptor
baseApi.interceptors.request.use(
  (config) => {
    const newConfig = config
    newConfig.headers.Authorization = `Basic ${btoa(`depviz:${store.getItem('auth_token')}`)}`
    return newConfig
  },
)

// Add a response interceptor
baseApi.interceptors.response.use((response) => response,
  (error) => {
    const newError = error
    const status = newError.response ? newError.response.status : null

    if (status === 401) {
      if (process.env.AUTH_TOKEN) {
        store.setItem('auth_token', process.env.AUTH_TOKEN)
        newError.config.headers.Authorization = `Basic ${btoa(`depviz:${process.env.AUTH_TOKEN}`)}`
      }
      retryCounter += 1
      if (retryCounter < 4) { // Allow 3 attempts to request
        return baseApi.request(newError.config)
      }
    }
    console.error('failed', newError, status, newError)
    alert(`failed: ${newError}`)
    return Promise.reject(newError)
  })

export default baseApi
