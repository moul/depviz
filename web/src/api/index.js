import axios from 'axios'

export const baseApi = axios.create({
  baseURL: process.env.API_URL,
})

// Authenticated routes
/* baseApi.interceptors.request.use(
  (config) => {
    config.headers.common.Authorization = `Basic ${btoa(`depviz:${config.params.auth}`)}`
    return config
  },
  (error) => {
    // Any status codes that falls outside the range of 2xx cause this function to trigger
    // Do something with response error
    if (error.status === 401) {
      Promise.reject(error)
    }

    console.error('failed', error, status, error)
    alert(`failed: ${error}`)
  },
) */

// Add a response interceptor
baseApi.interceptors.response.use((response) => response,
  (error) => {
    const status = error.response ? error.response.status : null

    if (status === 401) {
      const auth = 'd3pviz' // FIXME: remove hardcoded value
      // return refreshToken(store, _ => {
      error.config.headers.Authorization = `Basic ${btoa(`depviz:${auth}`)}`
      // error.config.baseURL = undefined
      return baseApi.request(error.config)
    // });
    }
    console.error('failed', error, status, error)
    alert(`failed: ${error}`)
    return Promise.reject(error)
  })
