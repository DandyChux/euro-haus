// euro-haus/src/components/admin/submission-issues-manager.tsx
import React, { useState, useEffect } from 'react';
import { format } from 'date-fns';
import {
	AlertTriangle,
	Mail,
	DollarSign,
	CheckCircle,
	ExternalLink,
	Loader2,
	RefreshCw,
	User,
	AlertCircle,
	Filter,
	Search,
	X,
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from '~/components/ui/card';
import { Button } from '~/components/ui/button';
import { Badge } from '~/components/ui/badge';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '~/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '~/components/ui/select';
import { Alert, AlertDescription } from '~/components/ui/alert';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '~/components/ui/tabs';
import { toast } from 'sonner';
import { submissionService, type PaymentStatus } from '~/lib/services/submission-service';
import { apiClient } from '~/lib/api';
import { Input } from '~/components/ui/input';
import { Checkbox } from '~/components/ui/checkbox';
import { Switch } from '~/components/ui/switch';
import { Label } from '~/components/ui/label';
import type { VehicleSubmission } from '~/lib/interfaces/submission';
import type { StripeProduct } from '~/lib/services/stripe-service';
import { getRouteApi, useNavigate } from '@tanstack/react-router';

interface SubmissionWithIssues extends VehicleSubmission {
	issues?: string[];
}

interface EventPrice {
	id: string;
	label: string;
	amount: number;
}

interface PriceResponse {
	id: string;
	nickname?: string;
	unit_amount: number;
	active: boolean;
}

interface SubmissionIssuesManagerProps {
	submissions: SubmissionWithIssues[];
	events: StripeProduct[];
}

interface FilterOptions {
	debug: boolean;
	all: boolean;
	includeId: string;
	issueType: string[];
	status: string[];
	searchTerm: string;
}

export function SubmissionIssuesManager({ submissions, events }: SubmissionIssuesManagerProps) {
	const routeApi = getRouteApi('/admin/submission-issues');
	const { debug, all, include_id } = routeApi.useSearch();
	const navigate = useNavigate();
	const [selectedSubmission, setSelectedSubmission] = useState<SubmissionWithIssues | null>(null);
	const [paymentStatus, setPaymentStatus] = useState<PaymentStatus | null>(null);
	const [isCheckingPayment, setIsCheckingPayment] = useState(false);
	const [isCreatingPayment, setIsCreatingPayment] = useState(false);
	const [isResendingEmail, setIsResendingEmail] = useState(false);
	const [selectedPriceId, setSelectedPriceId] = useState<string>('');
	const [showPaymentDialog, setShowPaymentDialog] = useState(false);
	const [eventPrices, setEventPrices] = useState<EventPrice[]>([]);
	const [isLoadingPrices, setIsLoadingPrices] = useState(false);
	const [showFilterDialog, setShowFilterDialog] = useState(false);

	// Filter states
	const [filterOptions, setFilterOptions] = useState<FilterOptions>({
		debug: debug === true,
		all: all === true,
		includeId: include_id || '',
		issueType: [],
		status: [],
		searchTerm: '',
	});

	// Apply filters from URL on component mount
	useEffect(() => {
		setFilterOptions(prev => ({
			...prev,
			debug: debug === true,
			all: all === true,
			includeId: include_id || '',
		}));
	}, [debug, all, include_id]);

	// Filtered submissions
	const filteredSubmissions = submissions.filter(submission => {
		// Apply search filter
		if (filterOptions.searchTerm) {
			const searchLower = filterOptions.searchTerm.toLowerCase();
			const matchesSearch =
				submission.participantName?.toLowerCase().includes(searchLower) ||
				submission.participantEmail?.toLowerCase().includes(searchLower) ||
				submission.vehicleMake?.toLowerCase().includes(searchLower) ||
				submission.vehicleModel?.toLowerCase().includes(searchLower) ||
				submission.id?.toLowerCase().includes(searchLower);

			if (!matchesSearch) return false;
		}

		// Apply status filter
		if (filterOptions.status.length > 0 && !filterOptions.status.includes(submission.status)) {
			return false;
		}

		// Apply issue type filter
		if (filterOptions.issueType.length > 0) {
			const hasMatchingIssue = submission.issues?.some(issue =>
				filterOptions.issueType.includes(issue)
			);
			if (!hasMatchingIssue) return false;
		}

		return true;
	});

	// Group submissions by issue type
	const paymentIssues = filteredSubmissions.filter(s =>
		s.issues?.some(i => ['no_payment', 'payment_failed', 'payment_expired', 'payment_incomplete',
			'missing_payment_intent', 'payment_intent_check_failed', 'payment_not_succeeded',
			'missing_checkout_data', 'incomplete_payment_process'].includes(i))
	);

	const emailIssues = filteredSubmissions.filter(s =>
		s.issues?.includes('email_not_sent')
	);

	const ticketIssues = filteredSubmissions.filter(s =>
		s.issues?.includes('no_ticket_created')
	);

	const allIssues = filteredSubmissions;

	// Load event prices when a submission is selected for payment
	useEffect(() => {
		if (selectedSubmission && showPaymentDialog) {
			loadEventPrices(selectedSubmission.eventId);
		}
	}, [selectedSubmission, showPaymentDialog]);

	// Apply filters via URL parameters
	const applyFilters = () => {
		const params: Record<string, string> = {};

		if (filterOptions.debug) params.debug = 'true';
		if (filterOptions.all) params.all = 'true';
		if (filterOptions.includeId) params.include_id = filterOptions.includeId;

		navigate({
			to: '/admin/submission-issues',
			search: {
				...params
			}
		})
		setShowFilterDialog(false);

		// Reload data with new filters
		refreshWithFilters();
	};

	const refreshWithFilters = async () => {
		try {
			// Build the query string for the API request
			const queryParams = new URLSearchParams();
			if (filterOptions.debug) queryParams.set('debug', 'true');
			if (filterOptions.all) queryParams.set('all', 'true');
			if (filterOptions.includeId) queryParams.set('include_id', filterOptions.includeId);

			// Make the API call with the query parameters
			const response = await apiClient.get(`/admin/submissions/issues?${queryParams.toString()}`);

			// Update the submissions data
			// Note: In a real application, you would need to update the submissions in your state management
			if (response.data && response.data.submissions) {
				toast.success(`Found ${response.data.submissions.length} submissions with issues`);
			}

			// Force a full reload to get fresh data with the new filters
			window.location.reload();
		} catch (error) {
			toast.error('Failed to refresh with filters');
		}
	};

	const loadEventPrices = async (eventId: string) => {
		setIsLoadingPrices(true);
		setEventPrices([]);
		try {
			const response = await apiClient.get<{ prices: PriceResponse[] }>(`/products/${eventId}/prices`);
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
		} catch {
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
		} catch {
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
		} catch {
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
		} catch {
			toast.error('Failed to resend email');
		} finally {
			setIsResendingEmail(false);
		}
	};

	const resetFilters = () => {
		setFilterOptions({
			debug: false,
			all: false,
			includeId: '',
			issueType: [],
			status: [],
			searchTerm: '',
		});
	};

	// Get a list of all unique issue types
	const allIssueTypes = [...new Set(
		submissions.flatMap(submission => submission.issues || [])
	)].sort();

	const SubmissionCard = ({ submission }: { submission: SubmissionWithIssues }) => {
		const event = events.find(e => e.id === submission.eventId);
		const hasPaymentIssue = submission.issues?.some(i =>
			['no_payment', 'payment_failed', 'payment_expired', 'payment_incomplete',
				'missing_payment_intent', 'payment_intent_check_failed', 'payment_not_succeeded',
				'missing_checkout_data', 'incomplete_payment_process'].includes(i)
		);
		const hasEmailIssue = submission.issues?.includes('email_not_sent');
		const hasTicketIssue = submission.issues?.includes('no_ticket_created');

		return (
			<Card className="hover:shadow-lg transition-shadow">
				<CardHeader>
					<div className="flex justify-between items-start">
						<div>
							<CardTitle className="text-lg flex items-center gap-2">
								<User className="h-4 w-4" />
								{submission.participantName}
							</CardTitle>
							<CardDescription className="mt-1 flex items-center gap-1">
								{submission.participantEmail}
								<Badge variant="outline" className="ml-2">{submission.id}</Badge>
							</CardDescription>
						</div>
						<div className="flex flex-col gap-2">
							<Badge className={
								submission.status === 'approved' ? 'bg-green-100 text-green-800' :
									submission.status === 'pending' ? 'bg-yellow-100 text-yellow-800' :
										submission.status === 'denied' ? 'bg-red-100 text-red-800' :
											'bg-gray-100 text-gray-800'
							}>
								{submission.status}
							</Badge>
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
							{hasTicketIssue && (
								<Badge variant="outline" className="flex items-center gap-1 border-orange-300 text-orange-700">
									<AlertCircle className="h-3 w-3" />
									Ticket Issue
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
				<CardFooter className="text-xs text-muted-foreground pt-0">
					{submission.checkoutSessionId && <span className="mr-2">Session: {submission.checkoutSessionId.substring(0, 8)}...</span>}
					{submission.paymentIntentId && <span className="mr-2">Payment: {submission.paymentIntentId.substring(0, 8)}...</span>}
					{submission.ticketId && <span>Ticket: {submission.ticketId.substring(0, 8)}...</span>}
				</CardFooter>
			</Card>
		);
	};

	return (
		<>
			<div className="flex justify-between items-center mb-6">
				<div className="relative w-full max-w-sm">
					<Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
					<Input
						placeholder="Search by name, email, vehicle..."
						className="pl-9 pr-4"
						value={filterOptions.searchTerm}
						onChange={(e) => setFilterOptions({ ...filterOptions, searchTerm: e.target.value })}
					/>
				</div>
				<div className="flex gap-2">
					<Button variant="outline" onClick={() => setShowFilterDialog(true)}>
						<Filter className="h-4 w-4 mr-2" />
						Filter
						{(filterOptions.debug || filterOptions.all || filterOptions.includeId ||
							filterOptions.status.length > 0 || filterOptions.issueType.length > 0) && (
								<Badge className="ml-2 h-5 w-5 p-0 flex items-center justify-center">
									{[filterOptions.debug, filterOptions.all, filterOptions.includeId !== ''].filter(Boolean).length +
										filterOptions.status.length + filterOptions.issueType.length}
								</Badge>
							)}
					</Button>
				</div>
			</div>

			<Tabs defaultValue="all" className="w-full">
				<TabsList className="grid w-full grid-cols-4">
					<TabsTrigger value="all">
						All Issues ({allIssues.length})
					</TabsTrigger>
					<TabsTrigger value="payment">
						Payment Issues ({paymentIssues.length})
					</TabsTrigger>
					<TabsTrigger value="email">
						Email Issues ({emailIssues.length})
					</TabsTrigger>
					<TabsTrigger value="ticket">
						Ticket Issues ({ticketIssues.length})
					</TabsTrigger>
				</TabsList>

				<TabsContent value="all" className="space-y-4 mt-6">
					{allIssues.length === 0 ? (
						<Card>
							<CardContent className="text-center py-8">
								<CheckCircle className="w-12 h-12 mx-auto text-green-500 mb-4" />
								<p className="text-muted-foreground">No submissions with issues found</p>
								{(filterOptions.debug || filterOptions.all || filterOptions.includeId ||
									filterOptions.status.length > 0 || filterOptions.issueType.length > 0 ||
									filterOptions.searchTerm) && (
										<Button
											variant="link"
											onClick={resetFilters}
											className="mt-2"
										>
											Clear filters
										</Button>
									)}
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

				<TabsContent value="ticket" className="space-y-4 mt-6">
					{ticketIssues.length === 0 ? (
						<Card>
							<CardContent className="text-center py-8">
								<CheckCircle className="w-12 h-12 mx-auto text-green-500 mb-4" />
								<p className="text-muted-foreground">No ticket issues found</p>
							</CardContent>
						</Card>
					) : (
						<div className="grid gap-4">
							{ticketIssues.map((submission) => (
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

			{/* Filter Dialog */}
			<Dialog open={showFilterDialog} onOpenChange={setShowFilterDialog}>
				<DialogContent className="max-w-md">
					<DialogHeader>
						<DialogTitle>Filter Submissions</DialogTitle>
						<DialogDescription>
							Customize which submissions are displayed
						</DialogDescription>
					</DialogHeader>
					<div className="space-y-6 py-2">
						<div className="space-y-4">
							<h3 className="text-sm font-medium">Debug Options</h3>

							<div className="flex items-center justify-between">
								<Label htmlFor="debug-mode" className="flex items-center gap-2">
									Debug Mode
									<span className="text-xs text-muted-foreground">(Show all submissions)</span>
								</Label>
								<Switch
									id="debug-mode"
									checked={filterOptions.debug}
									onCheckedChange={(checked) => setFilterOptions({ ...filterOptions, debug: checked })}
								/>
							</div>

							<div className="flex items-center justify-between">
								<Label htmlFor="all-mode" className="flex items-center gap-2">
									Show All
									<span className="text-xs text-muted-foreground">(Including without issues)</span>
								</Label>
								<Switch
									id="all-mode"
									checked={filterOptions.all}
									onCheckedChange={(checked) => setFilterOptions({ ...filterOptions, all: checked })}
								/>
							</div>

							<div className="flex flex-col gap-2">
								<Label htmlFor="include-id">Include Specific Submission ID</Label>
								<Input
									id="include-id"
									placeholder="Enter submission ID"
									value={filterOptions.includeId}
									onChange={(e) => setFilterOptions({ ...filterOptions, includeId: e.target.value })}
								/>
							</div>
						</div>

						<div className="space-y-4">
							<h3 className="text-sm font-medium">Status Filters</h3>
							<div className="grid grid-cols-2 gap-2">
								{['approved', 'pending', 'denied'].map(status => (
									<div key={status} className="flex items-center gap-2">
										<Checkbox
											id={`status-${status}`}
											checked={filterOptions.status.includes(status)}
											onCheckedChange={(checked) => {
												const newStatus = checked
													? [...filterOptions.status, status]
													: filterOptions.status.filter(s => s !== status);
												setFilterOptions({ ...filterOptions, status: newStatus });
											}}
										/>
										<Label htmlFor={`status-${status}`}>{status}</Label>
									</div>
								))}
							</div>
						</div>

						<div className="space-y-4">
							<h3 className="text-sm font-medium">Issue Type Filters</h3>
							<div className="grid grid-cols-1 gap-2 max-h-60 overflow-y-auto">
								{allIssueTypes.map(issue => (
									<div key={issue} className="flex items-center gap-2">
										<Checkbox
											id={`issue-${issue}`}
											checked={filterOptions.issueType.includes(issue)}
											onCheckedChange={(checked) => {
												const newIssues = checked
													? [...filterOptions.issueType, issue]
													: filterOptions.issueType.filter(i => i !== issue);
												setFilterOptions({ ...filterOptions, issueType: newIssues });
											}}
										/>
										<Label htmlFor={`issue-${issue}`}>{issue.replace(/_/g, ' ')}</Label>
									</div>
								))}
							</div>
						</div>
					</div>
					<DialogFooter className="flex justify-between items-center">
						<Button
							variant="ghost"
							onClick={resetFilters}
							className="mr-auto"
						>
							Reset
						</Button>
						<div className="flex gap-2">
							<Button variant="outline" onClick={() => setShowFilterDialog(false)}>
								Cancel
							</Button>
							<Button onClick={applyFilters}>
								Apply Filters
							</Button>
						</div>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	);
}
