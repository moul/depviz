import { baseApi } from "./index";

export function fetchDepviz(url) {
	return baseApi.get(`${url}`);
}
