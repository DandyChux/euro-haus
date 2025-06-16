import axios, { AxiosError, AxiosRequestConfig } from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080/api";

export const apiClient = axios.create({
	baseURL: API_BASE_URL,
	headers: {
		'Content-Type': 'application/json'
	}
});

// Store auth context reference for interceptors
let authContextRef: any = null;

export const setAuthContext = (authContext: any) => {
	authContextRef = authContext;
};

// Add a request interceptor for auth
apiClient.interceptors.request.use(
	(config) => {
		const token = localStorage.getItem('accessToken');
		if (token) {
			config.headers.Authorization = `Bearer ${token}`;
		}
		return config;
	}, (error) => Promise.reject(error)
);

// Add a response interceptor to handle token refresh
apiClient.interceptors.response.use(
	(response) => response,
	async (error) => {
		const originalRequest = error.config;

		if (!originalRequest) return Promise.reject(error);

		// If the error is 401 and hasn't been retried yet
		if (error.response?.status === 401 && !originalRequest._retry) {
			originalRequest._retry = true;

			// Try to refresh the token
			const refreshToken = localStorage.getItem('refreshToken');
			if (refreshToken) {
				try {
					const response = await axios.post(`${API_BASE_URL}/auth/refresh`, {
						refreshToken
					});

					// Update tokens through auth context if available
					if (authContextRef) {
						authContextRef.updateTokens({
							accessToken: response.data.accessToken,
							refreshToken: response.data.refreshToken,
						});
					} else {
						// Fallback to localStorage
						localStorage.setItem('accessToken', response.data.accessToken);
						localStorage.setItem('refreshToken', response.data.refreshToken);
					}

					// Retry the original request with the new token
					originalRequest.headers.Authorization = `Bearer ${response.data.accessToken}`;
					return axios(originalRequest);
				} catch (refreshError) {
					// Refresh failed, logout user
					if (authContextRef) {
						authContextRef.logout();
					} else {
						localStorage.removeItem('accessToken');
						localStorage.removeItem('refreshToken');
						window.location.href = '/admin/login';
					}
				}
			} else {
				// No refresh token, logout user
				if (authContextRef) {
					authContextRef.logout();
				} else {
					window.location.href = '/admin/login';
				}
			}
		}

		return Promise.reject(error);
	}
);

// Generic request function for React Query
export async function apiRequest<T>(
	config: AxiosRequestConfig
): Promise<T> {
	try {
		const response = await apiClient(config);
		return response.data as T;
	} catch (error) {
		const axiosError = error as AxiosError;
		// Handle specific cases here
		throw axiosError;
	}
}
