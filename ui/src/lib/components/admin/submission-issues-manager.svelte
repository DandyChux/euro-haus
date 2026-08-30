<script lang="ts">
	import { toast } from "svelte-sonner";

	import apiClient from "$lib/api";
	import type { IssueSubmission } from "$lib/schemas/submission";
	import type { Price } from "$lib/schemas/price";

	import { Input } from "$lib/components/ui/input";
	import { Button } from "$lib/components/ui/button";
	import { Checkbox } from "$lib/components/ui/checkbox";
	import { Card } from "$lib/components/ui/card";
	import { formatDate } from "$lib/utils";
	import { Badge } from "../ui/badge";

	interface Props {
		submissions: IssueSubmission[];
		onRefresh?: () => void | Promise<void>;
	}

	interface PaymentStatus {
		has_payment: boolean;
		payment_status?: string;
		payment_amount?: number;
		payment_currency?: string;
		checkout_url?: string;
		error_message?: string;
		email_sent_at?: string;
	}

	interface PaymentResponse {
		success?: boolean;
		message?: string;
	}

	let { submissions, onRefresh }: Props = $props();

	let activeTab = $state<"all" | "payment" | "email" | "ticket">("all");

	let searchTerm = $state("");
	let selectedStatuses = $state<string[]>([]);
	let selectedIssueTypes = $state<string[]>([]);

	let debugMode = $state(false);
	let showAll = $state(false);
	let includeId = $state("");

	let showFilters = $state(false);
	let showPaymentStatus = $state(false);
	let showCreatePayment = $state(false);

	let selectedSubmission = $state<IssueSubmission | null>(null);
	let paymentStatus = $state<PaymentStatus | null>(null);

	let prices = $state<Price[]>([]);
	let selectedPriceId = $state("");

	let checkingPayment = $state(false);
	let creatingPayment = $state(false);
	let resendingEmail = $state(false);
	let loadingPrices = $state(false);
	let busyAction = $state<string | null>(null);

	const paymentIssueTypes = [
		"no_payment",
		"payment_check_failed",
		"payment_expired",
		"payment_incomplete",
		"missing_payment_intent",
		"payment_intent_check_failed",
		"payment_not_succeeded",
		"missing_checkout_data",
		"incomplete_payment_process",
		"orphaned_checkout_session",
		"capture_failed",
		"payment_processing",
		"payment_requires_capture",
		"capture_method_mismatch",
	];

	const issueTypes = $derived(
		[
			...new Set(
				submissions.flatMap((submission) => submission.issues ?? []),
			),
		].sort(),
	);

	const filteredSubmissions = $derived(
		submissions.filter((submission) => {
			const query = searchTerm.trim().toLowerCase();

			if (query) {
				const matchesSearch = [
					submission.participant_name,
					submission.participant_email,
					submission.vehicle_make,
					submission.vehicle_model,
					submission.id,
				].some((value) => value?.toLowerCase().includes(query));

				if (!matchesSearch) return false;
			}

			if (
				selectedStatuses.length > 0 &&
				!selectedStatuses.includes(submission.status)
			) {
				return false;
			}

			if (selectedIssueTypes.length > 0) {
				const hasSelectedIssue = (submission.issues ?? []).some(
					(issue) => selectedIssueTypes.includes(issue),
				);

				if (!hasSelectedIssue) return false;
			}

			return true;
		}),
	);

	const paymentIssues = $derived(
		filteredSubmissions.filter((submission) =>
			(submission.issues ?? []).some((issue) =>
				paymentIssueTypes.includes(issue),
			),
		),
	);

	const emailIssues = $derived(
		filteredSubmissions.filter((submission) =>
			(submission.issues ?? []).includes("email_not_sent"),
		),
	);

	const ticketIssues = $derived(
		filteredSubmissions.filter((submission) =>
			(submission.issues ?? []).includes("no_ticket_created"),
		),
	);

	const visibleSubmissions = $derived(
		activeTab === "payment"
			? paymentIssues
			: activeTab === "email"
				? emailIssues
				: activeTab === "ticket"
					? ticketIssues
					: filteredSubmissions,
	);

	async function repairPayment(submission: IssueSubmission) {
		await runAction(
			submission.id,
			`/admin/submissions/${submission.id}/repair-payment`,
			"Payment references repaired.",
		);
	}

	async function retryApproval(submission: IssueSubmission) {
		await runAction(
			submission.id,
			`/admin/submissions/${submission.id}/retry-approval`,
			"Approval/payment processing retried.",
		);
	}

	async function retryTicket(submission: IssueSubmission) {
		await runAction(
			submission.id,
			`/admin/submissions/${submission.id}/retry-ticket`,
			"Ticket and ticket email processing retried.",
		);
	}

	async function runAction(
		id: string,
		endpoint: string,
		successMessage: string,
		body: unknown = {},
	) {
		busyAction = `${id}:${endpoint}`;

		try {
			const response = await apiClient.post<{
				message?: string;
			}>(endpoint, body);

			toast.success(response.message ?? successMessage);
			await refreshSubmissions();
		} catch (error) {
			toast.error(
				error instanceof Error ? error.message : "Action failed.",
			);
		} finally {
			busyAction = null;
		}
	}

	function getEventName(submission: IssueSubmission): string {
		return submission.event_slug ?? "Euro Haus Event";
	}

	async function refreshSubmissions() {
		if (onRefresh) {
			await onRefresh();
		}
	}

	function formatIssue(issue: string): string {
		return issue.replaceAll("_", " ");
	}

	function hasPaymentIssue(submission: IssueSubmission): boolean {
		return (submission.issues ?? []).some((issue) =>
			paymentIssueTypes.includes(issue),
		);
	}

	function hasEmailIssue(submission: IssueSubmission): boolean {
		return (submission.issues ?? []).includes("email_not_sent");
	}

	function hasTicketIssue(submission: IssueSubmission): boolean {
		return (submission.issues ?? []).includes("no_ticket_created");
	}

	function hasUnpaidCheckout(submission: IssueSubmission): boolean {
		return (
			submission.status === "approved" &&
			Boolean(submission.checkout_session_id) &&
			!submission.payment_intent_id
		);
	}

	async function applyServerFilters() {
		showFilters = false;
		await refreshSubmissions();
		toast.success("Filters applied.");
	}

	function resetFilters() {
		debugMode = false;
		showAll = false;
		includeId = "";
		searchTerm = "";
		selectedStatuses = [];
		selectedIssueTypes = [];
		showFilters = false;

		void refreshSubmissions();
	}

	async function checkPaymentStatus(submission: IssueSubmission) {
		checkingPayment = true;
		selectedSubmission = submission;

		try {
			paymentStatus = await apiClient.get<PaymentStatus>(
				`/admin/submissions/${submission.id}/payment-status`,
			);

			showPaymentStatus = true;
		} catch (error) {
			console.error("Failed to check payment status:", error);
			toast.error("Failed to check payment status.");
		} finally {
			checkingPayment = false;
		}
	}

	async function openCreatePayment(submission: IssueSubmission) {
		selectedSubmission = submission;
		selectedPriceId = "";
		prices = [];
		showCreatePayment = true;

		await loadPrices(submission.event_id);
	}

	async function loadPrices(eventId: string) {
		loadingPrices = true;

		try {
			const response = await apiClient.get<{
				prices?: Price[];
			}>(`/events/${eventId}/prices`);

			prices = (response.prices ?? []).filter((price) => price.active);

			if (prices.length === 1) {
				selectedPriceId = prices[0].id;
			}
		} catch (error) {
			console.error("Failed to load event prices:", error);
			toast.error("Failed to load ticket prices.");
		} finally {
			loadingPrices = false;
		}
	}

	async function createPayment() {
		if (!selectedSubmission || !selectedPriceId) {
			toast.error("Please select a ticket price.");
			return;
		}

		creatingPayment = true;

		try {
			const response = await apiClient.post<PaymentResponse>(
				`/admin/submissions/${selectedSubmission.id}/create-payment`,
				{
					priceId: selectedPriceId,
					eventName: getEventName(selectedSubmission),
				},
			);

			if (response.success) {
				toast.success(response.message ?? "Payment link email queued.");

				showCreatePayment = false;
				selectedPriceId = "";

				await refreshSubmissions();
			}
		} catch (error) {
			console.error("Failed to create payment:", error);
			toast.error("Failed to create payment link.");
		} finally {
			creatingPayment = false;
		}
	}

	async function resendApprovalEmail(submission: IssueSubmission) {
		resendingEmail = true;

		try {
			const response = await apiClient.post<{
				success?: boolean;
				message?: string;
			}>(`/admin/submissions/${submission.id}/resend-email`);

			if (response.success) {
				toast.success(response.message ?? "Approval email resent.");
			} else {
				toast.error(response.message ?? "Failed to resend email.");
			}

			await refreshSubmissions();
		} catch (error) {
			console.error("Failed to resend approval email:", error);
			toast.error("Failed to resend approval email.");
		} finally {
			resendingEmail = false;
		}
	}
</script>

<div class="space-y-6">
	<div class="flex flex-col gap-3 md:flex-row md:items-center">
		<Input
			bind:value={searchTerm}
			placeholder="Search by name, email, vehicle, or ID..."
			class="md:max-w-sm"
		/>

		<div class="flex gap-2 md:ml-auto">
			<Button
				type="button"
				variant="outline"
				onclick={() => (showFilters = !showFilters)}
			>
				Filters
			</Button>

			<Button
				type="button"
				variant="outline"
				onclick={() => void refreshSubmissions()}
			>
				Refresh
			</Button>
		</div>
	</div>

	{#if showFilters}
		<Card class="space-y-5 p-5">
			<div>
				<h2 class="font-semibold">Submission filters</h2>

				<p class="text-sm text-muted-foreground">
					Control which issue records are loaded and displayed.
				</p>
			</div>

			<div class="grid gap-4 md:grid-cols-2">
				<label class="flex items-center gap-2 text-sm">
					<Checkbox bind:checked={debugMode} />
					Debug mode
				</label>

				<label class="flex items-center gap-2 text-sm">
					<Checkbox bind:checked={showAll} />
					Show all submissions
				</label>
			</div>

			<label class="space-y-2 text-sm font-medium">
				<span>Include submission ID</span>

				<Input
					bind:value={includeId}
					placeholder="Optional submission ID"
				/>
			</label>

			<div>
				<p class="mb-2 text-sm font-medium">Status</p>

				<div class="flex flex-wrap gap-4">
					{#each ["approved", "pending", "denied"] as status}
						<label class="flex items-center gap-2 text-sm">
							<Checkbox
								checked={selectedStatuses.includes(status)}
								onCheckedChange={(checked) => {
									if (checked === true) {
										selectedStatuses = [
											...selectedStatuses,
											status,
										];
									} else {
										selectedStatuses =
											selectedStatuses.filter(
												(value) => value !== status,
											);
									}
								}}
							/>

							<span class="capitalize">{status}</span>
						</label>
					{/each}
				</div>
			</div>

			<div>
				<p class="mb-2 text-sm font-medium">Issue types</p>

				<div class="grid gap-2 md:grid-cols-2">
					{#each issueTypes as issue}
						<label class="flex items-center gap-2 text-sm">
							<Checkbox
								checked={selectedIssueTypes.includes(issue)}
								onCheckedChange={(checked) => {
									if (checked === true) {
										selectedIssueTypes = [
											...selectedIssueTypes,
											issue,
										];
									} else {
										selectedIssueTypes =
											selectedIssueTypes.filter(
												(value) => value !== issue,
											);
									}
								}}
							/>

							<span class="capitalize">
								{formatIssue(issue)}
							</span>
						</label>
					{/each}
				</div>
			</div>

			<div class="flex justify-end gap-2">
				<Button type="button" variant="ghost" onclick={resetFilters}>
					Reset
				</Button>

				<Button type="button" onclick={applyServerFilters}>
					Apply filters
				</Button>
			</div>
		</Card>
	{/if}

	<div class="flex flex-wrap gap-2 border-b">
		<Button
			type="button"
			variant={activeTab === "all" ? "default" : "ghost"}
			onclick={() => (activeTab = "all")}
		>
			All issues ({filteredSubmissions.length})
		</Button>

		<Button
			type="button"
			variant={activeTab === "payment" ? "default" : "ghost"}
			onclick={() => (activeTab = "payment")}
		>
			Payment ({paymentIssues.length})
		</Button>

		<Button
			type="button"
			variant={activeTab === "email" ? "default" : "ghost"}
			onclick={() => (activeTab = "email")}
		>
			Email ({emailIssues.length})
		</Button>

		<Button
			type="button"
			variant={activeTab === "ticket" ? "default" : "ghost"}
			onclick={() => (activeTab = "ticket")}
		>
			Ticket ({ticketIssues.length})
		</Button>
	</div>

	{#if visibleSubmissions.length === 0}
		<Card class="p-8 text-center text-muted-foreground">
			No submission issues found.
		</Card>
	{:else}
		<div class="grid gap-4">
			{#each visibleSubmissions as submission (submission.id)}
				<Card class="space-y-5 p-5">
					<div
						class="flex flex-col justify-between gap-4 md:flex-row"
					>
						<div>
							<h2 class="text-lg font-semibold">
								{submission.participant_name}
							</h2>

							<p class="text-sm text-muted-foreground">
								{submission.participant_email}
							</p>

							<p class="mt-1 text-xs text-muted-foreground">
								{submission.id}
							</p>
						</div>

						<div class="inline-grid grid-cols-1 gap-2">
							<Badge class="text-xs capitalize" variant="circle">
								{submission.status}
							</Badge>

							{#if submission.status === "approved" && (submission.checkout_session_id || submission.payment_intent_id)}
								<Button
									type="button"
									variant="outline"
									disabled={busyAction !== null}
									onclick={() => retryApproval(submission)}
								>
									Retry approval/capture
								</Button>
							{/if}

							{#if hasTicketIssue(submission)}
								<Button
									type="button"
									variant="outline"
									disabled={busyAction !== null}
									onclick={() => retryTicket(submission)}
								>
									Retry ticket/email
								</Button>
							{/if}

							{#if hasEmailIssue(submission) && submission.status === "approved"}
								<Button
									type="button"
									variant="outline"
									disabled={busyAction !== null}
									onclick={() =>
										resendApprovalEmail(submission)}
								>
									Resend approval email
								</Button>
							{/if}
						</div>
					</div>

					<div class="grid gap-4 text-sm md:grid-cols-4">
						<div>
							<p class="text-muted-foreground">Event</p>
							<p class="font-medium">
								{getEventName(submission)}
							</p>
						</div>

						<div>
							<p class="text-muted-foreground">Vehicle</p>
							<p class="font-medium">
								{submission.vehicle_year}
								{submission.vehicle_make}
								{submission.vehicle_model}
							</p>
						</div>

						<div>
							<p class="text-muted-foreground">Submitted</p>
							<p class="font-medium">
								{formatDate(submission.submitted_at, {
									dateStyle: "medium",
								})}
							</p>
						</div>

						<!-- <div>
								<p class="text-muted-foreground">Reviewed</p>
								<p class="font-medium">
									{formatDate(submission.reviewed_at, {
										dateStyle: "medium",
									})}
								</p>
							</div> -->
					</div>

					{#if submission.issues?.length}
						<div
							class="rounded-xl border border-yellow-300/50 bg-yellow-50 p-4 dark:bg-yellow-950/20"
						>
							<p class="font-medium">Issues detected</p>

							<ul class="mt-2 list-inside list-disc text-sm">
								{#each submission.issues as issue}
									<li class="capitalize">
										{formatIssue(issue)}
									</li>
								{/each}
							</ul>
						</div>
					{/if}

					<div class="flex flex-wrap gap-2">
						{#if hasPaymentIssue(submission)}
							<Button
								type="button"
								variant="outline"
								onclick={() => checkPaymentStatus(submission)}
								disabled={checkingPayment}
							>
								{checkingPayment
									? "Checking…"
									: "Check payment"}
							</Button>

							<Button
								type="button"
								onclick={() => openCreatePayment(submission)}
							>
								Create payment
							</Button>
						{/if}

						{#if hasEmailIssue(submission) && submission.status === "approved"}
							<Button
								type="button"
								variant="outline"
								onclick={() => resendApprovalEmail(submission)}
								disabled={resendingEmail}
							>
								{resendingEmail
									? "Sending…"
									: "Resend approval email"}
							</Button>
						{/if}
					</div>

					<div
						class="flex flex-wrap gap-3 border-t pt-3 text-xs text-muted-foreground"
					>
						{#if submission.checkout_session_id}
							<span>
								Session:
								{submission.checkout_session_id.slice(0, 8)}…
							</span>
						{/if}

						{#if submission.payment_intent_id}
							<span>
								Payment:
								{submission.payment_intent_id.slice(0, 8)}…
							</span>
						{/if}
					</div>
				</Card>
			{/each}
		</div>
	{/if}
</div>

{#if showPaymentStatus && paymentStatus}
	<div
		class="fixed inset-0 z-50 overflow-y-auto bg-background/80 p-4 backdrop-blur-sm"
	>
		<div class="mx-auto max-w-md">
			<Card class="space-y-5 p-5">
				<div class="flex items-start justify-between">
					<div>
						<h2 class="text-lg font-semibold">Payment status</h2>

						<p class="text-sm text-muted-foreground">
							{selectedSubmission?.participant_name}
						</p>
					</div>

					<Button
						type="button"
						variant="ghost"
						onclick={() => (showPaymentStatus = false)}
					>
						Close
					</Button>
				</div>

				<div class="space-y-3 text-sm">
					<div class="flex justify-between">
						<span class="text-muted-foreground"> Has payment </span>

						<strong>
							{paymentStatus.has_payment ? "Yes" : "No"}
						</strong>
					</div>

					{#if paymentStatus.payment_status}
						<div class="flex justify-between">
							<span class="text-muted-foreground"> Status </span>

							<strong>
								{paymentStatus.payment_status}
							</strong>
						</div>
					{/if}

					{#if paymentStatus.payment_amount}
						<div class="flex justify-between">
							<span class="text-muted-foreground"> Amount </span>

							<strong>
								{(paymentStatus.payment_amount / 100).toFixed(
									2,
								)}
								{paymentStatus.payment_currency?.toUpperCase()}
							</strong>
						</div>
					{/if}

					{#if paymentStatus.email_sent_at}
						<div class="flex justify-between">
							<span class="text-muted-foreground">
								Email sent
							</span>

							<strong>
								{formatDate(paymentStatus.email_sent_at, {
									dateStyle: "medium",
								})}
							</strong>
						</div>
					{/if}
				</div>

				{#if paymentStatus.error_message}
					<p
						class="rounded-lg border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive"
					>
						{paymentStatus.error_message}
					</p>
				{/if}
			</Card>
		</div>
	</div>
{/if}

{#if showCreatePayment && selectedSubmission}
	<div
		class="fixed inset-0 z-50 overflow-y-auto bg-background/80 p-4 backdrop-blur-sm"
	>
		<div class="mx-auto max-w-md">
			<Card class="space-y-5 p-5">
				<div class="flex items-start justify-between">
					<div>
						<h2 class="text-lg font-semibold">
							Create payment link
						</h2>

						<p class="text-sm text-muted-foreground">
							{selectedSubmission.participant_name}
						</p>
					</div>

					<Button
						type="button"
						variant="ghost"
						onclick={() => (showCreatePayment = false)}
					>
						Close
					</Button>
				</div>

				{#if loadingPrices}
					<p class="py-6 text-center text-sm text-muted-foreground">
						Loading ticket prices…
					</p>
				{:else}
					<label class="space-y-2 text-sm font-medium">
						<span>Ticket price</span>

						<select
							class="w-full rounded-md border bg-background px-3 py-2"
							bind:value={selectedPriceId}
						>
							<option value="">Choose a ticket type</option>

							{#each prices as price (price.id)}
								<option value={price.id}>
									{price.nickname || "Ticket"} —
									{(price.unit_amount / 100).toFixed(2)}
									{price.currency.toUpperCase()}
								</option>
							{/each}
						</select>
					</label>

					<div class="flex justify-end gap-2">
						<Button
							type="button"
							variant="outline"
							onclick={() => (showCreatePayment = false)}
						>
							Cancel
						</Button>

						<Button
							type="button"
							onclick={createPayment}
							disabled={!selectedPriceId || creatingPayment}
						>
							{creatingPayment
								? "Creating…"
								: "Create payment link"}
						</Button>
					</div>
				{/if}
			</Card>
		</div>
	</div>
{/if}
