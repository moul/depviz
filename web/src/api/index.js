import axios from "axios";

const witAuth = function(config) {
  //FIXME: Check auth sending on requests
	const { au } = {};
	if (au) {
		config.headers.Authorization = au;
	}
	return config;
};

export const baseApi = axios.create({
	baseURL: "https://depviz-demo.moul.io/api"
});

// Authenticated routes
baseApi.interceptors.request.use(witAuth);
