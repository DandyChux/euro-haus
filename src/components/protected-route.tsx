import { Navigate } from '@tanstack/react-router';
import { useAuth } from '~/lib/contexts/auth-context';
import { Loader2 } from 'lucide-react';

interface ProtectedRouteProps {
	children: React.ReactNode;
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
	const { isAuthenticated, isLoading } = useAuth();

	if (isLoading) {
		return (
			<div className="min-h-screen flex items-center justify-center">
				<Loader2 className="h-8 w-8 animate-spin" />
			</div>
		);
	}

	if (!isAuthenticated) {
		return <Navigate to="/admin/login" />;
	}

	return <>{children}</>;
}
