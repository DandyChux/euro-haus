import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useState, useEffect } from 'react';
import { Button } from '~/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '~/components/ui/card';
import { Badge } from '~/components/ui/badge';
import { Skeleton } from '~/components/ui/skeleton';
import { toast } from 'sonner';
import { apiClient } from '~/lib/api';
import {
	Package,
	Image,
	Calendar,
	LogOut,
	TrendingUp,
	Users,
	DollarSign,
	FileImage,
	Settings,
	ExternalLink,
	BarChart,
	ShoppingBag
} from 'lucide-react';
import { ProtectedRoute } from '~/components/protected-route';
import { useAuth } from '~/lib/contexts/auth-context';

interface DashboardStats {
	totalProducts: number;
	totalEvents: number;
	activeProducts: number;
	featuredItems: number;
	mediaFiles?: number;
}

export const Route = createFileRoute('/admin/')({
	component: AdminDashboardPage,
});

function AdminDashboardPage() {
	return (
		<ProtectedRoute>
			<AdminDashboardContent />
		</ProtectedRoute>
	);
}

function AdminDashboardContent() {
	const navigate = useNavigate();
	const { logout } = useAuth();
	const [stats, setStats] = useState<DashboardStats | null>(null);
	const [isLoading, setIsLoading] = useState(true);

	// Fetch dashboard statistics
	const fetchStats = async () => {
		setIsLoading(true);
		try {
			const response = await apiClient.get('/products');
			const products = response.data.products || [];

			// Calculate stats
			const stats: DashboardStats = {
				totalProducts: products.filter((p: any) => p.metadata.type !== 'event').length,
				totalEvents: products.filter((p: any) => p.metadata.type === 'event').length,
				activeProducts: products.filter((p: any) => p.active).length,
				featuredItems: products.filter((p: any) => p.metadata.featured === 'true').length,
			};

			// Try to fetch media count (if endpoint exists)
			try {
				const mediaResponse = await apiClient.get('/admin/media');
				stats.mediaFiles = mediaResponse.data.files?.length || 0;
			} catch {
				// Media endpoint might not be available
			}

			setStats(stats);
		} catch (error) {
			console.error('Error fetching stats:', error);
			toast.error('Failed to load dashboard statistics');
		} finally {
			setIsLoading(false);
		}
	};

	useEffect(() => {
		fetchStats();
	}, []);

	const handleLogout = async () => {
		await logout();
		navigate({ to: '/admin/login' });
	};

	// Admin sections with navigation
	const adminSections = [
		{
			title: 'Product Management',
			description: 'Create, edit, and manage your products and merchandise',
			icon: Package,
			href: '/admin/products',
			color: 'text-blue-600',
			bgColor: 'bg-blue-50',
			stats: stats ? `${stats.totalProducts} products` : null,
		},
		{
			title: 'Event Management',
			description: 'Manage event tickets, schedules, and attendee information',
			icon: Calendar,
			href: '/admin/products',
			color: 'text-purple-600',
			bgColor: 'bg-purple-50',
			stats: stats ? `${stats.totalEvents} events` : null,
		},
		{
			title: 'Media Library',
			description: 'Upload and manage images and videos for your site',
			icon: Image,
			href: '/admin/media',
			color: 'text-green-600',
			bgColor: 'bg-green-50',
			stats: stats?.mediaFiles ? `${stats.mediaFiles} files` : null,
		},
	];

	// Quick actions
	const quickActions = [
		{
			label: 'Create Product',
			icon: Package,
			onClick: () => navigate({ to: '/admin/products', search: { tab: 'create' } }),
		},
		{
			label: 'Create Event',
			icon: Calendar,
			onClick: () => navigate({ to: '/admin/products', search: { tab: 'create', type: 'event' } }),
		},
		{
			label: 'Upload Media',
			icon: FileImage,
			onClick: () => navigate({ to: '/admin/media', search: { tab: 'upload' } }),
		},
		{
			label: 'View Site',
			icon: ExternalLink,
			onClick: () => window.open('/', '_blank'),
		},
	];

	// External links
	const externalLinks = [
		{
			title: 'Stripe Dashboard',
			description: 'View payments and customer data',
			icon: DollarSign,
			href: 'https://dashboard.stripe.com',
			color: 'text-indigo-600',
		},
		{
			title: 'Analytics',
			description: 'Track site performance and visitor data',
			icon: BarChart,
			href: 'https://plausible.blackstacksolutions.com/theeurohaus.com/',
			color: 'text-orange-600',
		},
		{
			title: 'DigitalOcean Spaces',
			description: 'Manage storage and CDN settings',
			icon: Settings,
			href: 'https://cloud.digitalocean.com/spaces',
			color: 'text-cyan-600',
		},
	];

	return (
		<div className="min-h-screen bg-background">
			<div className="max-w-7xl mx-auto p-6">
				{/* Header */}
				<div className="flex justify-between items-center mb-8">
					<div>
						<h1 className="text-3xl font-bold">Admin Dashboard</h1>
						<p className="text-muted-foreground">Welcome to Euro Haus Admin Panel</p>
					</div>
					<div className="flex gap-2">
						<Button variant="outline" onClick={() => window.open('/', '_blank')}>
							<ExternalLink className="h-4 w-4 mr-2" />
							View Site
						</Button>
						<Button variant="ghost" size="icon" onClick={handleLogout} title="Logout">
							<LogOut className="h-4 w-4" />
						</Button>
					</div>
				</div>

				{/* Stats Overview */}
				<div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
					{isLoading ? (
						[...Array(4)].map((_, i) => (
							<Skeleton key={i} className="h-24" />
						))
					) : stats ? (
						<>
							<Card>
								<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
									<CardTitle className="text-sm font-medium">Active Products</CardTitle>
									<ShoppingBag className="h-4 w-4 text-muted-foreground" />
								</CardHeader>
								<CardContent>
									<div className="text-2xl font-bold">{stats.activeProducts}</div>
									<p className="text-xs text-muted-foreground">
										{stats.totalProducts} products, {stats.totalEvents} events
									</p>
								</CardContent>
							</Card>

							<Card>
								<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
									<CardTitle className="text-sm font-medium">Featured Items</CardTitle>
									<TrendingUp className="h-4 w-4 text-muted-foreground" />
								</CardHeader>
								<CardContent>
									<div className="text-2xl font-bold">{stats.featuredItems}</div>
									<p className="text-xs text-muted-foreground">
										Displayed on homepage
									</p>
								</CardContent>
							</Card>

							<Card>
								<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
									<CardTitle className="text-sm font-medium">Media Files</CardTitle>
									<FileImage className="h-4 w-4 text-muted-foreground" />
								</CardHeader>
								<CardContent>
									<div className="text-2xl font-bold">{stats.mediaFiles || 'N/A'}</div>
									<p className="text-xs text-muted-foreground">
										Images and videos
									</p>
								</CardContent>
							</Card>

							<Card>
								<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
									<CardTitle className="text-sm font-medium">Quick Actions</CardTitle>
									<Settings className="h-4 w-4 text-muted-foreground" />
								</CardHeader>
								<CardContent>
									<div className="flex gap-1">
										{quickActions.slice(0, 3).map((action, i) => (
											<Button
												key={i}
												size="sm"
												variant="ghost"
												onClick={action.onClick}
												title={action.label}
											>
												<action.icon className="h-4 w-4" />
											</Button>
										))}
									</div>
								</CardContent>
							</Card>
						</>
					) : (
						<div className="col-span-4 text-center text-muted-foreground">
							Failed to load statistics
						</div>
					)}
				</div>

				{/* Quick Actions */}
				<Card className="mb-8">
					<CardHeader>
						<CardTitle>Quick Actions</CardTitle>
						<CardDescription>Common tasks and shortcuts</CardDescription>
					</CardHeader>
					<CardContent>
						<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
							{quickActions.map((action, index) => (
								<Button
									key={index}
									variant="outline"
									className="h-auto flex-col gap-2 p-4"
									onClick={action.onClick}
								>
									<action.icon className="h-5 w-5" />
									<span className="text-sm">{action.label}</span>
								</Button>
							))}
						</div>
					</CardContent>
				</Card>

				{/* Main Sections */}
				<div className="grid md:grid-cols-3 gap-6 mb-8">
					{adminSections.map((section, index) => {
						const Icon = section.icon;
						return (
							<Card
								key={index}
								className="hover:shadow-lg transition-shadow cursor-pointer"
								onClick={() => navigate({ to: section.href as any })}
							>
								<CardHeader>
									<div className="flex items-start justify-between">
										<div className={`p-3 rounded-lg ${section.bgColor}`}>
											<Icon className={`h-6 w-6 ${section.color}`} />
										</div>
										{section.stats && (
											<Badge variant="secondary">{section.stats}</Badge>
										)}
									</div>
									<CardTitle className="mt-4">{section.title}</CardTitle>
									<CardDescription>{section.description}</CardDescription>
								</CardHeader>
								<CardContent>
									<Button variant="ghost" className="w-full">
										Manage
										<ExternalLink className="h-4 w-4 ml-2" />
									</Button>
								</CardContent>
							</Card>
						);
					})}
				</div>

				{/* External Links */}
				<Card>
					<CardHeader>
						<CardTitle>External Services</CardTitle>
						<CardDescription>Access third-party dashboards and tools</CardDescription>
					</CardHeader>
					<CardContent>
						<div className="grid md:grid-cols-3 gap-4">
							{externalLinks.map((link, index) => {
								const Icon = link.icon;
								return (
									<a
										key={index}
										href={link.href}
										target="_blank"
										rel="noopener noreferrer"
										className="flex items-start gap-3 p-4 rounded-lg border hover:bg-accent transition-colors"
									>
										<Icon className={`h-5 w-5 mt-0.5 ${link.color}`} />
										<div className="flex-1">
											<h4 className="font-medium text-sm">{link.title}</h4>
											<p className="text-xs text-muted-foreground">
												{link.description}
											</p>
										</div>
										<ExternalLink className="h-4 w-4 text-muted-foreground" />
									</a>
								);
							})}
						</div>
					</CardContent>
				</Card>

				{/* Footer */}
				<div className="mt-8 text-center text-sm text-muted-foreground">
					<p>Euro Haus Admin Panel v1.0</p>
					<p className="mt-1">
						Need help? Contact{' '}
						<a href="mailto:support@theeurohaus.com" className="underline">
							support@theeurohaus.com
						</a>
					</p>
				</div>
			</div>
		</div>
	);
}
