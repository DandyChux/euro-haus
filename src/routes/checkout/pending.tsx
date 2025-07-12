import { createFileRoute } from '@tanstack/react-router';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '~/components/ui/card';
import { Clock, Car, Calendar, User, Mail } from 'lucide-react';
import { Button } from '~/components/ui/button';
import { Link } from '@tanstack/react-router';
import { Separator } from '~/components/ui/separator';
import { Badge } from '~/components/ui/badge';
import z from 'zod';
import { submissionService } from '~/lib/services/submission-service';
import { format } from 'date-fns';

export const Route = createFileRoute('/checkout/pending')({
	component: PendingCheckoutPage,
	validateSearch: z.object({
		submission_id: z.string()
	}),
	loaderDeps: ({ search: { submission_id } }) => ({
		submissionId: submission_id,
	}),
	loader: async ({ deps: { submissionId } }) => {
		try {
			const submission = await submissionService.getSubmission(submissionId);
			return { submission };
		} catch (error) {
			console.error('Failed to load submission:', error);
			return { submission: null };
		}
	},
});

function PendingCheckoutPage() {
	const { submission } = Route.useLoaderData();

	return (
		<div className="min-h-screen flex items-center justify-center p-4 bg-gray-50">
			<Card className="max-w-2xl w-full">
				<CardHeader className="text-center">
					<div className="mx-auto mb-4 h-16 w-16 text-orange-500 bg-orange-50 rounded-full flex items-center justify-center">
						<Clock className="h-8 w-8" />
					</div>
					<CardTitle className="text-2xl">Submission Received!</CardTitle>
					<CardDescription>
						Your vehicle submission is pending approval
					</CardDescription>
				</CardHeader>
				<CardContent className="space-y-6">
					{submission ? (
						<>
							{/* Submission Details */}
							<div className="bg-gray-50 rounded-lg p-4 space-y-3">
								<div className="flex items-center gap-2 text-sm font-medium">
									<Car className="h-4 w-4 text-muted-foreground" />
									Vehicle Details
								</div>
								<div className="pl-6 space-y-1">
									<p className="text-lg font-semibold">
										{submission.vehicleYear} {submission.vehicleMake} {submission.vehicleModel}
									</p>
									{submission.vehicleDescription && (
										<p className="text-sm text-muted-foreground">{submission.vehicleDescription}</p>
									)}
								</div>
							</div>

							<Separator />

							{/* Participant Information */}
							<div className="space-y-3">
								<h3 className="text-sm font-medium">Your Information</h3>
								<div className="grid gap-3 text-sm">
									<div className="flex items-center gap-2">
										<User className="h-4 w-4 text-muted-foreground" />
										<span>{submission.participantName}</span>
									</div>
									<div className="flex items-center gap-2">
										<Mail className="h-4 w-4 text-muted-foreground" />
										<span>{submission.participantEmail}</span>
									</div>
									<div className="flex items-center gap-2">
										<Calendar className="h-4 w-4 text-muted-foreground" />
										<span>Submitted on {format(new Date(submission.submittedAt), 'MMMM d, yyyy')}</span>
									</div>
								</div>
							</div>

							<Separator />

							{/* Reference Number */}
							<div className="bg-blue-50 rounded-lg p-4">
								<p className="text-sm font-medium text-blue-900 mb-1">Reference Number</p>
								<p className="font-mono text-lg">{submission.id}</p>
								<p className="text-xs text-blue-700 mt-2">
									Keep this number for your records. You'll need it if you contact support.
								</p>
							</div>

							{/* What's Next */}
							<div className="space-y-3">
								<h3 className="font-medium">What happens next?</h3>
								<ol className="space-y-2 text-sm text-muted-foreground">
									<li className="flex gap-2">
										<span className="font-medium">1.</span>
										<span>Our team will review your vehicle submission within 24-48 hours</span>
									</li>
									<li className="flex gap-2">
										<span className="font-medium">2.</span>
										<span>If approved, your payment will be processed automatically</span>
									</li>
									<li className="flex gap-2">
										<span className="font-medium">3.</span>
										<span>You'll receive your event ticket and participant details via email</span>
									</li>
									<li className="flex gap-2">
										<span className="font-medium">4.</span>
										<span>If not approved, you'll receive an email with the reason and no charges will be made</span>
									</li>
								</ol>
							</div>

							{/* Status Badge */}
							<div className="flex justify-center">
								<Badge variant="secondary" className="text-orange-600 bg-orange-50">
									<Clock className="h-3 w-3 mr-1" />
									Awaiting Review
								</Badge>
							</div>
						</>
					) : (
						<div className="text-center py-4">
							<p className="text-sm text-muted-foreground">
								Unable to load submission details. Your submission has been received and is being processed.
							</p>
						</div>
					)}

					<div className="pt-4 space-y-3">
						<p className="text-sm text-muted-foreground text-center">
							You'll receive an email notification once your submission has been reviewed.
							Please allow 24-48 hours for processing.
						</p>

						<div className="flex gap-3">
							<Button asChild variant="outline" className="flex-1">
								<Link to="/events">View More Events</Link>
							</Button>
							<Button asChild className="flex-1">
								<Link to="/">Return to Home</Link>
							</Button>
						</div>
					</div>
				</CardContent>
			</Card>
		</div>
	);
}
