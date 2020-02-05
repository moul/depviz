import axios from "axios";

const witAuth = function(config) {
  //FIXME: Check auth sending on requests
  const au = false;
	if (au) {
		config.headers.Authorization = au;
	}
	return config;
};

export const baseApi = axios.create({
	baseURL: process.env.API_URL
});

// Authenticated routes
baseApi.interceptors.request.use(witAuth);
