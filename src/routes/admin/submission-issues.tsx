// euro-haus/src/routes/admin/submission-issues.tsx
import { createFileRoute, useRouter } from '@tanstack/react-router';
import { useState } from 'react';
import { ProtectedRoute } from '~/components/protected-route';
import { SubmissionIssuesManager } from '~/components/admin/submission-issues-manager';
import { submissionService } from '~/lib/services/submission-service';
import { apiClient } from '~/lib/api';
import { toast } from 'sonner';
import type { StripeProduct } from '~/lib/services/stripe-service';

interface ProductsResponse {
	products: StripeProduct[];
}

export const Route = createFileRoute('/admin/submission-issues')({
	loader: async () => {
		try {
			const [submissions, productsResponse] = await Promise.all([
				submissionService.getSubmissionsWithIssues(),
				apiClient.get<ProductsResponse>('/products'),
			]);

			// Get raw StripeProduct data and filter for events
			const events = productsResponse.data.products.filter(
				(p: StripeProduct) => p.metadata?.type === 'event'
			);

			return { submissions, events };
		} catch {
			toast.error('Failed to load submission issues data');
			return { submissions: [], events: [] };
		}
	},
	component: SubmissionIssuesPage,
});

function SubmissionIssuesPage() {
	return (
		<ProtectedRoute>
			<SubmissionIssuesContent />
		</ProtectedRoute>
	);
}

function SubmissionIssuesContent() {
	const { submissions, events } = Route.useLoaderData();
	const router = useRouter();
	const [isLoading, setIsLoading] = useState(false);

	const handleRefresh = async () => {
		setIsLoading(true);
		try {
			await router.invalidate();
			toast.success('Data refreshed');
		} catch {
			toast.error('Failed to refresh data');
		} finally {
			setIsLoading(false);
		}
	};

	return (
		<div className="p-6 space-y-6 min-h-screen">
			<div className="flex justify-between items-center">
				<div>
					<h1 className="text-3xl font-bold">Submission Issues</h1>
					<p className="text-muted-foreground">
						Manage submissions with payment or email issues
					</p>
				</div>
				<button
					onClick={handleRefresh}
					disabled={isLoading}
					className="px-4 py-2 text-sm font-medium text-white bg-primary rounded-md hover:bg-primary/90 disabled:opacity-50"
				>
					{isLoading ? 'Refreshing...' : 'Refresh'}
				</button>
			</div>

			<SubmissionIssuesManager
				submissions={submissions}
				events={events}
			/>
		</div>
	);
}
