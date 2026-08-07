<script lang="ts">
	import { fade, fly } from "svelte/transition";
	import { toast } from "svelte-sonner";

	import apiClient from "$lib/api";
	import type { VehicleSubmission } from "$lib/schemas/event";

	interface Props {
		eventId: string;
		eventSlug: string;
		priceId: string;
		ticketTier: string;
		ticketPrice: number;
		ticketQuantity: number;
		onsucceed: (submission: VehicleSubmission) => Promise<void>;
		oncancel: () => void;
	}

	interface VehicleDetails {
		participantName: string;
		participantEmail: string;
		participantPhone: string;
		vehicleYear: string;
		vehicleMake: string;
		vehicleModel: string;
		vehicleDescription: string;
	}

	interface UploadProgress {
		loaded: number;
		total: number;
		percent: number;
	}

	let {
		eventId,
		eventSlug,
		priceId,
		ticketTier,
		ticketPrice,
		ticketQuantity,
		onsucceed,
		oncancel,
	}: Props = $props();

	let step = $state<1 | 2>(1);

	let details = $state<VehicleDetails>({
		participantName: "",
		participantEmail: "",
		participantPhone: "",
		vehicleYear: "",
		vehicleMake: "",
		vehicleModel: "",
		vehicleDescription: "",
	});

	let modifications = $state<string[]>([]);
	let files = $state<File[]>([]);
	let previews = $state<string[]>([]);
	let uploadProgress = $state(0);
	let submitting = $state(false);
	let errorMessage = $state("");

	let formElement: HTMLFormElement;

	const total = $derived(ticketPrice * ticketQuantity);

	function addModification() {
		modifications = [...modifications, ""];
	}

	function removeModification(index: number) {
		modifications = modifications.filter(
			(_, itemIndex) => itemIndex !== index,
		);
	}

	function addImages(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const selected = Array.from(input.files ?? []);

		const valid = selected.filter(
			(file) =>
				["image/jpeg", "image/png", "image/webp"].includes(file.type) &&
				file.size <= 10 * 1024 * 1024,
		);

		if (valid.length !== selected.length) {
			errorMessage = "Use JPEG, PNG, or WebP files up to 10MB each.";
		}

		if (files.length + valid.length > 5) {
			errorMessage = "You can upload a maximum of five images.";
		}

		const nextFiles = [...files, ...valid].slice(0, 5);

		files = nextFiles;
		previews = nextFiles.map((file) => URL.createObjectURL(file));

		// Allows the same file to be selected again after removing it.
		input.value = "";
	}

	function removeImage(index: number) {
		files = files.filter((_, fileIndex) => fileIndex !== index);
		previews = previews.filter((_, fileIndex) => fileIndex !== index);
	}

	function continueToImages() {
		errorMessage = "";

		if (!formElement.reportValidity()) {
			return;
		}

		step = 2;
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		errorMessage = "";

		if (step === 1) {
			continueToImages();
			return;
		}

		if (files.length === 0) {
			errorMessage =
				"Add at least one image of your vehicle to continue.";
			return;
		}

		if (!priceId) {
			errorMessage = "The selected ticket price is unavailable.";
			return;
		}

		submitting = true;
		uploadProgress = 0;

		const body = new FormData();

		// These names match internal/handlers/submission.go.
		body.set("event_id", eventId);
		body.set("event_slug", eventSlug);
		body.set("participant_name", details.participantName);
		body.set("participant_email", details.participantEmail);
		body.set("participant_phone", details.participantPhone);
		body.set("vehicle_year", details.vehicleYear);
		body.set("vehicle_make", details.vehicleMake);
		body.set("vehicle_model", details.vehicleModel);
		body.set("vehicle_description", details.vehicleDescription);
		body.set(
			"vehicle_modifications",
			modifications
				.filter((modification) => modification.trim())
				.join("\n"),
		);

		// This must be the Stripe Price ID, not the display name.
		body.set("price_id", priceId);

		for (const file of files) {
			body.append("images", file);
		}

		try {
			const submission = await apiClient.upload<VehicleSubmission>(
				"/submissions",
				body,
				({ percent }: UploadProgress) => {
					uploadProgress = percent;
				},
			);

			await onsucceed(submission);

			toast.success(
				"Vehicle submitted. Continue to checkout when ready.",
			);
		} catch (error) {
			console.error("Vehicle submission failed:", error);

			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to submit your vehicle right now.";
		} finally {
			submitting = false;
		}
	}
</script>

<section
	class="submission-shell"
	aria-labelledby="submission-title"
	in:fly={{ y: 18, duration: 300 }}
>
	<div class="submission-header">
		<div>
			<p class="eyebrow">Vehicle entry</p>

			<h2 id="submission-title">Show us your build.</h2>

			<p class="submission-intro">
				A few details help our team understand what you are bringing to
				the event.
			</p>
		</div>

		<button
			type="button"
			class="close-button"
			onclick={oncancel}
			aria-label="Close vehicle submission"
		>
			×
		</button>
	</div>

	<div class="ticket-summary">
		<div>
			<span>Ticket</span>
			<strong>{ticketTier || "Vehicle entry"}</strong>
		</div>

		<div>
			<span>Quantity</span>
			<strong>{ticketQuantity}</strong>
		</div>

		<div>
			<span>Total</span>
			<strong>${total.toFixed(2)}</strong>
		</div>
	</div>

	<div class="progress" aria-label={`Step ${step} of 2`}>
		<div class="progress-track">
			<span style={`width: ${step === 1 ? 50 : 100}%`}></span>
		</div>

		<div class="progress-labels">
			<span class:current={step === 1}>01 · Your details</span>
			<span class:current={step === 2}>
				02 · Photos and modifications
			</span>
		</div>
	</div>

	{#if errorMessage}
		<p class="error-message" role="alert">
			{errorMessage}
		</p>
	{/if}

	<form bind:this={formElement} onsubmit={submit}>
		{#if step === 1}
			<div class="form-grid" in:fade={{ duration: 180 }}>
				<label class="full">
					Full name
					<input
						bind:value={details.participantName}
						required
						minlength="2"
						placeholder="Alex Morgan"
					/>
				</label>

				<label>
					Email
					<input
						bind:value={details.participantEmail}
						required
						type="email"
						placeholder="alex@example.com"
					/>
				</label>

				<label>
					Phone <span>(optional)</span>
					<input
						bind:value={details.participantPhone}
						type="tel"
						placeholder="(555) 123-4567"
					/>
				</label>

				<label>
					Year
					<input
						bind:value={details.vehicleYear}
						required
						type="number"
						pattern="[0-9]{4}"
						maxlength={4}
						inputmode="numeric"
						placeholder="2024"
					/>
				</label>

				<label>
					Make
					<input
						bind:value={details.vehicleMake}
						required
						minlength="2"
						placeholder="Porsche"
					/>
				</label>

				<label>
					Model
					<input
						bind:value={details.vehicleModel}
						required
						minlength="2"
						placeholder="911 GT3"
					/>
				</label>

				<label class="full">
					Tell us about the car <span>(optional)</span>
					<textarea
						bind:value={details.vehicleDescription}
						rows="4"
						placeholder="What makes this car special?"></textarea>
				</label>
			</div>

			<div class="form-actions">
				<button type="button" class="secondary" onclick={oncancel}>
					Cancel
				</button>

				<button type="submit" class="primary">
					Continue
					<span aria-hidden="true">→</span>
				</button>
			</div>
		{:else}
			<div class="photo-step" in:fade={{ duration: 180 }}>
				<div class="upload-copy">
					<p class="eyebrow">01 / Photos</p>

					<h3>Let the details speak.</h3>

					<p>
						Upload up to five clear photos. Your first photo will be
						used as the cover image.
					</p>
				</div>

				<label class="dropzone">
					<input
						type="file"
						accept="image/jpeg,image/png,image/webp"
						multiple
						onchange={addImages}
						disabled={submitting}
					/>

					<span class="upload-icon">＋</span>
					<strong>Choose vehicle photos</strong>
					<small>JPEG, PNG, or WebP · 10MB max each</small>
				</label>

				{#if previews.length > 0}
					<div class="preview-grid">
						{#each previews as preview, index (preview)}
							<figure>
								<img
									src={preview}
									alt={`Vehicle preview ${index + 1}`}
								/>

								<button
									type="button"
									onclick={() => removeImage(index)}
									aria-label={`Remove photo ${index + 1}`}
								>
									×
								</button>

								{#if index === 0}
									<figcaption>Cover</figcaption>
								{/if}
							</figure>
						{/each}
					</div>
				{/if}

				<div class="modifications">
					<div class="section-row">
						<div>
							<p class="eyebrow">02 / Details</p>
							<h3>What have you changed?</h3>
						</div>

						<button
							type="button"
							class="secondary small"
							onclick={addModification}
						>
							＋ Add modification
						</button>
					</div>

					{#if modifications.length === 0}
						<p class="empty-state">No modifications added yet.</p>
					{:else}
						{#each modifications as _, index (index)}
							<div class="modification-row">
								<input
									bind:value={modifications[index]}
									placeholder="e.g. KW suspension, forged wheels"
								/>

								<button
									type="button"
									onclick={() => removeModification(index)}
									aria-label={`Remove modification ${index + 1}`}
								>
									×
								</button>
							</div>
						{/each}
					{/if}
				</div>

				<div class="form-actions">
					<button
						type="button"
						class="secondary"
						onclick={() => (step = 1)}
						disabled={submitting}
					>
						← Back
					</button>

					<button
						type="submit"
						class="primary"
						disabled={submitting || files.length === 0}
					>
						{#if submitting}
							Uploading {uploadProgress}%…
						{:else}
							Submit and continue
						{/if}
					</button>
				</div>
			</div>
		{/if}
	</form>
</section>

<style>
	.submission-shell {
		max-width: 760px;
		margin: 0 auto;
		padding: clamp(24px, 5vw, 54px);
		background: var(--background);
		color: var(--foreground);
		box-shadow: 0 24px 80px
			color-mix(in srgb, var(--foreground) 20%, transparent);
	}

	.submission-header,
	.section-row,
	.form-actions,
	.progress-labels {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 20px;
	}

	.submission-header h2 {
		margin: 8px 0 12px;
		font-family: var(--font-display);
		font-size: clamp(52px, 8vw, 92px);
		letter-spacing: -0.055em;
		line-height: 0.8;
		text-transform: uppercase;
	}

	.submission-intro {
		max-width: 460px;
		margin: 0;
		color: var(--muted);
	}

	.close-button,
	.modification-row button,
	.preview-grid figure button {
		border: 0;
		background: transparent;
		color: inherit;
		cursor: pointer;
		font-size: 24px;
	}

	.ticket-summary {
		display: grid;
		grid-template-columns: 1fr 1fr 1fr;
		gap: 1px;
		margin: 36px 0;
		background: var(--border);
		border: 1px solid var(--border);
	}

	.ticket-summary div {
		display: grid;
		gap: 6px;
		padding: 16px;
		background: var(--background);
	}

	.ticket-summary span,
	label span {
		color: var(--muted);
		font-size: 11px;
	}

	.ticket-summary strong {
		font-size: 15px;
	}

	.progress {
		margin-bottom: 34px;
	}

	.progress-track {
		height: 3px;
		background: var(--border);
	}

	.progress-track span {
		display: block;
		height: 100%;
		background: var(--primary);
		transition: width 0.3s ease;
	}

	.progress-labels {
		margin-top: 10px;
		color: var(--muted);
		font-size: 10px;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.progress-labels .current {
		color: var(--foreground);
	}

	.error-message {
		padding: 12px 14px;
		border-left: 3px solid var(--destructive);
		color: var(--destructive);
		background: color-mix(in srgb, var(--destructive) 8%, transparent);
	}

	.form-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 18px;
	}

	label {
		display: grid;
		gap: 8px;
		color: var(--foreground);
		font-size: 12px;
	}

	label.full {
		grid-column: 1 / -1;
	}

	input,
	textarea {
		width: 100%;
		border: 1px solid var(--border);
		border-radius: 0;
		padding: 13px 14px;
		background: transparent;
		color: inherit;
		font: inherit;
	}

	input:focus,
	textarea:focus {
		outline: 2px solid var(--primary);
		outline-offset: 1px;
	}

	.form-actions {
		align-items: center;
		margin-top: 34px;
	}

	.primary,
	.secondary {
		border: 1px solid var(--foreground);
		padding: 13px 18px;
		cursor: pointer;
		font: inherit;
	}

	.primary {
		background: var(--foreground);
		color: var(--background);
	}

	.primary:disabled {
		cursor: wait;
		opacity: 0.55;
	}

	.secondary {
		background: transparent;
	}

	.small {
		padding: 9px 12px;
		font-size: 11px;
	}

	.photo-step {
		display: grid;
		gap: 26px;
	}

	.upload-copy h3,
	.modifications h3 {
		margin: 6px 0;
		font-family: var(--font-display);
		font-size: 34px;
		text-transform: uppercase;
	}

	.upload-copy p:last-child,
	.empty-state {
		margin: 0;
		color: var(--muted);
	}

	.dropzone {
		display: grid;
		justify-items: center;
		gap: 8px;
		padding: 36px 20px;
		border: 1px dashed var(--border);
		text-align: center;
		cursor: pointer;
	}

	.dropzone input {
		display: none;
	}

	.upload-icon {
		font-size: 32px;
	}

	.dropzone small {
		color: var(--muted);
	}

	.preview-grid {
		display: grid;
		grid-template-columns: repeat(5, 1fr);
		gap: 10px;
	}

	.preview-grid figure {
		position: relative;
		min-width: 0;
		margin: 0;
	}

	.preview-grid img {
		display: block;
		width: 100%;
		aspect-ratio: 1;
		object-fit: cover;
	}

	.preview-grid figure button {
		position: absolute;
		top: 3px;
		right: 5px;
		color: white;
		text-shadow: 0 1px 4px black;
	}

	.preview-grid figcaption {
		position: absolute;
		bottom: 5px;
		left: 5px;
		padding: 3px 6px;
		background: var(--foreground);
		color: var(--background);
		font-size: 9px;
		text-transform: uppercase;
	}

	.modifications {
		padding-top: 20px;
		border-top: 1px solid var(--border);
	}

	.modification-row {
		display: flex;
		gap: 8px;
		margin-top: 10px;
	}

	.modification-row input {
		flex: 1;
	}

	@media (max-width: 600px) {
		.form-grid {
			grid-template-columns: 1fr;
		}

		label.full {
			grid-column: auto;
		}

		.ticket-summary {
			grid-template-columns: 1fr;
		}

		.preview-grid {
			grid-template-columns: repeat(3, 1fr);
		}

		.submission-header h2 {
			font-size: 58px;
		}
	}
</style>
