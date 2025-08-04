import { apiClient } from '../api';
import type { VehicleSubmission, SubmissionCreateRequest } from '../interfaces/submission';

export interface SubmissionWithFiles extends SubmissionCreateRequest {
	images: File[];
}

export interface PaymentStatus {
	hasPayment: boolean;
	paymentStatus?: string;
	paymentAmount?: number;
	paymentCurrency?: string;
	checkoutURL?: string;
	errorMessage?: string;
	emailSent?: boolean;
	emailSentAt?: string;
}

export interface SubmissionWithIssues extends VehicleSubmission {
	issues?: string[];
}

export const submissionService = {
	async createSubmission(data: SubmissionWithFiles): Promise<VehicleSubmission> {
		try {
			// Create FormData for multipart upload
			const formData = new FormData();

			// Append text fields
			Object.entries(data).forEach(([key, value]) => {
				if (key !== 'images' && value !== undefined) {
					// Handle array values specially - join with newlines for vehicleModifications
					if (Array.isArray(value)) {
						formData.append(key, value.join('\n'));
					} else {
						formData.append(key, String(value));
					}
				}
			});

			// Append image files
			data.images.forEach((file, index) => {
				formData.append(`images`, file);
			});

			const response = await apiClient.post<VehicleSubmission>('/submissions', formData, {
				headers: {
					'Content-Type': 'multipart/form-data',
				},
			});

			return response.data;
		} catch (error) {
			console.error('Failed to create submission:', error);
			throw new Error('Failed to submit vehicle information');
		}
	},

	async getEventSubmissions(eventId: string): Promise<VehicleSubmission[]> {
		try {
			const response = await apiClient.get<{ submissions: VehicleSubmission[] }>(
				`/admin/submissions/${eventId}`
			);
			return response.data.submissions;
		} catch (error) {
			console.error('Failed to fetch submissions:', error);
			throw new Error('Failed to load submissions');
		}
	},

	async getSubmission(submissionId: string): Promise<VehicleSubmission> {
		try {
			const response = await apiClient.get<VehicleSubmission>(`/submissions/${submissionId}`);
			return response.data;
		} catch (error) {
			console.error('Failed to fetch submission:', error);
			throw new Error('Failed to load submission details');
		}
	},

	async approveSubmission(submissionId: string, notes?: string): Promise<VehicleSubmission> {
		try {
			const response = await apiClient.put<VehicleSubmission>(
				`/admin/submissions/${submissionId}/approve`,
				{ notes }
			);
			return response.data;
		} catch (error) {
			console.error('Failed to approve submission:', error);
			throw new Error('Failed to approve submission');
		}
	},

	async denySubmission(submissionId: string, notes: string): Promise<VehicleSubmission> {
		try {
			const response = await apiClient.put<VehicleSubmission>(
				`/admin/submissions/${submissionId}/deny`,
				{ notes }
			);
			return response.data;
		} catch (error) {
			console.error('Failed to deny submission:', error);
			throw new Error('Failed to deny submission');
		}
	},

	async createCheckoutWithSubmission(
		submissionId: string,
		priceId: string,
		eventName: string
	): Promise<{ sessionUrl: string }> {
		try {
			const response = await apiClient.post<{ sessionUrl: string }>(
				'/checkout/submission',
				{
					submissionId,
					priceId,
					eventName,
				}
			);
			return response.data;
		} catch (error) {
			console.error('Failed to create checkout session:', error);
			throw new Error('Failed to create checkout session');
		}
	},

	// Get all submissions with issues
	async getSubmissionsWithIssues(): Promise<SubmissionWithIssues[]> {
		try {
			const response = await apiClient.get<{ submissions: SubmissionWithIssues[] }>(
				'/admin/submissions/issues'
			);
			return response.data.submissions || [];
		} catch (error) {
			console.error('Failed to fetch submissions with issues:', error);
			throw new Error('Failed to load submissions with issues');
		}
	},

	// Check payment status for a submission
	async getPaymentStatus(submissionId: string): Promise<PaymentStatus> {
		try {
			const response = await apiClient.get<PaymentStatus>(
				`/admin/submissions/${submissionId}/payment-status`
			);
			return response.data;
		} catch (error) {
			console.error('Failed to fetch payment status:', error);
			throw new Error('Failed to check payment status');
		}
	},

	// Create payment for submission
	async createPayment(submissionId: string, priceId: string, eventName: string): Promise<{ sessionUrl: string }> {
		try {
			const response = await apiClient.post<{ sessionUrl: string }>(
				`/admin/submissions/${submissionId}/create-payment`,
				{ priceId, eventName }
			);
			return response.data;
		} catch (error) {
			console.error('Failed to create payment:', error);
			throw new Error('Failed to create payment');
		}
	},

	// Resend approval email
	async resendApprovalEmail(submissionId: string): Promise<{ success: boolean; message: string }> {
		try {
			const response = await apiClient.post<{ success: boolean; message: string }>(
				`/admin/submissions/${submissionId}/resend-email`
			);
			return response.data;
		} catch (error) {
			console.error('Failed to resend approval email:', error);
			throw new Error('Failed to resend approval email');
		}
	},

	// Get pending submissions count
	async getPendingSubmissionsCount(): Promise<number> {
		try {
			const response = await apiClient.get<{ count: number }>(
				'/admin/submissions/pending-count'
			);
			return response.data.count || 0;
		} catch (error) {
			console.error('Failed to fetch pending submissions count:', error);
			return 0;
		}
	},

	validateImages(files: File[]): { valid: boolean; error?: string } {
		const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB
		const ALLOWED_TYPES = ['image/jpeg', 'image/png', 'image/webp'];

		for (const file of files) {
			if (!ALLOWED_TYPES.includes(file.type)) {
				return { valid: false, error: `Invalid file type: ${file.name}. Only JPEG, PNG, and WebP are allowed.` };
			}
			if (file.size > MAX_FILE_SIZE) {
				return { valid: false, error: `File too large: ${file.name}. Maximum size is 10MB.` };
			}
		}

		return { valid: true };
	},
};
