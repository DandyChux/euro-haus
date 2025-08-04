// euro-haus/src/components/admin/submission-issues-manager.tsx
import React, { useState, useEffect } from 'react';
import { format } from 'date-fns';
import {
	AlertTriangle,
	Mail,
	DollarSign,
	CheckCircle,
	XCircle,
	ExternalLink,
	Loader2,
	RefreshCw,
	User,
	Calendar,
	Car,
	AlertCircle,
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '~/components/ui/card';
import { Button } from '~/components/ui/button';
import { Badge } from '~/components/ui/badge';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '~/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '~/components/ui/select';
import { Alert, AlertDescription } from '~/components/ui/alert';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '~/components/ui/tabs';
import { toast } from 'sonner';
import { submissionService, type PaymentStatus } from '~/lib/services/submission-service';
import { apiClient } from '~/lib/api';
import type { VehicleSubmission } from '~/lib/interfaces/submission';
import type { StripeProduct } from '~/lib/services/stripe-service';

interface SubmissionWithIssues extends VehicleSubmission {
	issues?: string[];
}

interface EventPrice {
	id: string;
	label: string;
	amount: number;
}

interface SubmissionIssuesManagerProps {
	submissions: SubmissionWithIssues[];
	events: StripeProduct[];
}

export function SubmissionIssuesManager({ submissions, events }: SubmissionIssuesManagerProps) {
	const [selectedSubmission, setSelectedSubmission] = useState<SubmissionWithIssues | null>(null);
	const [paymentStatus, setPaymentStatus] = useState<PaymentStatus | null>(null);
	const [isCheckingPayment, setIsCheckingPayment] = useState(false);
	const [isCreatingPayment, setIsCreatingPayment] = useState(false);
	const [isResendingEmail, setIsResendingEmail] = useState(false);
	const [selectedPriceId, setSelectedPriceId] = useState<string>('');
	const [showPaymentDialog, setShowPaymentDialog] = useState(false);
	const [eventPrices, setEventPrices] = useState<EventPrice[]>([]);
	const [isLoadingPrices, setIsLoadingPrices] = useState(false);

	// Group submissions by issue type
	const paymentIssues = submissions.filter(s =>
		s.issues?.some(i => ['no_payment', 'payment_failed', 'payment_expired', 'payment_incomplete'].includes(i))
	);
	const emailIssues = submissions.filter(s => s.issues?.includes('email_not_sent'));
	const allIssues = submissions;

	// Load event prices when a submission is selected for payment
	useEffect(() => {
		if (selectedSubmission && showPaymentDialog) {
			loadEventPrices(selectedSubmission.eventId);
		}
	}, [selectedSubmission, showPaymentDialog]);

	const loadEventPrices = async (eventId: string) => {
		setIsLoadingPrices(true);
		setEventPrices([]);
		try {
			const response = await apiClient.get<{ prices: any[] }>(`/products/${eventId}/prices`);
			const prices = response.data.prices || [];

			const formattedPrices = prices
				.filter(p => p.active)
				.map(p => ({
					id: p.id,
					label: `${p.nickname || 'Ticket'} - $${(p.unit_amount / 100).toFixed(2)}`,
					amount: p.unit_amount,
				}));

			setEventPrices(formattedPrices);

			// Auto-select first price if only one available
			if (formattedPrices.length === 1) {
				setSelectedPriceId(formattedPrices[0].id);
			}
		} catch (error) {
			console.error('Failed to fetch event prices:', error);
			toast.error('Failed to load ticket prices');
		} finally {
			setIsLoadingPrices(false);
		}
	};

	const handleCheckPaymentStatus = async (submission: SubmissionWithIssues) => {
		setIsCheckingPayment(true);
		try {
			const status = await submissionService.getPaymentStatus(submission.id);
			setPaymentStatus(status);
			setSelectedSubmission(submission);
			toast.success('Payment status retrieved');
		} catch (error) {
			toast.error('Failed to check payment status');
		} finally {
			setIsCheckingPayment(false);
		}
	};

	const handleCreatePayment = async () => {
		if (!selectedSubmission || !selectedPriceId) {
			toast.error('Please select a price');
			return;
		}

		setIsCreatingPayment(true);
		try {
			const event = events.find(e => e.id === selectedSubmission.eventId);
			const eventName = event?.name || 'Event';

			const response = await submissionService.createPayment(
				selectedSubmission.id,
				selectedPriceId,
				eventName
			);

			if (response.sessionUrl) {
				toast.success('Payment link created successfully');
				// Open the payment link in a new tab
				window.open(response.sessionUrl, '_blank');
				setShowPaymentDialog(false);
				setSelectedPriceId('');
			}
		} catch (error) {
			toast.error('Failed to create payment');
		} finally {
			setIsCreatingPayment(false);
		}
	};

	const handleResendEmail = async (submission: SubmissionWithIssues) => {
		setIsResendingEmail(true);
		try {
			const response = await submissionService.resendApprovalEmail(submission.id);
			if (response.success) {
				toast.success(response.message || 'Email sent successfully');
			} else {
				toast.error(response.message || 'Failed to send email');
			}
		} catch (error) {
			toast.error('Failed to resend email');
		} finally {
			setIsResendingEmail(false);
		}
	};

	const SubmissionCard = ({ submission }: { submission: SubmissionWithIssues }) => {
		const event = events.find(e => e.id === submission.eventId);
		const hasPaymentIssue = submission.issues?.some(i =>
			['no_payment', 'payment_failed', 'payment_expired', 'payment_incomplete'].includes(i)
		);
		const hasEmailIssue = submission.issues?.includes('email_not_sent');

		return (
			<Card className="hover:shadow-lg transition-shadow">
				<CardHeader>
					<div className="flex justify-between items-start">
						<div>
							<CardTitle className="text-lg flex items-center gap-2">
								<User className="h-4 w-4" />
								{submission.participantName}
							</CardTitle>
							<CardDescription className="mt-1">
								{submission.participantEmail}
							</CardDescription>
						</div>
						<div className="flex flex-col gap-2">
							{submission.status === 'approved' && (
								<Badge className="bg-green-100 text-green-800">Approved</Badge>
							)}
							{hasPaymentIssue && (
								<Badge variant="destructive" className="flex items-center gap-1">
									<DollarSign className="h-3 w-3" />
									Payment Issue
								</Badge>
							)}
							{hasEmailIssue && (
								<Badge variant="secondary" className="flex items-center gap-1">
									<Mail className="h-3 w-3" />
									Email Issue
								</Badge>
							)}
						</div>
					</div>
				</CardHeader>
				<CardContent className="space-y-4">
					<div className="grid grid-cols-2 gap-4 text-sm">
						<div>
							<p className="text-muted-foreground">Event</p>
							<p className="font-medium">{event?.name || 'Unknown Event'}</p>
						</div>
						<div>
							<p className="text-muted-foreground">Vehicle</p>
							<p className="font-medium">
								{submission.vehicleYear} {submission.vehicleMake} {submission.vehicleModel}
							</p>
						</div>
						<div>
							<p className="text-muted-foreground">Submitted</p>
							<p className="font-medium">
								{format(new Date(submission.submittedAt), 'MMM d, yyyy')}
							</p>
						</div>
						{submission.reviewedAt && (
							<div>
								<p className="text-muted-foreground">Reviewed</p>
								<p className="font-medium">
									{format(new Date(submission.reviewedAt), 'MMM d, yyyy')}
								</p>
							</div>
						)}
					</div>

					{submission.issues && submission.issues.length > 0 && (
						<Alert className="border-yellow-200 bg-yellow-50">
							<AlertTriangle className="h-4 w-4 text-yellow-600" />
							<AlertDescription className="text-sm">
								<strong>Issues detected:</strong>
								<ul className="mt-1 list-disc list-inside">
									{submission.issues.map((issue, idx) => (
										<li key={idx}>{issue.replace(/_/g, ' ')}</li>
									))}
								</ul>
							</AlertDescription>
						</Alert>
					)}

					<div className="flex flex-wrap gap-2">
						{hasPaymentIssue && (
							<>
								<Button
									size="sm"
									variant="outline"
									onClick={() => handleCheckPaymentStatus(submission)}
									disabled={isCheckingPayment}
								>
									{isCheckingPayment ? (
										<Loader2 className="h-4 w-4 animate-spin" />
									) : (
										<RefreshCw className="h-4 w-4" />
									)}
									Check Payment
								</Button>
								<Button
									size="sm"
									onClick={() => {
										setSelectedSubmission(submission);
										setShowPaymentDialog(true);
									}}
								>
									<DollarSign className="h-4 w-4" />
									Create Payment
								</Button>
							</>
						)}
						{hasEmailIssue && submission.status === 'approved' && (
							<Button
								size="sm"
								variant="outline"
								onClick={() => handleResendEmail(submission)}
								disabled={isResendingEmail}
							>
								{isResendingEmail ? (
									<Loader2 className="h-4 w-4 animate-spin" />
								) : (
									<Mail className="h-4 w-4" />
								)}
								Resend Email
							</Button>
						)}
					</div>
				</CardContent>
			</Card>
		);
	};

	return (
		<>
			<Tabs defaultValue="all" className="w-full">
				<TabsList className="grid w-full grid-cols-3">
					<TabsTrigger value="all">
						All Issues ({allIssues.length})
					</TabsTrigger>
					<TabsTrigger value="payment">
						Payment Issues ({paymentIssues.length})
					</TabsTrigger>
					<TabsTrigger value="email">
						Email Issues ({emailIssues.length})
					</TabsTrigger>
				</TabsList>

				<TabsContent value="all" className="space-y-4 mt-6">
					{allIssues.length === 0 ? (
						<Card>
							<CardContent className="text-center py-8">
								<CheckCircle className="w-12 h-12 mx-auto text-green-500 mb-4" />
								<p className="text-muted-foreground">No submissions with issues found</p>
							</CardContent>
						</Card>
					) : (
						<div className="grid gap-4">
							{allIssues.map((submission) => (
								<SubmissionCard key={submission.id} submission={submission} />
							))}
						</div>
					)}
				</TabsContent>

				<TabsContent value="payment" className="space-y-4 mt-6">
					{paymentIssues.length === 0 ? (
						<Card>
							<CardContent className="text-center py-8">
								<CheckCircle className="w-12 h-12 mx-auto text-green-500 mb-4" />
								<p className="text-muted-foreground">No payment issues found</p>
							</CardContent>
						</Card>
					) : (
						<div className="grid gap-4">
							{paymentIssues.map((submission) => (
								<SubmissionCard key={submission.id} submission={submission} />
							))}
						</div>
					)}
				</TabsContent>

				<TabsContent value="email" className="space-y-4 mt-6">
					{emailIssues.length === 0 ? (
						<Card>
							<CardContent className="text-center py-8">
								<CheckCircle className="w-12 h-12 mx-auto text-green-500 mb-4" />
								<p className="text-muted-foreground">No email issues found</p>
							</CardContent>
						</Card>
					) : (
						<div className="grid gap-4">
							{emailIssues.map((submission) => (
								<SubmissionCard key={submission.id} submission={submission} />
							))}
						</div>
					)}
				</TabsContent>
			</Tabs>

			{/* Payment Status Modal */}
			{paymentStatus && selectedSubmission && (
				<Dialog open={!!paymentStatus} onOpenChange={() => setPaymentStatus(null)}>
					<DialogContent className="max-w-md">
						<DialogHeader>
							<DialogTitle>Payment Status</DialogTitle>
							<DialogDescription>
								Payment details for {selectedSubmission.participantName}
							</DialogDescription>
						</DialogHeader>
						<div className="space-y-4">
							<div className="flex items-center justify-between">
								<span className="text-sm text-muted-foreground">Has Payment:</span>
								<Badge variant={paymentStatus.hasPayment ? 'default' : 'destructive'}>
									{paymentStatus.hasPayment ? 'Yes' : 'No'}
								</Badge>
							</div>
							{paymentStatus.paymentStatus && (
								<div className="flex items-center justify-between">
									<span className="text-sm text-muted-foreground">Status:</span>
									<span className="font-medium">{paymentStatus.paymentStatus}</span>
								</div>
							)}
							{paymentStatus.paymentAmount && (
								<div className="flex items-center justify-between">
									<span className="text-sm text-muted-foreground">Amount:</span>
									<span className="font-medium">
										${(paymentStatus.paymentAmount / 100).toFixed(2)} {paymentStatus.paymentCurrency?.toUpperCase()}
									</span>
								</div>
							)}
							{paymentStatus.emailSent !== undefined && (
								<div className="flex items-center justify-between">
									<span className="text-sm text-muted-foreground">Email Sent:</span>
									<Badge variant={paymentStatus.emailSent ? 'default' : 'secondary'}>
										{paymentStatus.emailSent ? 'Yes' : 'No'}
									</Badge>
								</div>
							)}
							{paymentStatus.emailSentAt && (
								<div className="flex items-center justify-between">
									<span className="text-sm text-muted-foreground">Email Sent At:</span>
									<span className="font-medium">
										{format(new Date(paymentStatus.emailSentAt), 'MMM d, yyyy h:mm a')}
									</span>
								</div>
							)}
							{paymentStatus.errorMessage && (
								<Alert className="border-red-200 bg-red-50">
									<AlertCircle className="h-4 w-4 text-red-600" />
									<AlertDescription className="text-sm text-red-800">
										{paymentStatus.errorMessage}
									</AlertDescription>
								</Alert>
							)}
							{paymentStatus.checkoutURL && (
								<Button
									className="w-full"
									onClick={() => window.open(paymentStatus.checkoutURL, '_blank')}
								>
									<ExternalLink className="h-4 w-4 mr-2" />
									Open Checkout
								</Button>
							)}
						</div>
					</DialogContent>
				</Dialog>
			)}

			{/* Create Payment Dialog */}
			<Dialog open={showPaymentDialog} onOpenChange={(open) => {
				setShowPaymentDialog(open);
				if (!open) {
					setSelectedPriceId('');
					setEventPrices([]);
				}
			}}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Create Payment Link</DialogTitle>
						<DialogDescription>
							Generate a payment link for {selectedSubmission?.participantName}
						</DialogDescription>
					</DialogHeader>
					<div className="space-y-4">
						{isLoadingPrices ? (
							<div className="flex items-center justify-center py-4">
								<Loader2 className="h-6 w-6 animate-spin" />
							</div>
						) : (
							<div>
								<label className="text-sm font-medium">Select Ticket Type</label>
								<Select value={selectedPriceId} onValueChange={setSelectedPriceId}>
									<SelectTrigger className="w-full mt-1">
										<SelectValue placeholder="Choose a ticket type" />
									</SelectTrigger>
									<SelectContent>
										{eventPrices.map((price) => (
											<SelectItem key={price.id} value={price.id}>
												{price.label}
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>
						)}
					</div>
					<DialogFooter>
						<Button variant="outline" onClick={() => setShowPaymentDialog(false)}>
							Cancel
						</Button>
						<Button
							onClick={handleCreatePayment}
							disabled={!selectedPriceId || isCreatingPayment || isLoadingPrices}
						>
							{isCreatingPayment ? (
								<>
									<Loader2 className="h-4 w-4 animate-spin mr-2" />
									Creating...
								</>
							) : (
								'Create Payment Link'
							)}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	);
}
