import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useState } from 'react';
import { Button } from '~/components/ui/button';
import { Input } from '~/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '~/components/ui/card';
import { Label } from '~/components/ui/label';
import { useAuth } from '~/lib/contexts/auth-context';
import { Loader2 } from 'lucide-react';

export const Route = createFileRoute('/admin/login')({
	component: AdminLoginPage,
});

function AdminLoginPage() {
	const navigate = useNavigate();
	const { login, isAuthenticated } = useAuth();
	const [password, setPassword] = useState('');
	const [isLoading, setIsLoading] = useState(false);

	// Redirect if already authenticated
	if (isAuthenticated) {
		navigate({ to: `${import.meta.env.VITE_API_URL}/admin/products` });
		return null;
	}

	const handleLogin = async (e: React.FormEvent) => {
		e.preventDefault();
		setIsLoading(true);

		const success = await login(password);

		if (success) {
			navigate({ to: `${import.meta.env.VITE_API_URL}/admin/products` });
		}

		setIsLoading(false);
	};

	return (
		<div className="min-h-screen flex items-center justify-center bg-background">
			<Card className="w-full max-w-md">
				<CardHeader>
					<CardTitle>Admin Access</CardTitle>
					<CardDescription>Enter your admin password to continue</CardDescription>
				</CardHeader>
				<CardContent>
					<form onSubmit={handleLogin} className="space-y-4">
						<div className="space-y-2">
							<Label htmlFor="password">Password</Label>
							<Input
								id="password"
								type="password"
								value={password}
								onChange={(e) => setPassword(e.target.value)}
								placeholder="Enter admin password"
								disabled={isLoading}
							/>
						</div>
						<Button type="submit" className="w-full" disabled={isLoading}>
							{isLoading ? (
								<>
									<Loader2 className="mr-2 h-4 w-4 animate-spin" />
									Logging in...
								</>
							) : (
								'Login'
							)}
						</Button>
					</form>
				</CardContent>
			</Card>
		</div>
	);
}
