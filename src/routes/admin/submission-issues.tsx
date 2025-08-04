import { createFileRoute } from '@tanstack/react-router';
import { useState } from 'react';
import { ProtectedRoute } from '~/components/protected-route';
import { SubmissionIssuesManager } from '~/components/admin/submission-issues-manager';
import { submissionIssuesService } from '~/lib/services/submission-issues-service';
import { stripeService } from '~/lib/services/stripe-service';
import { toast } from 'sonner';

export const Route = createFileRoute('/admin/submission-issues')({
	loader: async () => {
		try {
			const [submissions, events] = await Promise.all([
				submissionIssuesService.getSubmissionsWithIssues(),
				stripeService.getAllEvents(),
			]);

			return { submissions, events };
		} catch (error) {
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
	const [isLoading, setIsLoading] = useState(false);

	const handleRefresh = async () => {
		setIsLoading(true);
		try {
			await Route.router.invalidate();
			toast.success('Data refreshed');
		} catch (error) {
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
