import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { apiClient, apiRequest } from '~/lib/api';
import { toast } from 'sonner';

interface AuthContextType {
	isAuthenticated: boolean;
	isLoading: boolean;
	login: (password: string) => Promise<boolean>;
	logout: () => Promise<void>;
	checkAuth: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
	const [isAuthenticated, setIsAuthenticated] = useState(false);
	const [isLoading, setIsLoading] = useState(true);
	const [token, setToken] = useState<string | null>(null);

	// Check if user is authenticated on mount
	useEffect(() => {
		checkAuth();
	}, []);

	// Set auth header whenever token changes
	useEffect(() => {
		if (token) {
			apiClient.defaults.headers.common['Authorization'] = `Bearer ${token}`;
			localStorage.setItem('auth_token', token);
		} else {
			delete apiClient.defaults.headers.common['Authorization'];
			localStorage.removeItem('auth_token');
		}
	}, [token]);

	const checkAuth = async () => {
		try {
			const storedToken = localStorage.getItem('auth_token');
			if (!storedToken) {
				setIsAuthenticated(false);
				setIsLoading(false);
				return;
			}

			// Validate token with backend
			const response = await apiClient.get('/auth/validate', {
				headers: {
					Authorization: `Bearer ${storedToken}`
				}
			});

			if (response.data.valid) {
				setToken(storedToken);
				setIsAuthenticated(true);
			} else {
				setToken(null);
				setIsAuthenticated(false);
			}
		} catch (error) {
			console.error('Auth check failed:', error);
			setToken(null);
			setIsAuthenticated(false);
		} finally {
			setIsLoading(false);
		}
	};

	const login = async (password: string): Promise<boolean> => {
		try {
			const response = await apiRequest<{ success: boolean; token: string; message?: string }>({
				method: 'post',
				url: '/auth/login',
				data: { password }
			});

			if (response.success) {
				setToken(response.token);
				setIsAuthenticated(true);
				toast.success('Logged in successfully');
				return true;
			} else {
				toast.error(response.message || 'Login failed');
				return false;
			}
		} catch (error: any) {
			const message = error.response?.message || 'Invalid password';
			toast.error(message);
			return false;
		}
	};

	const logout = async () => {
		try {
			if (token) {
				await apiRequest({
					method: 'post',
					url: '/auth/logout',
					headers: {
						Authorization: `Bearer ${token}`
					}
				});
			}
		} catch (error) {
			console.error('Logout error:', error);
		} finally {
			setToken(null);
			setIsAuthenticated(false);
			toast.success('Logged out successfully');
		}
	};

	return (
		<AuthContext.Provider value={{
			isAuthenticated,
			isLoading,
			login,
			logout,
			checkAuth,
		}}>
			{children}
		</AuthContext.Provider>
	);
}

export function useAuth() {
	const context = useContext(AuthContext);
	if (context === undefined) {
		throw new Error('useAuth must be used within an AuthProvider');
	}
	return context;
}
