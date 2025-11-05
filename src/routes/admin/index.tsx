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
	DollarSign,
	FileImage,
	Settings,
	ExternalLink,
	BarChart,
	ShoppingBag,
	Car,
	Tag,
	AlertTriangle,
	TicketIcon
} from 'lucide-react';
import { ProtectedRoute } from '~/components/protected-route';
import { useAuth } from '~/lib/contexts/auth-context';
import { galleryService } from '~/lib/services/gallery-service';
import { stripeService } from '~/lib/services/stripe-service';

interface DashboardStats {
	totalProducts: number;
	totalEvents: number;
	activeProducts: number;
	featuredItems: number;
	mediaFiles?: number;
	pendingSubmissions?: number;
	activeCoupons?: number;
}

export const Route = createFileRoute('/admin/')({
	loader: async () => {

		const [products, mediaResponse] = await Promise.all([
			stripeService.getAllProducts(),
			galleryService.getAllMedia(),
		]);

		console.log('Products:', products);
		console.log('Media:', mediaResponse);

		// Calculate stats
		const stats: DashboardStats = {
			totalProducts: products.length,
			totalEvents: products.filter((p) => Object.keys(p).includes('slug')).length,
			activeProducts: products.filter((p) => p.inStock).length,
			featuredItems: products.filter((p) => p.featured).length,
			mediaFiles: mediaResponse.files?.length || 0,
		};

		return stats;

	},
	component: AdminDashboardPage,
	pendingComponent: () => (
		<div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
			{[...Array(4)].map((_, i) => (
				<Skeleton key={i} className="h-24" />
			))}
		</div>
	),
	errorComponent: () => (
		<div className="col-span-4 text-center text-muted-foreground">
			Failed to load statistics
		</div>
	)
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
	const stats = Route.useLoaderData();
	const { logout } = useAuth();

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
			description: 'Manage event tickets, schedules, and attendee information. Check in attendees and manage event tickets',
			icon: Calendar,
			href: '/admin/events',
			color: 'text-purple-600',
			bgColor: 'bg-purple-50',
			stats: stats ? `${stats.totalEvents} events` : null,
		},
		{
			title: 'Discount Management',
			description: 'Create and manage coupons and promotion codes',
			icon: Tag,
			href: '/admin/coupons',
			color: 'text-green-600',
			bgColor: 'bg-green-50',
			stats: stats?.activeCoupons ? `${stats.activeCoupons} active` : null,
		},
		{
			title: 'Vehicle Submissions',
			description: 'Review and approve participant vehicle submissions',
			icon: Car,
			href: '/admin/submissions',
			color: 'text-orange-600',
			bgColor: 'bg-orange-50',
			stats: stats?.pendingSubmissions ? `${stats.pendingSubmissions} pending` : null,
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
		{
			title: 'Submission Issues',
			description: 'Fix payment and email issues for approved submissions',
			icon: AlertTriangle,
			href: '/admin/submission-issues',
			color: 'text-red-600',
			bgColor: 'bg-red-50',
			stats: null, // You could fetch count of issues if needed
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
			label: 'Review Submissions',
			icon: Car,
			onClick: () => navigate({ to: '/admin/submissions' }),
		},
		{
			label: 'Upload Media',
			icon: FileImage,
			onClick: () => navigate({ to: '/admin/media', search: { tab: 'upload' } }),
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
					<Card>
						<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
							<CardTitle className="text-sm font-medium">Active Products</CardTitle>
							<ShoppingBag className="h-4 w-4 text-muted-foreground" />
						</CardHeader>
						<CardContent>
							<div className="text-2xl font-bold">{stats?.activeProducts}</div>
							<p className="text-xs text-muted-foreground">
								{stats?.totalProducts} products, {stats?.totalEvents} events
							</p>
						</CardContent>
					</Card>

					<Card>
						<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
							<CardTitle className="text-sm font-medium">Featured Items</CardTitle>
							<TrendingUp className="h-4 w-4 text-muted-foreground" />
						</CardHeader>
						<CardContent>
							<div className="text-2xl font-bold">{stats?.featuredItems}</div>
							<p className="text-xs text-muted-foreground">
								Displayed on homepage
							</p>
						</CardContent>
					</Card>

					<Card>
						<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
							<CardTitle className="text-sm font-medium">Active Discounts</CardTitle>
							<Tag className="h-4 w-4 text-muted-foreground" />
						</CardHeader>
						<CardContent>
							<div className="text-2xl font-bold">{stats?.activeCoupons || 0}</div>
							<p className="text-xs text-muted-foreground">
								Coupons available
							</p>
						</CardContent>
					</Card>

					<Card>
						<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
							<CardTitle className="text-sm font-medium">Pending Submissions</CardTitle>
							<Car className="h-4 w-4 text-muted-foreground" />
						</CardHeader>
						<CardContent>
							<div className="text-2xl font-bold">{stats?.pendingSubmissions || 0}</div>
							<p className="text-xs text-muted-foreground">
								Vehicle submissions
							</p>
						</CardContent>
					</Card>

					<Card>
						<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
							<CardTitle className="text-sm font-medium">Media Files</CardTitle>
							<FileImage className="h-4 w-4 text-muted-foreground" />
						</CardHeader>
						<CardContent>
							<div className="text-2xl font-bold">{stats?.mediaFiles || 'N/A'}</div>
							<p className="text-xs text-muted-foreground">
								Images and videos
							</p>
						</CardContent>
					</Card>
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
				<div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
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
						<a href="mailto:info@theeurohaus.com" className="underline">
							info@theeurohaus.com
						</a>
					</p>
				</div>
			</div>
		</div>
	);
}
