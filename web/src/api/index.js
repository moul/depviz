import axios from 'axios'

const witAuth = (config) => {
  config.headers.common.Authorization = `Basic ${btoa(`depviz:${config.params.auth}`)}`
  return config
}

const baseApi = (url, params) => {
  const api = axios.create({
    baseURL: process.env.API_URL,
    params,
  })

  // Authenticated routes
  api.interceptors.request.use(
    witAuth,
    (error) => {
      // Do something with request error
      Promise.reject(error)
    },
  )

  // Add a response interceptor
  api.interceptors.response.use((response) =>
    // Any status code that lie within the range of 2xx cause this function to trigger
    // Do something with response data
    response,
  (error) => {
    // Any status codes that falls outside the range of 2xx cause this function to trigger
    // Do something with response error
    if (error.status == 401) {
      Promise.reject(error)
    }

    console.error('failed', error, status, error)
    alert(`failed: ${error}`)
  })
}

export default baseApi
