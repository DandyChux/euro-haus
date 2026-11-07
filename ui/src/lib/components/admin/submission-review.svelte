<script lang="ts">
	import { toast } from "svelte-sonner";
	import apiClient from "$lib/api";
	import type { VehicleSubmission } from "$lib/schemas/submission";
	import { Button } from "$lib/components/ui/button";
	import * as Card from "$lib/components/ui/card";
	import * as Dialog from "$lib/components/ui/dialog";
	import { Textarea } from "$lib/components/ui/textarea";
	import { Badge } from "$lib/components/ui/badge";
	import { formatDate } from "$lib/utils";

	interface Price {
		id: string;
		nickname?: string;
		unit_amount: number;
		currency: string;
		active: boolean;
	}

	interface Props {
		eventName: string;
		submissions: VehicleSubmission[];
		loading?: boolean;
		errorMessage?: string;
		onSubmissionUpdated?: () => void | Promise<void>;
	}

	let {
		eventName,
		submissions = [],
		loading = false,
		errorMessage = "",
		onSubmissionUpdated,
	}: Props = $props();

	let selectedSubmission = $state<VehicleSubmission | null>(null);
	let isDetailOpen = $derived(selectedSubmission !== null);

	let activeTab = $state<"pending" | "reviewed">("pending");
	let currentImageIndex = $state(0);
	let processing = $state(false);

	let action = $state<"approve" | "deny" | null>(null);
	let actionNotes = $state("");

	// Tier editing & upgrade modal
	let eventPrices = $state<Price[]>([]);
	let loadingPrices = $state(false);
	let selectedApprovalPriceId = $state("");

	let showUpgradeModal = $state(false);
	let selectedTargetPriceId = $state("");
	let isComplimentary = $state(false);
	let autoApproveOnUpgrade = $state(true);
	let upgradeNotes = $state("");
	let upgrading = $state(false);

	const pendingSubmissions = $derived(
		submissions.filter((submission) => submission.status === "pending"),
	);

	const reviewedSubmissions = $derived(
		submissions
			.filter((submission) => submission.status !== "pending")
			.sort(
				(a, b) =>
					new Date(b.reviewed_at ?? "").getTime() -
					new Date(a.reviewed_at ?? "").getTime(),
			),
	);

	const visibleSubmissions = $derived(
		activeTab === "pending" ? pendingSubmissions : reviewedSubmissions,
	);

	async function loadEventTiers(eventId: string) {
		loadingPrices = true;
		try {
			const res = await apiClient.get<{ prices: Price[] }>(
				`/admin/events/${eventId}/prices`,
			);
			eventPrices = res.prices ?? [];
		} catch (err) {
			console.error("Failed to load event tiers:", err);
		} finally {
			loadingPrices = false;
		}
	}

	function openSubmission(submission: VehicleSubmission) {
		selectedSubmission = submission;
		currentImageIndex = 0;
		action = null;
		actionNotes = "";
		selectedApprovalPriceId = submission.price_id ?? "";
		void loadEventTiers(submission.event_id);
	}

	function closeSubmission() {
		selectedSubmission = null;
		action = null;
		actionNotes = "";
		showUpgradeModal = false;
	}

	function showPreviousImage() {
		if (!selectedSubmission) return;
		currentImageIndex =
			currentImageIndex === 0
				? selectedSubmission.images.length - 1
				: currentImageIndex - 1;
	}

	function showNextImage() {
		if (!selectedSubmission) return;
		currentImageIndex =
			currentImageIndex === selectedSubmission.images.length - 1
				? 0
				: currentImageIndex + 1;
	}

	async function submitAction() {
		if (!selectedSubmission || !action) return;

		if (action === "deny" && !actionNotes.trim()) {
			toast.error("A reason is required when denying a submission.");
			return;
		}

		processing = true;
		try {
			const endpoint =
				action === "approve"
					? `/admin/submissions/${selectedSubmission.id}/approve`
					: `/admin/submissions/${selectedSubmission.id}/deny`;

			await apiClient.put(endpoint, {
				notes: actionNotes.trim(),
				price_id:
					action === "approve" ? selectedApprovalPriceId : undefined,
			});

			toast.success(
				action === "approve"
					? "Submission approved."
					: "Submission denied.",
			);

			closeSubmission();
			await onSubmissionUpdated?.();
		} catch (error) {
			toast.error(
				action === "approve"
					? "Failed to approve submission."
					: "Failed to deny submission.",
			);
		} finally {
			processing = false;
		}
	}

	// Upgrade calculations
	let currentPrice = $derived(
		eventPrices.find((p) => p.id === selectedSubmission?.price_id),
	);

	let targetPrice = $derived(
		eventPrices.find((p) => p.id === selectedTargetPriceId),
	);

	let priceDifference = $derived.by(() => {
		if (!targetPrice) return 0;
		const currentAmount = currentPrice ? currentPrice.unit_amount : 0;
		const diff = targetPrice.unit_amount - currentAmount;
		return diff > 0 ? diff : 0;
	});

	function openUpgrade() {
		if (!selectedSubmission) return;
		selectedTargetPriceId = "";
		isComplimentary = false;
		autoApproveOnUpgrade = selectedSubmission.status === "pending";
		upgradeNotes = "";
		showUpgradeModal = true;
	}

	async function submitUpgrade() {
		if (!selectedSubmission || !selectedTargetPriceId) {
			toast.error("Select a target tier");
			return;
		}

		upgrading = true;
		try {
			const res = await apiClient.post<{ message?: string }>(
				`/admin/submissions/${selectedSubmission.id}/upgrade`,
				{
					target_price_id: selectedTargetPriceId,
					complimentary: isComplimentary,
					auto_approve: autoApproveOnUpgrade,
					notes: upgradeNotes.trim(),
				},
			);

			toast.success(res.message ?? "Tier updated successfully");
			showUpgradeModal = false;
			closeSubmission();
			await onSubmissionUpdated?.();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : "Upgrade failed");
		} finally {
			upgrading = false;
		}
	}
</script>

{#if loading}
	<div class="flex min-h-64 items-center justify-center text-muted">
		Loading submissions…
	</div>
{:else}
	<div class="space-y-6">
		<div class="flex items-center justify-between">
			<div>
				<h2 class="text-2xl font-bold">Vehicle submissions</h2>
				<p class="text-muted">
					Review and approve participant vehicles for {eventName}.
				</p>
			</div>

			<div class="flex gap-2">
				<Button
					variant={activeTab === "pending" ? "default" : "outline"}
					onclick={() => (activeTab = "pending")}
				>
					Pending ({pendingSubmissions.length})
				</Button>
				<Button
					variant={activeTab === "reviewed" ? "default" : "outline"}
					onclick={() => (activeTab = "reviewed")}
				>
					Reviewed ({reviewedSubmissions.length})
				</Button>
			</div>
		</div>

		{#if errorMessage}
			<p
				class="rounded-xl border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive"
			>
				{errorMessage}
			</p>
		{/if}

		{#if visibleSubmissions.length === 0}
			<Card.Root class="p-8 text-center text-muted">
				No {activeTab} submissions found.
			</Card.Root>
		{:else}
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each visibleSubmissions as submission (submission.id)}
					<Card.Root
						class="cursor-pointer overflow-hidden transition-all hover:ring-2 hover:ring-primary"
					>
						<button
							type="button"
							class="w-full text-left"
							onclick={() => openSubmission(submission)}
						>
							{#if submission.images?.[0]}
								<img
									src={submission.images[0]}
									alt={`${submission.vehicle_year} ${submission.vehicle_make} ${submission.vehicle_model}`}
									class="h-48 w-full object-cover"
								/>
							{:else}
								<div
									class="flex h-48 w-full items-center justify-center bg-muted text-muted-foreground"
								>
									No image
								</div>
							{/if}

							<Card.Header class="space-y-1 p-4 pb-2">
								<div class="flex items-center justify-between">
									<Card.Title class="text-base font-semibold">
										{submission.vehicle_year}
										{submission.vehicle_make}
										{submission.vehicle_model}
									</Card.Title>
									<Badge class="capitalize" variant="outline"
										>{submission.status}</Badge
									>
								</div>
								<Card.Description class="text-sm text-muted">
									{submission.participant_name}
								</Card.Description>
							</Card.Header>

							<Card.Footer
								class="p-4 pt-0 text-xs font-medium text-muted"
							>
								Tier: {submission.price_nickname || "General"}
							</Card.Footer>
						</button>
					</Card.Root>
				{/each}
			</div>
		{/if}
	</div>
{/if}

<!-- Submission Detail Dialog -->
<Dialog.Root
	open={isDetailOpen}
	onOpenChange={(open) => {
		if (!open) closeSubmission();
	}}
>
	{#if selectedSubmission}
		<Dialog.Content
			class="max-h-[92vh] max-w-3xl overflow-y-auto sm:max-w-3xl"
		>
			<Dialog.Header>
				<div
					class="flex flex-col justify-between gap-2 sm:flex-row sm:items-center"
				>
					<div>
						<Dialog.Title class="text-xl font-bold">
							{selectedSubmission.vehicle_year}
							{selectedSubmission.vehicle_make}
							{selectedSubmission.vehicle_model}
						</Dialog.Title>
						<Dialog.Description>
							Submitted by {selectedSubmission.participant_name} ({selectedSubmission.participant_email})
						</Dialog.Description>
					</div>

					<!-- Available for both pending and reviewed submissions -->
					<Button variant="outline" size="sm" onclick={openUpgrade}>
						Upgrade / Change Tier
					</Button>
				</div>
			</Dialog.Header>

			<div class="space-y-5">
				<!-- Image Gallery -->
				{#if selectedSubmission.images?.length}
					<div class="relative overflow-hidden rounded-xl bg-black">
						<img
							src={selectedSubmission.images[currentImageIndex]}
							alt={`${selectedSubmission.vehicle_year} ${selectedSubmission.vehicle_make} ${selectedSubmission.vehicle_model}`}
							class="max-h-96 w-full object-contain"
						/>
						{#if selectedSubmission.images.length > 1}
							<div
								class="absolute inset-0 flex items-center justify-between p-2"
							>
								<Button
									variant="outline"
									size="sm"
									onclick={showPreviousImage}>←</Button
								>
								<Button
									variant="outline"
									size="sm"
									onclick={showNextImage}>→</Button
								>
							</div>
						{/if}
					</div>
				{/if}

				<div class="grid gap-3 sm:grid-cols-2 text-sm">
					<div>
						<span class="text-muted">Current Tier:</span>
						<strong class="ml-1"
							>{selectedSubmission.price_nickname ||
								selectedSubmission.price_id}</strong
						>
					</div>
					<div>
						<span class="text-muted">Status:</span>
						<strong class="ml-1 capitalize"
							>{selectedSubmission.status}</strong
						>
					</div>
				</div>

				{#if selectedSubmission.vehicle_description}
					<div>
						<h3 class="font-semibold text-sm">Description</h3>
						<p class="text-sm text-muted whitespace-pre-wrap">
							{selectedSubmission.vehicle_description}
						</p>
					</div>
				{/if}

				{#if selectedSubmission.vehicle_modifications}
					<div>
						<h3 class="font-semibold text-sm">Modifications</h3>
						<p class="text-sm text-muted whitespace-pre-wrap">
							{selectedSubmission.vehicle_modifications}
						</p>
					</div>
				{/if}

				<!-- Action for Pending Submissions: Quick Approve / Deny -->
				{#if selectedSubmission.status === "pending"}
					{#if action}
						<Card.Root class="border bg-muted/20">
							<Card.Header class="pb-2">
								<Card.Title class="text-base font-semibold">
									{action === "approve"
										? "Approve Submission"
										: "Deny Submission"}
								</Card.Title>
								<Card.Description>
									{action === "approve"
										? "Select confirmed ticket tier and add optional approval notes."
										: "A reason is required when denying a submission."}
								</Card.Description>
							</Card.Header>

							<Card.Content class="space-y-3">
								{#if action === "approve"}
									<div class="space-y-1">
										<label
											for="approval-tier-select"
											class="text-xs font-semibold uppercase text-muted"
										>
											Confirmed Ticket Tier
										</label>
										<select
											id="approval-tier-select"
											bind:value={selectedApprovalPriceId}
											class="w-full rounded-md border bg-background px-3 py-2 text-sm"
										>
											{#each eventPrices as price (price.id)}
												<option value={price.id}>
													{price.nickname || "Tier"} — {(
														price.unit_amount / 100
													).toFixed(2)}
													{price.currency.toUpperCase()}
													{!price.active
														? " (Internal/Hidden)"
														: ""}
												</option>
											{/each}
										</select>
									</div>
								{/if}

								<Textarea
									bind:value={actionNotes}
									placeholder={action === "approve"
										? "Optional approval notes"
										: "Reason for denial"}
									rows={3}
								/>
							</Card.Content>

							<Card.Footer class="justify-end gap-2 pt-2">
								<Button
									variant="ghost"
									onclick={() => (action = null)}
									disabled={processing}>Cancel</Button
								>
								<Button
									variant={action === "deny"
										? "destructive"
										: "default"}
									onclick={submitAction}
									disabled={processing}
								>
									{processing
										? "Saving…"
										: action === "approve"
											? "Confirm Approval"
											: "Confirm Denial"}
								</Button>
							</Card.Footer>
						</Card.Root>
					{:else}
						<Dialog.Footer class="border-t pt-4">
							<Button
								variant="outline"
								onclick={() => (action = "deny")}
							>
								Deny
							</Button>
							<Button onclick={() => (action = "approve")}>
								Approve
							</Button>
						</Dialog.Footer>
					{/if}
				{:else}
					<Dialog.Footer
						class="flex items-center justify-between border-t pt-4 sm:justify-between"
					>
						<p class="text-xs text-muted">
							Reviewed on {formatDate(
								selectedSubmission.reviewed_at ?? "",
							)}
						</p>
					</Dialog.Footer>
				{/if}
			</div>
		</Dialog.Content>
	{/if}
</Dialog.Root>

<!-- Upgrade / Change Tier Dialog (Usable on BOTH Pending & Reviewed Submissions) -->
<Dialog.Root
	open={showUpgradeModal}
	onOpenChange={(open) => {
		showUpgradeModal = open;
	}}
>
	{#if selectedSubmission}
		<Dialog.Content class="max-w-lg">
			<Dialog.Header>
				<Dialog.Title class="text-lg font-bold"
					>Upgrade / Change Vehicle Tier</Dialog.Title
				>
				<Dialog.Description>
					Update tier for {selectedSubmission.participant_name}
				</Dialog.Description>
			</Dialog.Header>

			<div class="space-y-3 text-sm py-2">
				<div class="flex justify-between">
					<span class="text-muted">Current Tier:</span>
					<strong
						>{selectedSubmission.price_nickname ||
							"General"}</strong
					>
				</div>

				<div class="space-y-1">
					<label
						for="upgrade-tier-select"
						class="text-xs font-semibold uppercase text-muted"
					>
						Target Tier
					</label>
					<select
						id="upgrade-tier-select"
						bind:value={selectedTargetPriceId}
						class="w-full rounded-md border bg-background px-3 py-2 text-sm"
					>
						<option value=""
							>Select target tier (e.g. Top 25)</option
						>
						{#each eventPrices as price (price.id)}
							{#if price.id !== selectedSubmission.price_id}
								<option value={price.id}>
									{price.nickname || "Tier"} — ${(
										price.unit_amount / 100
									).toFixed(2)}
									{!price.active
										? " [Hidden / Invite Only]"
										: ""}
								</option>
							{/if}
						{/each}
					</select>
				</div>

				{#if targetPrice}
					<div
						class="rounded-xl border border-primary/40 bg-primary/10 p-3 space-y-1"
					>
						<div class="flex justify-between font-semibold">
							<span>Price Difference:</span>
							<span>${(priceDifference / 100).toFixed(2)}</span>
						</div>
						<p class="text-xs text-muted">
							{!isComplimentary && priceDifference > 0
								? `Participant will be emailed a Stripe link to pay the $${(priceDifference / 100).toFixed(2)} difference.`
								: "Tier will be updated immediately without charging additional funds."}
						</p>
					</div>
				{/if}

				<!-- If still pending, offer one-click approval with the upgrade -->
				{#if selectedSubmission.status === "pending"}
					<label class="flex items-center gap-2 pt-2 text-xs">
						<input
							type="checkbox"
							bind:checked={autoApproveOnUpgrade}
							class="rounded"
						/>
						<span>Also mark submission as approved</span>
					</label>
				{/if}

				<label class="flex items-center gap-2 text-xs">
					<input
						type="checkbox"
						bind:checked={isComplimentary}
						class="rounded"
					/>
					<span>Complimentary (skip charging difference)</span>
				</label>

				<div>
					<label
						for="upgrade-notes"
						class="text-xs font-semibold uppercase text-muted"
					>
						Notes / Reason
					</label>
					<Textarea
						id="upgrade-notes"
						bind:value={upgradeNotes}
						placeholder="e.g. Selected for Top 25 showcase"
						rows={2}
					/>
				</div>
			</div>

			<Dialog.Footer>
				<Button
					variant="outline"
					onclick={() => (showUpgradeModal = false)}
				>
					Cancel
				</Button>
				<Button
					onclick={submitUpgrade}
					disabled={!selectedTargetPriceId || upgrading}
				>
					{upgrading
						? "Processing…"
						: !isComplimentary && priceDifference > 0
							? `Send Payment Link ($${(priceDifference / 100).toFixed(2)})`
							: "Apply Update"}
				</Button>
			</Dialog.Footer>
		</Dialog.Content>
	{/if}
</Dialog.Root>
