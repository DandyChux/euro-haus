import React, { useState, useEffect } from 'react';
import { format } from 'date-fns';
import {
	Car,
	CheckCircle,
	XCircle,
	Clock,
	Mail,
	Phone,
	Calendar,
	User,
	Image as ImageIcon,
	ChevronLeft,
	ChevronRight,
	Loader2,
	AlertCircle,
} from 'lucide-react';
import { Button } from '~/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '~/components/ui/card';
import { Badge } from '~/components/ui/badge';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '~/components/ui/dialog';
import { Textarea } from '~/components/ui/textarea';
import { Label } from '~/components/ui/label';
import { Alert, AlertDescription } from '~/components/ui/alert';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '~/components/ui/tabs';
import { submissionService } from '~/lib/services/submission-service';
import type { VehicleSubmission } from '~/lib/interfaces/submission';

interface SubmissionReviewProps {
	eventId: string;
	eventName: string;
}

export function SubmissionReview({ eventId, eventName }: SubmissionReviewProps) {
	const [submissions, setSubmissions] = useState<VehicleSubmission[]>([]);
	const [selectedSubmission, setSelectedSubmission] = useState<VehicleSubmission | null>(null);
	const [currentImageIndex, setCurrentImageIndex] = useState(0);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [actionDialog, setActionDialog] = useState<{
		open: boolean;
		type: 'approve' | 'deny' | null;
		submission: VehicleSubmission | null;
	}>({ open: false, type: null, submission: null });
	const [actionNotes, setActionNotes] = useState('');
	const [processing, setProcessing] = useState(false);

	useEffect(() => {
		loadSubmissions();
	}, [eventId]);

	const loadSubmissions = async () => {
		try {
			setLoading(true);
			const data = await submissionService.getEventSubmissions(eventId);
			setSubmissions(data);
			setError(null);
		} catch (err) {
			setError('Failed to load submissions');
			console.error(err);
		} finally {
			setLoading(false);
		}
	};

	const handleAction = async () => {
		if (!actionDialog.submission || !actionDialog.type) return;

		setProcessing(true);
		try {
			if (actionDialog.type === 'approve') {
				await submissionService.approveSubmission(actionDialog.submission.id, actionNotes);
			} else {
				await submissionService.denySubmission(actionDialog.submission.id, actionNotes);
			}

			// Reload submissions
			await loadSubmissions();

			// Close dialog and reset
			setActionDialog({ open: false, type: null, submission: null });
			setActionNotes('');
			setSelectedSubmission(null);
		} catch (err) {
			setError(`Failed to ${actionDialog.type} submission`);
		} finally {
			setProcessing(false);
		}
	};

	const getStatusBadge = (status: VehicleSubmission['status']) => {
		switch (status) {
			case 'pending':
				return <Badge variant="secondary"><Clock className="w-3 h-3 mr-1" />Pending Review</Badge>;
			case 'approved':
				return <Badge variant="default" className="bg-green-500"><CheckCircle className="w-3 h-3 mr-1" />Approved</Badge>;
			case 'denied':
				return <Badge variant="destructive"><XCircle className="w-3 h-3 mr-1" />Denied</Badge>;
		}
	};

	const pendingSubmissions = submissions.filter(s => s.status === 'pending');
	const reviewedSubmissions = submissions.filter(s => s.status !== 'pending');

	if (loading) {
		return (
			<div className="flex items-center justify-center h-64">
				<Loader2 className="w-8 h-8 animate-spin" />
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<div>
				<h2 className="text-2xl font-bold">Vehicle Submissions</h2>
				<p className="text-muted-foreground">Review and approve participant vehicles for {eventName}</p>
			</div>

			{error && (
				<Alert variant="destructive">
					<AlertCircle className="h-4 w-4" />
					<AlertDescription>{error}</AlertDescription>
				</Alert>
			)}

			<Tabs defaultValue="pending" className="w-full">
				<TabsList className="grid w-full grid-cols-2">
					<TabsTrigger value="pending">
						Pending Review ({pendingSubmissions.length})
					</TabsTrigger>
					<TabsTrigger value="reviewed">
						Reviewed ({reviewedSubmissions.length})
					</TabsTrigger>
				</TabsList>

				<TabsContent value="pending" className="space-y-4">
					{pendingSubmissions.length === 0 ? (
						<Card>
							<CardContent className="text-center py-8">
								<Car className="w-12 h-12 mx-auto text-muted-foreground mb-4" />
								<p className="text-muted-foreground">No pending submissions</p>
							</CardContent>
						</Card>
					) : (
						<div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
							{pendingSubmissions.map((submission) => (
								<Card
									key={submission.id}
									className="cursor-pointer hover:shadow-lg transition-shadow"
									onClick={() => setSelectedSubmission(submission)}
								>
									<CardHeader>
										<div className="flex justify-between items-start">
											<div>
												<CardTitle className="text-lg">
													{submission.vehicleYear} {submission.vehicleMake} {submission.vehicleModel}
												</CardTitle>
												<CardDescription>{submission.participantName}</CardDescription>
											</div>
											{getStatusBadge(submission.status)}
										</div>
									</CardHeader>
									<CardContent>
										<div className="aspect-video relative overflow-hidden rounded-lg bg-gray-100">
											{submission.images[0] ? (
												<img
													src={submission.images[0]}
													alt={`${submission.vehicleMake} ${submission.vehicleModel}`}
													className="object-cover w-full h-full"
												/>
											) : (
												<div className="flex items-center justify-center h-full">
													<ImageIcon className="w-8 h-8 text-gray-400" />
												</div>
											)}
										</div>
										<div className="mt-4 flex items-center text-sm text-muted-foreground">
											<Calendar className="w-4 h-4 mr-1" />
											{format(new Date(submission.submittedAt), 'MMM dd, yyyy')}
										</div>
									</CardContent>
								</Card>
							))}
						</div>
					)}
				</TabsContent>

				<TabsContent value="reviewed" className="space-y-4">
					{reviewedSubmissions.length === 0 ? (
						<Card>
							<CardContent className="text-center py-8">
								<CheckCircle className="w-12 h-12 mx-auto text-muted-foreground mb-4" />
								<p className="text-muted-foreground">No reviewed submissions yet</p>
							</CardContent>
						</Card>
					) : (
						<div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
							{reviewedSubmissions.map((submission) => (
								<Card
									key={submission.id}
									className="cursor-pointer hover:shadow-lg transition-shadow"
									onClick={() => setSelectedSubmission(submission)}
								>
									<CardHeader>
										<div className="flex justify-between items-start">
											<div>
												<CardTitle className="text-lg">
													{submission.vehicleYear} {submission.vehicleMake} {submission.vehicleModel}
												</CardTitle>
												<CardDescription>{submission.participantName}</CardDescription>
											</div>
											{getStatusBadge(submission.status)}
										</div>
									</CardHeader>
									<CardContent>
										<div className="aspect-video relative overflow-hidden rounded-lg bg-gray-100">
											{submission.images[0] ? (
												<img
													src={submission.images[0]}
													alt={`${submission.vehicleMake} ${submission.vehicleModel}`}
													className="object-cover w-full h-full"
												/>
											) : (
												<div className="flex items-center justify-center h-full">
													<ImageIcon className="w-8 h-8 text-gray-400" />
												</div>
											)}
										</div>
										<div className="mt-4 space-y-1">
											<div className="flex items-center text-sm text-muted-foreground">
												<Calendar className="w-4 h-4 mr-1" />
												Submitted: {format(new Date(submission.submittedAt), 'MMM dd, yyyy')}
											</div>
											{submission.reviewedAt && (
												<div className="flex items-center text-sm text-muted-foreground">
													<Clock className="w-4 h-4 mr-1" />
													Reviewed: {format(new Date(submission.reviewedAt), 'MMM dd, yyyy')}
												</div>
											)}
										</div>
									</CardContent>
								</Card>
							))}
						</div>
					)}
				</TabsContent>
			</Tabs>

			{/* Submission Detail Dialog */}
			<Dialog open={!!selectedSubmission} onOpenChange={() => setSelectedSubmission(null)}>
				<DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
					{selectedSubmission && (
						<>
							<DialogHeader>
								<DialogTitle>
									{selectedSubmission.vehicleYear} {selectedSubmission.vehicleMake} {selectedSubmission.vehicleModel}
								</DialogTitle>
								<DialogDescription>
									Submission by {selectedSubmission.participantName}
								</DialogDescription>
							</DialogHeader>

							<div className="space-y-6">
								{/* Image Gallery */}
								<div className="space-y-4">
									<div className="aspect-video relative overflow-hidden rounded-lg bg-gray-100">
										<img
											src={selectedSubmission.images[currentImageIndex]}
											alt={`Vehicle image ${currentImageIndex + 1}`}
											className="object-contain w-full h-full"
										/>
										{selectedSubmission.images.length > 1 && (
											<>
												<Button
													variant="ghost"
													size="icon"
													className="absolute left-2 top-1/2 -translate-y-1/2"
													onClick={(e) => {
														e.stopPropagation();
														setCurrentImageIndex((prev) =>
															prev === 0 ? selectedSubmission.images.length - 1 : prev - 1
														);
													}}
												>
													<ChevronLeft className="h-4 w-4" />
												</Button>
												<Button
													variant="ghost"
													size="icon"
													className="absolute right-2 top-1/2 -translate-y-1/2"
													onClick={(e) => {
														e.stopPropagation();
														setCurrentImageIndex((prev) =>
															prev === selectedSubmission.images.length - 1 ? 0 : prev + 1
														);
													}}
												>
													<ChevronRight className="h-4 w-4" />
												</Button>
											</>
										)}
									</div>
									{selectedSubmission.images.length > 1 && (
										<div className="flex gap-2 justify-center">
											{selectedSubmission.images.map((_, index) => (
												<button
													key={index}
													className={`w-2 h-2 rounded-full transition-colors ${index === currentImageIndex ? 'bg-primary' : 'bg-gray-300'
														}`}
													onClick={() => setCurrentImageIndex(index)}
												/>
											))}
										</div>
									)}
								</div>

								{/* Participant Details */}
								<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
									<div className="space-y-2">
										<h3 className="font-semibold">Participant Information</h3>
										<div className="space-y-1">
											<div className="flex items-center gap-2 text-sm">
												<User className="w-4 h-4 text-muted-foreground" />
												<span>{selectedSubmission.participantName}</span>
											</div>
											<div className="flex items-center gap-2 text-sm">
												<Mail className="w-4 h-4 text-muted-foreground" />
												<a href={`mailto:${selectedSubmission.participantEmail}`} className="text-primary hover:underline">
													{selectedSubmission.participantEmail}
												</a>
											</div>
											{selectedSubmission.participantPhone && (
												<div className="flex items-center gap-2 text-sm">
													<Phone className="w-4 h-4 text-muted-foreground" />
													<a href={`tel:${selectedSubmission.participantPhone}`} className="text-primary hover:underline">
														{selectedSubmission.participantPhone}
													</a>
												</div>
											)}
										</div>
									</div>

									<div className="space-y-2">
										<h3 className="font-semibold">Vehicle Details</h3>
										<div className="space-y-1 text-sm">
											<div>
												<span className="font-medium">Year:</span> {selectedSubmission.vehicleYear}
											</div>
											<div>
												<span className="font-medium">Make:</span> {selectedSubmission.vehicleMake}
											</div>
											<div>
												<span className="font-medium">Model:</span> {selectedSubmission.vehicleModel}
											</div>
										</div>
									</div>
								</div>

								{selectedSubmission.vehicleDescription && (
									<div className="space-y-2">
										<h3 className="font-semibold">Description</h3>
										<p className="text-sm text-muted-foreground">{selectedSubmission.vehicleDescription}</p>
									</div>
								)}

								{selectedSubmission.vehicleModifications && (
									<div className="space-y-2">
										<h3 className="font-semibold">Modifications</h3>
										<p className="text-sm text-muted-foreground">{selectedSubmission.vehicleModifications}</p>
									</div>
								)}

								{selectedSubmission.reviewNotes && (
									<Alert>
										<AlertCircle className="h-4 w-4" />
										<AlertDescription>
											<strong>Review Notes:</strong> {selectedSubmission.reviewNotes}
										</AlertDescription>
									</Alert>
								)}

								{/* Actions */}
								{selectedSubmission.status === 'pending' && (
									<div className="flex gap-2 justify-end">
										<Button
											variant="outline"
											onClick={() => {
												setActionDialog({
													open: true,
													type: 'deny',
													submission: selectedSubmission,
												});
											}}
										>
											<XCircle className="w-4 h-4 mr-2" />
											Deny
										</Button>
										<Button
											onClick={() => {
												setActionDialog({
													open: true,
													type: 'approve',
													submission: selectedSubmission,
												});
											}}
										>
											<CheckCircle className="w-4 h-4 mr-2" />
											Approve
										</Button>
									</div>
								)}
							</div>
						</>
					)}
				</DialogContent>
			</Dialog>

			{/* Action Confirmation Dialog */}
			<Dialog open={actionDialog.open} onOpenChange={(open) => !processing && setActionDialog({ ...actionDialog, open })}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>
							{actionDialog.type === 'approve' ? 'Approve' : 'Deny'} Submission
						</DialogTitle>
						<DialogDescription>
							{actionDialog.type === 'approve'
								? 'Approving this submission will allow the participant to complete their ticket purchase.'
								: 'Denying this submission will prevent the participant from purchasing a ticket.'}
						</DialogDescription>
					</DialogHeader>

					<div className="space-y-4">
						<div>
							<Label htmlFor="action-notes">
								Notes {actionDialog.type === 'deny' && <span className="text-destructive">*</span>}
							</Label>
							<Textarea
								id="action-notes"
								value={actionNotes}
								onChange={(e) => setActionNotes(e.target.value)}
								placeholder={
									actionDialog.type === 'approve'
										? 'Optional: Add any notes about this approval'
										: 'Required: Please provide a reason for denial'
								}
								rows={3}
							/>
						</div>
					</div>

					<DialogFooter>
						<Button
							variant="outline"
							onClick={() => setActionDialog({ open: false, type: null, submission: null })}
							disabled={processing}
						>
							Cancel
						</Button>
						<Button
							variant={actionDialog.type === 'approve' ? 'default' : 'destructive'}
							onClick={handleAction}
							disabled={processing || (actionDialog.type === 'deny' && !actionNotes.trim())}
						>
							{processing && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
							{actionDialog.type === 'approve' ? 'Approve' : 'Deny'}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</div>
	);
}
