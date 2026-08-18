<script lang="ts">
	import { untrack } from "svelte";
	import { fade, fly } from "svelte/transition";
	import { toast } from "svelte-sonner";
	import { superForm, type SuperValidated } from "sveltekit-superforms";
	import { zod4Client } from "sveltekit-superforms/adapters";

	import * as Form from "$lib/components/ui/form";
	import { Input } from "$lib/components/ui/input";
	import { Textarea } from "$lib/components/ui/textarea";
	import { Button } from "$lib/components/ui/button";

	import apiClient from "$lib/api";
	import {
		vehicleSubmissionFormSchema,
		type VehicleSubmission,
		type VehicleSubmissionFormData,
	} from "$lib/schemas/submission";
	import type {
		RequirementAnswer,
		SubmissionRequirement,
	} from "$lib/schemas/submission";

	interface Props {
		data: {
			form: SuperValidated<VehicleSubmissionFormData>;
		};

		eventId: string;
		priceId: string;
		ticketTier: string;
		ticketPrice: number;
		ticketQuantity: number;
		requirements?: SubmissionRequirement[];

		onsucceed: (submission: VehicleSubmission) => Promise<void>;
		oncancel: () => void;
	}

	interface UploadProgress {
		loaded: number;
		total: number;
		percent: number;
	}

	let {
		data,
		eventId,
		priceId,
		ticketTier,
		ticketPrice,
		ticketQuantity,
		requirements,
		onsucceed,
		oncancel,
	}: Props = $props();

	const form = superForm(
		untrack(() => data.form),
		{
			SPA: true,
			dataType: "json",
			validators: zod4Client(vehicleSubmissionFormSchema),

			async onUpdate({ form }) {
				if (!form.valid) {
					console.error(
						"Vehicle submission validation failed:",
						form.errors,
					);
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

				if (!validateRequirements()) {
					return;
				}

				submitting = true;
				uploadProgress = 0;
				errorMessage = "";

				try {
					const body = new FormData();

					body.set("event_id", form.data.event_id);
					body.set("participant_name", form.data.participant_name);
					body.set("participant_email", form.data.participant_email);
					body.set(
						"participant_phone",
						form.data.participant_phone ?? "",
					);
					body.set("vehicle_year", form.data.vehicle_year);
					body.set("vehicle_make", form.data.vehicle_make);
					body.set("vehicle_model", form.data.vehicle_model);
					body.set(
						"vehicle_description",
						form.data.vehicle_description ?? "",
					);
					body.set(
						"vehicle_modifications",
						modifications
							.filter((modification) => modification.trim())
							.join("\n"),
					);
					body.set("price_id", form.data.price_id);
					body.set(
						"requirement_answers",
						JSON.stringify(requirementAnswers),
					);

					for (const file of files) {
						body.append("images", file);
					}

					const submission =
						await apiClient.upload<VehicleSubmission>(
							"/submissions",
							body,
							({ percent }: UploadProgress) => {
								uploadProgress = percent;
							},
						);

					if (!submission.id) {
						throw new Error(
							"The submission was created without an ID.",
						);
					}

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
			},
		},
	);

	const { form: formData, enhance, submitting: formSubmitting } = form;

	let step = $state<1 | 2>(1);
	let files = $state<File[]>([]);
	let previews = $state<string[]>([]);
	let modifications = $state<string[]>([]);
	let uploadProgress = $state(0);
	let submitting = $state(false);
	let errorMessage = $state("");
	let requirementAnswers = $state<Record<string, RequirementAnswer>>({});

	const total = $derived(ticketPrice * ticketQuantity);
	const isSubmitting = $derived(submitting || $formSubmitting);

	$formData.event_id = eventId;
	$formData.price_id = priceId;

	function updateRequirement(
		requirementId: string,
		value: RequirementAnswer,
	) {
		requirementAnswers = {
			...requirementAnswers,
			[requirementId]: value,
		};
	}

	function validateRequirements(): boolean {
		const missing = (requirements ?? []).filter((requirement) => {
			if (!requirement.required) return false;

			const value = requirementAnswers[requirement.id];

			return (
				value === undefined ||
				value === "" ||
				(requirement.type === "boolean" && value !== true)
			);
		});

		if (missing.length === 0) return true;

		errorMessage = `Complete: ${missing
			.map((requirement) => requirement.label)
			.join(", ")}.`;

		return false;
	}

	function addModification() {
		modifications = [...modifications, ""];
	}

	function updateModification(index: number, value: string) {
		modifications[index] = value;
	}

	function removeModification(index: number) {
		modifications = modifications.filter(
			(_, itemIndex) => itemIndex !== index,
		);
	}

	function addImages(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const selected = Array.from(input.files ?? []);

		errorMessage = "";

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

		input.value = "";
	}

	function removeImage(index: number) {
		const preview = previews[index];

		if (preview) {
			URL.revokeObjectURL(preview);
		}

		files = files.filter((_, fileIndex) => fileIndex !== index);
		previews = previews.filter((_, fileIndex) => fileIndex !== index);
	}

	async function continueToImages() {
		errorMessage = "";

		const validationErrors = await Promise.all([
			form.validate("participant_name"),
			form.validate("participant_email"),
			form.validate("vehicle_year"),
			form.validate("vehicle_make"),
			form.validate("vehicle_model"),
		]);

		if (validationErrors.some((errors) => (errors?.length ?? 0) > 0)) {
			return;
		}

		step = 2;
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

	<form method="POST" use:enhance>
		{#if step === 1}
			<div class="form-grid" in:fade={{ duration: 180 }}>
				<Form.Field {form} name="participant_name" class="full">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Full name</Form.Label>

							<Input
								{...props}
								bind:value={$formData.participant_name}
								placeholder="Alex Morgan"
								autocomplete="name"
							/>
						{/snippet}
					</Form.Control>

					<Form.FieldErrors />
				</Form.Field>

				<Form.Field {form} name="participant_email">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Email</Form.Label>

							<Input
								{...props}
								type="email"
								bind:value={$formData.participant_email}
								placeholder="alex@example.com"
								autocomplete="email"
							/>
						{/snippet}
					</Form.Control>

					<Form.FieldErrors />
				</Form.Field>

				<Form.Field {form} name="participant_phone">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>
								Phone <span>(optional)</span>
							</Form.Label>

							<Input
								{...props}
								type="tel"
								bind:value={$formData.participant_phone}
								placeholder="(555) 123-4567"
								autocomplete="tel"
							/>
						{/snippet}
					</Form.Control>

					<Form.FieldErrors />
				</Form.Field>

				<Form.Field {form} name="vehicle_year">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Year</Form.Label>

							<Input
								{...props}
								type="text"
								inputmode="numeric"
								maxlength={4}
								bind:value={$formData.vehicle_year}
								placeholder="2024"
							/>
						{/snippet}
					</Form.Control>

					<Form.FieldErrors />
				</Form.Field>

				<Form.Field {form} name="vehicle_make">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Make</Form.Label>

							<Input
								{...props}
								bind:value={$formData.vehicle_make}
								placeholder="Porsche"
								autocomplete="organization"
							/>
						{/snippet}
					</Form.Control>

					<Form.FieldErrors />
				</Form.Field>

				<Form.Field {form} name="vehicle_model">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Model</Form.Label>

							<Input
								{...props}
								bind:value={$formData.vehicle_model}
								placeholder="911 GT3"
							/>
						{/snippet}
					</Form.Control>

					<Form.FieldErrors />
				</Form.Field>

				{#if (requirements?.length ?? 0) > 0}
					<div class="full space-y-5">
						<div>
							<p class="eyebrow">Included items</p>
							<h3>Tell us what you need.</h3>
							<p>
								These details are based on your selected ticket
								tier.
							</p>
						</div>

						{#each requirements as requirement (requirement.id)}
							<div class="space-y-2">
								<label
									for={`requirement-${requirement.id}`}
									class="block text-sm font-medium"
								>
									{requirement.label}
									{#if requirement.required}
										<span aria-hidden="true">*</span>
									{:else}
										<span>(optional)</span>
									{/if}
								</label>

								{#if requirement.type === "textarea"}
									<Textarea
										id={`requirement-${requirement.id}`}
										value={String(
											requirementAnswers[
												requirement.id
											] ?? "",
										)}
										rows={4}
										required={requirement.required}
										disabled={isSubmitting}
										oninput={(event) =>
											updateRequirement(
												requirement.id,
												(
													event.currentTarget as HTMLTextAreaElement
												).value,
											)}
									/>
								{:else if requirement.type === "select"}
									<select
										id={`requirement-${requirement.id}`}
										class="w-full rounded-md border px-3 py-2"
										value={String(
											requirementAnswers[
												requirement.id
											] ?? "",
										)}
										required={requirement.required}
										disabled={isSubmitting}
										onchange={(event) =>
											updateRequirement(
												requirement.id,
												(
													event.currentTarget as HTMLSelectElement
												).value,
											)}
									>
										<option value=""
											>Choose an option</option
										>

										{#each requirement.options as option}
											<option value={option}
												>{option}</option
											>
										{/each}
									</select>
								{:else if requirement.type === "radio"}
									<div class="space-y-2">
										{#each requirement.options as option}
											<label
												class="flex items-center gap-2"
											>
												<input
													type="radio"
													name={`requirement-${requirement.id}`}
													value={option}
													checked={requirementAnswers[
														requirement.id
													] === option}
													disabled={isSubmitting}
													onchange={() =>
														updateRequirement(
															requirement.id,
															option,
														)}
												/>
												<span>{option}</span>
											</label>
										{/each}
									</div>
								{:else if requirement.type === "boolean"}
									<label class="flex items-center gap-2">
										<input
											id={`requirement-${requirement.id}`}
											type="checkbox"
											checked={requirementAnswers[
												requirement.id
											] === true}
											disabled={isSubmitting}
											onchange={(event) =>
												updateRequirement(
													requirement.id,
													(
														event.currentTarget as HTMLInputElement
													).checked,
												)}
										/>
										<span>
											{requirement.label}
											{#if requirement.required}*{/if}
										</span>
									</label>
								{:else}
									<Input
										id={`requirement-${requirement.id}`}
										type={requirement.type === "number"
											? "number"
											: "text"}
										value={String(
											requirementAnswers[
												requirement.id
											] ?? "",
										)}
										required={requirement.required}
										disabled={isSubmitting}
										oninput={(event) => {
											const value = (
												event.currentTarget as HTMLInputElement
											).value;

											updateRequirement(
												requirement.id,
												requirement.type === "number"
													? Number(value)
													: value,
											);
										}}
									/>
								{/if}
							</div>
						{/each}
					</div>
				{/if}

				<Form.Field {form} name="vehicle_description" class="full">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>
								Tell us about the car
								<span>(optional)</span>
							</Form.Label>

							<Textarea
								{...props}
								bind:value={$formData.vehicle_description}
								rows={4}
								placeholder="What makes this car special?"
							/>
						{/snippet}
					</Form.Control>

					<Form.FieldErrors />
				</Form.Field>
			</div>

			<div class="form-actions">
				<Button type="button" variant="outline" onclick={oncancel}>
					Cancel
				</Button>

				<Button type="button" onclick={continueToImages}>
					Continue
					<span aria-hidden="true">→</span>
				</Button>
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
						disabled={isSubmitting}
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

						<Button
							type="button"
							variant="outline"
							size="sm"
							onclick={addModification}
						>
							＋ Add modification
						</Button>
					</div>

					{#if modifications.length === 0}
						<p class="empty-state">No modifications added yet.</p>
					{:else}
						{#each modifications as modification, index (index)}
							<div class="modification-row">
								<Input
									value={modification}
									oninput={(event) =>
										updateModification(
											index,
											(
												event.currentTarget as HTMLInputElement
											).value,
										)}
									placeholder="e.g. KW suspension, forged wheels"
									disabled={isSubmitting}
								/>

								<Button
									type="button"
									variant="ghost"
									onclick={() => removeModification(index)}
									aria-label={`Remove modification ${index + 1}`}
								>
									×
								</Button>
							</div>
						{/each}
					{/if}
				</div>

				<div class="form-actions">
					<Button
						type="button"
						variant="outline"
						onclick={() => (step = 1)}
						disabled={isSubmitting}
					>
						← Back
					</Button>

					<Button
						type="submit"
						disabled={isSubmitting || files.length === 0}
					>
						{#if isSubmitting}
							Uploading {uploadProgress}%…
						{:else}
							Submit and continue
						{/if}
					</Button>
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

	.form-grid :global([data-slot="field"]) {
		min-width: 0;
	}

	.form-grid :global([data-slot="field"].full) {
		grid-column: 1 / -1;
	}

	.form-grid :global([data-slot="label"]) {
		color: var(--foreground);
		font-size: 12px;
	}

	.modification-row :global(input) {
		flex: 1;
	}

	.form-actions {
		align-items: center;
		margin-top: 34px;
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

	.modification-row :global(input) {
		flex: 1;
	}

	@media (max-width: 600px) {
		.form-grid {
			grid-template-columns: 1fr;
		}

		.form-grid :global([data-slot="field"].full) {
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
