<script lang="ts">
	import { untrack } from "svelte";
	import { superForm, type SuperValidated } from "sveltekit-superforms";
	import { zod4Client } from "sveltekit-superforms/adapters";
	import { toast } from "svelte-sonner";

	import * as Form from "$lib/components/ui/form";
	import { Input } from "$lib/components/ui/input";
	import { Textarea } from "$lib/components/ui/textarea";
	import { Button } from "$lib/components/ui/button";
	import { Card } from "$lib/components/ui/card";

	import { eventSchema, type Event } from "$lib/schemas/event";
	import { Checkbox } from "../ui/checkbox";
	import type { Price } from "$lib/schemas/price";
	import apiClient from "$lib/api";

	interface Props {
		data: {
			form: SuperValidated<Event>;
		};

		onsaved: (data: Event) => Promise<void>;
	}

	let { data, onsaved }: Props = $props();

	const form = superForm(
		untrack(() => data.form),
		{
			SPA: true,
			dataType: "json",
			validators: zod4Client(eventSchema),

			async onUpdate({ form }) {
				if (!form.valid) {
					console.error(
						"Event form validation failed: ",
						form.errors,
					);
					return;
				}

				try {
					await onsaved({
						...form.data,
						prices: hasTiers
							? form.data.prices
							: form.data.prices.slice(0, 1),
					});
					toast.success("Event saved.");
				} catch (error) {
					console.error("Saving event failed:", error);
					toast.error("Unable to save event.");
				}
			},
		},
	);

	const { form: formData, enhance, submitting } = form;

	let hasTiers = $state($formData.prices.length > 1);
	let uploadingImage = $state(false);
	let uploadProgress = $state(0);
	let imageInput: HTMLInputElement;
	let imageUrl = $state("");
	let imageUrlError = $state("");

	interface UploadedMediaResponse {
		success: boolean;
		file: {
			key: string;
			url: string;
			last_modified: string;
			size: number;
			type: "image" | "video" | "other";
			folder: string;
		};
		message: string;
	}

	function addImageUrl() {
		const url = imageUrl.trim();

		if (!url) {
			return;
		}

		try {
			const parsedUrl = new URL(url);

			if (!["http:", "https:"].includes(parsedUrl.protocol)) {
				throw new Error("Unsupported URL protocol");
			}
		} catch {
			imageUrlError = "Enter a valid HTTP or HTTPS image URL.";
			return;
		}

		$formData.images = [...$formData.images, url];
		imageUrl = "";
		imageUrlError = "";
	}

	function removeImage(index: number) {
		$formData.images = $formData.images.filter(
			(_, imageIndex) => imageIndex !== index,
		);
	}

	async function onFileUpload(event: any) {
		const input = event.currentTarget as HTMLInputElement;
		const selectedFiles = Array.from(input.files ?? []);

		if (selectedFiles.length === 0) {
			return;
		}

		const imageFiles = selectedFiles.filter((file) =>
			file.type.startsWith("image/"),
		);

		if (imageFiles.length !== selectedFiles.length) {
			toast.error(
				"Some files were skipped. Only image files are allowed.",
			);
		}

		if (imageFiles.length === 0) {
			input.value = "";
			return;
		}

		if (!$formData.slug.trim()) {
			generateSlug();
		}

		const eventSlug = $formData.slug.trim();

		if (!eventSlug) {
			toast.error("Add an event name and date before uploading images.");
			input.value = "";
			return;
		}

		uploadingImage = true;
		uploadProgress = 0;

		const uploadedUrls: string[] = [];

		try {
			for (const [index, file] of imageFiles.entries()) {
				const body = new FormData();

				body.append("file", file, file.name);
				body.append("folder", `events/${eventSlug}/images`);

				const response = await apiClient.upload<UploadedMediaResponse>(
					"/admin/media/upload",
					body,
					({ percent }) => {
						const completedFiles = index / imageFiles.length;
						const currentFileProgress =
							percent / 100 / imageFiles.length;

						uploadProgress = Math.round(
							(completedFiles + currentFileProgress) * 100,
						);
					},
				);

				uploadedUrls.push(response.file.url);
			}

			// Preserve existing order. If this is the first upload,
			// the first uploaded URL becomes the default image.
			$formData.images = [...$formData.images, ...uploadedUrls];

			uploadProgress = 100;

			toast.success(
				`Uploaded ${uploadedUrls.length} image${
					uploadedUrls.length === 1 ? "" : "s"
				}.`,
			);
		} catch (error) {
			console.error("Failed to upload event images:", error);

			toast.error(
				uploadedUrls.length > 0
					? `Uploaded ${uploadedUrls.length} image${
							uploadedUrls.length === 1 ? "" : "s"
						}, but another upload failed.`
					: "Failed to upload event images.",
			);

			// Preserve successfully uploaded files even if a later upload fails.
			if (uploadedUrls.length > 0) {
				$formData.images = [...$formData.images, ...uploadedUrls];
			}
		} finally {
			uploadingImage = false;
			input.value = "";
		}
	}

	function emptyPrice(): Price {
		return {
			id: "",
			stripe_product_id: $formData.stripe_product_id,

			unit_amount: 0,
			currency: "usd",
			nickname: "",
			description: "",

			active: true,

			features: [],
			default: false,
			most_popular: false,
			requires_approval: true,
			requires_submission: false,
			included_products: [],
			quantity: undefined,
			sold_out: false,
		};
	}

	function addPrice() {
		$formData.prices = [...$formData.prices, emptyPrice()];
	}

	function removePrice(index: number) {
		$formData.prices = $formData.prices.filter(
			(_, priceIndex) => priceIndex !== index,
		);
	}

	function addPriceFeature(priceIndex: number) {
		$formData.prices[priceIndex].features = [
			...$formData.prices[priceIndex].features,
			"",
		];
	}

	function removePriceFeature(priceIndex: number, featureIndex: number) {
		$formData.prices[priceIndex].features = $formData.prices[
			priceIndex
		].features.filter((_, index) => index !== featureIndex);
	}

	function setPriceAmount(priceIndex: number, event: any) {
		const rawValue = (event.currentTarget as HTMLInputElement).value;
		const amount = Number(rawValue);

		$formData.prices[priceIndex].unit_amount = Number.isFinite(amount)
			? Math.round(Math.max(0, amount) * 100)
			: 0;
	}

	function setPriceQuantity(priceIndex: number, event: any) {
		const rawValue = (event.currentTarget as HTMLInputElement).value;
		const quantity = Number(rawValue);

		$formData.prices[priceIndex].quantity = rawValue.trim()
			? Number.isFinite(quantity)
				? Math.max(1, Math.floor(quantity))
				: undefined
			: undefined;
	}

	function setRequiresSubmission(priceIndex: number, value: boolean) {
		$formData.prices[priceIndex].requires_submission = value;

		if (!value) {
			$formData.prices[priceIndex].requires_approval = false;
		}
	}

	function setHasTiers(value: boolean) {
		hasTiers = value;

		if (value && $formData.prices.length === 0) {
			addPrice();
		}
	}

	function addTag() {
		$formData.tags = [...$formData.tags, ""];
	}

	function removeTag(index: number) {
		$formData.tags = $formData.tags.filter(
			(_, itemIndex) => itemIndex !== index,
		);
	}

	function addAgendaItem() {
		$formData.agenda = [
			...$formData.agenda,
			{
				time: "",
				activity: "",
			},
		];
	}

	function removeAgendaItem(index: number) {
		$formData.agenda = $formData.agenda.filter(
			(_, itemIndex) => itemIndex !== index,
		);
	}

	function addIncludedItem() {
		$formData.includes = [...$formData.includes, ""];
	}

	function removeIncludedItem(index: number) {
		$formData.includes = $formData.includes.filter(
			(_, itemIndex) => itemIndex !== index,
		);
	}

	function addSponsor() {
		$formData.sponsors = [
			...$formData.sponsors,
			{
				name: "",
				tier: "",
				logo: "",
				url: "",
				description: "",
			},
		];
	}

	function removeSponsor(index: number) {
		$formData.sponsors = $formData.sponsors.filter(
			(_, sponsorIndex) => sponsorIndex !== index,
		);
	}

	function generateSlug() {
		$formData.slug = `${$formData.name}-${$formData.date.slice(0, 10)}`
			.toLowerCase()
			.normalize("NFKD")
			.replace(/[^a-z0-9]+/g, "-")
			.replace(/(^-|-$)/g, "");
	}
</script>

<form method="POST" use:enhance class="space-y-6">
	<Card class="space-y-6 p-5">
		<div class="grid gap-4 md:grid-cols-2">
			<Form.Field {form} name="name">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Event name</Form.Label>

						<Input {...props} bind:value={$formData.name} />
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>

			<Form.Field {form} name="slug">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>URL slug</Form.Label>

						<div class="flex gap-2">
							<Input {...props} bind:value={$formData.slug} />

							<Button
								type="button"
								variant="outline"
								onclick={generateSlug}
							>
								Generate
							</Button>
						</div>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>
		</div>

		<div class="grid gap-4 md:grid-cols-2">
			<Form.Field {form} name="date">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Date and time</Form.Label>

						<Input
							{...props}
							type="datetime-local"
							bind:value={$formData.date}
						/>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>

			<Form.Field {form} name="status">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Status</Form.Label>

						<select
							{...props}
							class="w-full rounded-2xl border bg-transparent px-3 py-2"
							bind:value={$formData.status}
						>
							<option value="upcoming"> Upcoming </option>

							<option value="ongoing"> Ongoing </option>

							<option value="completed"> Completed </option>

							<option value="cancelled"> Cancelled </option>

							<option value="sold_out"> Sold out </option>
						</select>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>
		</div>

		<div class="grid gap-4 md:grid-cols-2">
			<Form.Field {form} name="venue">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Venue</Form.Label>
						<Input {...props} bind:value={$formData.venue} />
					{/snippet}
				</Form.Control>
			</Form.Field>

			<Form.Field {form} name="location">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Location</Form.Label>
						<Input {...props} bind:value={$formData.location} />
					{/snippet}
				</Form.Control>
			</Form.Field>

			<Form.Field {form} name="organizer">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Organizer</Form.Label>
						<Input {...props} bind:value={$formData.organizer} />
					{/snippet}
				</Form.Control>
			</Form.Field>

			<Form.Field {form} name="capacity">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Total capacity</Form.Label>
						<Input
							{...props}
							type="number"
							min="1"
							bind:value={$formData.capacity}
						/>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>

			<Form.Field {form} name="available_spots">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Available spots</Form.Label>
						<Input
							{...props}
							type="number"
							min="0"
							bind:value={$formData.available_spots}
						/>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>
		</div>

		<Form.Field {form} name="description">
			<Form.Control>
				{#snippet children({ props })}
					<Form.Label>Short description</Form.Label>

					<Textarea {...props} bind:value={$formData.description} />
				{/snippet}
			</Form.Control>

			<Form.FieldErrors />
		</Form.Field>

		<Form.Field {form} name="long_description">
			<Form.Control>
				{#snippet children({ props })}
					<Form.Label>Long description</Form.Label>

					<Textarea
						{...props}
						rows={6}
						bind:value={$formData.long_description}
					/>
				{/snippet}
			</Form.Control>

			<Form.FieldErrors />
		</Form.Field>

		<div class="flex gap-6 text-sm">
			<Form.Field {form} name="active">
				<Form.Control>
					{#snippet children({ props })}
						<label class="flex items-center gap-2">
							<Checkbox
								{...props}
								bind:checked={$formData.active}
							/>
							Active
						</label>
					{/snippet}
				</Form.Control>
			</Form.Field>

			<Form.Field {form} name="featured">
				<Form.Control>
					{#snippet children({ props })}
						<label class="flex items-center gap-2">
							<Checkbox
								{...props}
								bind:checked={$formData.featured}
							/>
							Featured
						</label>
					{/snippet}
				</Form.Control>
			</Form.Field>
		</div>
	</Card>
	<Card class="space-y-4 p-5">
		<div class="flex items-center justify-between">
			<div>
				<h2 class="text-lg font-semibold">Event tags</h2>
				<p class="text-sm text-muted-foreground">
					Categorize the event.
				</p>
			</div>

			<Button type="button" variant="outline" onclick={addTag}>
				Add tag
			</Button>
		</div>

		<Form.Fieldset {form} name={"tags"}>
			<Form.Legend>Tags</Form.Legend>
			{#each $formData.tags as _, index (index)}
				<Form.Control>
					{#snippet children({ props })}
						<div class="flex gap-2">
							<Input
								{...props}
								bind:value={$formData.tags[index]}
								placeholder="BMW, Track Day"
							/>

							<Button
								type="button"
								variant="ghost"
								onclick={() => removeTag(index)}
							>
								Remove
							</Button>
						</div>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			{/each}
		</Form.Fieldset>
	</Card>

	<Card class="space-y-4 p-5">
		<div class="flex items-center justify-between">
			<div>
				<h2 class="text-lg font-semibold">Event schedule</h2>

				<p class="text-sm text-muted-foreground">
					Build the event agenda.
				</p>
			</div>

			<Button type="button" variant="outline" onclick={addAgendaItem}>
				Add item
			</Button>
		</div>

		<Form.Fieldset {form} name="agenda">
			<Form.Legend>Agenda</Form.Legend>

			{#each $formData.agenda as _, index (index)}
				<Form.Control>
					{#snippet children({ props })}
						<div
							class="grid gap-3 rounded-2xl border p-4 md:grid-cols-[10rem_minmax(0,1fr)_auto] md:items-start"
						>
							<div class="space-y-2">
								<label
									for={`agenda-time-${index}`}
									class="text-sm font-medium"
								>
									Time
								</label>

								<Input
									{...props}
									id={`agenda-time-${index}`}
									bind:value={$formData.agenda[index].time}
									placeholder="9:00 AM"
								/>
							</div>

							<div class="space-y-2">
								<label
									for={`agenda-activity-${index}`}
									class="text-sm font-medium"
								>
									Activity
								</label>

								<Input
									{...props}
									id={`agenda-activity-${index}`}
									bind:value={
										$formData.agenda[index].activity
									}
									placeholder="Registration and welcome"
								/>
							</div>

							<Button
								type="button"
								variant="ghost"
								class="md:mt-7"
								onclick={() => removeAgendaItem(index)}
							>
								Remove
							</Button>
						</div>
					{/snippet}
				</Form.Control>
			{/each}

			<Form.FieldErrors />
		</Form.Fieldset>
	</Card>

	<Card class="space-y-4 p-5">
		<div class="flex items-center justify-between">
			<div>
				<h2 class="text-lg font-semibold">What's included</h2>

				<p class="text-sm text-muted-foreground">
					List items included with the event ticket.
				</p>
			</div>

			<Button type="button" variant="outline" onclick={addIncludedItem}>
				Add item
			</Button>
		</div>

		<Form.Fieldset {form} name="includes">
			<Form.Legend>Included items</Form.Legend>

			{#each $formData.includes as _, index (index)}
				<Form.Control>
					{#snippet children({ props })}
						<div class="flex gap-2">
							<Input
								{...props}
								id={`included-item-${index}`}
								bind:value={$formData.includes[index]}
								placeholder="Lunch and refreshments"
							/>

							<Button
								type="button"
								variant="ghost"
								onclick={() => removeIncludedItem(index)}
							>
								Remove
							</Button>
						</div>
					{/snippet}
				</Form.Control>
			{/each}

			<Form.FieldErrors />
		</Form.Fieldset>
	</Card>

	<Card class="space-y-4 p-5">
		<div class="flex items-center justify-between">
			<div>
				<h2 class="text-lg font-semibold">Event sponsors</h2>

				<p class="text-sm text-muted-foreground">
					Sponsors use the canonical flat Sponsor model.
				</p>
			</div>

			<Button type="button" variant="outline" onclick={addSponsor}>
				Add sponsor
			</Button>
		</div>

		<Form.Fieldset {form} name="sponsors">
			<Form.Legend>Sponsors</Form.Legend>

			{#each $formData.sponsors as _, index (index)}
				<Form.Control>
					{#snippet children({ props })}
						<div class="space-y-4 rounded-2xl border p-4">
							<div class="flex justify-between">
								<h3 class="font-medium">
									Sponsor {index + 1}
								</h3>

								<Button
									type="button"
									variant="ghost"
									onclick={() => removeSponsor(index)}
								>
									Remove
								</Button>
							</div>

							<div class="grid gap-4 md:grid-cols-2">
								<div class="space-y-2">
									<label
										for={`sponsor-name-${index}`}
										class="text-sm font-medium"
									>
										Company name
									</label>

									<Input
										{...props}
										id={`sponsor-name-${index}`}
										bind:value={
											$formData.sponsors[index].name
										}
										placeholder="Porsche USA"
									/>
								</div>

								<div class="space-y-2">
									<label
										for={`sponsor-tier-${index}`}
										class="text-sm font-medium"
									>
										Sponsor tier
									</label>

									<Input
										{...props}
										id={`sponsor-tier-${index}`}
										bind:value={
											$formData.sponsors[index].tier
										}
										placeholder="Platinum"
									/>
								</div>

								<div class="space-y-2">
									<label
										for={`sponsor-logo-${index}`}
										class="text-sm font-medium"
									>
										Logo URL
									</label>

									<Input
										{...props}
										id={`sponsor-logo-${index}`}
										bind:value={
											$formData.sponsors[index].logo
										}
										placeholder="https://example.com/logo.png"
									/>
								</div>

								<div class="space-y-2">
									<label
										for={`sponsor-url-${index}`}
										class="text-sm font-medium"
									>
										Website
									</label>

									<Input
										{...props}
										id={`sponsor-url-${index}`}
										bind:value={
											$formData.sponsors[index].url
										}
										placeholder="https://example.com"
									/>
								</div>
							</div>

							<div class="space-y-2">
								<label
									for={`sponsor-description-${index}`}
									class="text-sm font-medium"
								>
									Description
								</label>

								<Textarea
									{...props}
									id={`sponsor-description-${index}`}
									bind:value={
										$formData.sponsors[index].description
									}
									placeholder="Describe the sponsor contribution"
								/>
							</div>
						</div>
					{/snippet}
				</Form.Control>
			{/each}

			<Form.FieldErrors />
		</Form.Fieldset>
	</Card>

	<Card class="space-y-4 p-5">
		<div>
			<h2 class="text-lg font-semibold">Event images</h2>

			<p class="text-sm text-muted-foreground">
				Upload images or add existing CDN URLs. The first image is used
				as the default event image for Stripe.
			</p>
		</div>

		<div class="space-y-2">
			<label for="event-image-url" class="text-sm font-medium">
				Add an existing image URL
			</label>

			<div class="flex gap-2">
				<Input
					id="event-image-url"
					bind:value={imageUrl}
					type="url"
					placeholder="https://cdn.example.com/event-image.jpg"
					aria-invalid={imageUrlError ? "true" : undefined}
					onkeydown={(event) => {
						if (event.key === "Enter") {
							event.preventDefault();
							addImageUrl();
						}
					}}
				/>

				<Button type="button" onclick={addImageUrl}>Add image</Button>
			</div>

			{#if imageUrlError}
				<p class="text-sm text-destructive">
					{imageUrlError}
				</p>
			{/if}

			<p class="text-sm text-muted-foreground">
				Use a publicly accessible HTTP or HTTPS image URL.
			</p>
		</div>

		<fieldset class="space-y-3">
			<legend class="text-sm font-medium"> Event images </legend>

			<label
				for="event-image-upload"
				class="flex cursor-pointer flex-col items-center justify-center rounded-2xl border-2 border-dashed p-8 text-center transition-colors hover:border-primary"
			>
				{#if uploadingImage}
					<p class="text-sm text-muted-foreground">
						Uploading images… {uploadProgress}%
					</p>
				{:else}
					<p class="font-medium">Choose images</p>

					<p class="mt-1 text-sm text-muted-foreground">
						Select one or more PNG, JPG, WEBP, or GIF files
					</p>
				{/if}
			</label>

			<input
				id="event-image-upload"
				bind:this={imageInput}
				type="file"
				accept="image/*"
				multiple
				class="sr-only"
				onchange={onFileUpload}
				disabled={uploadingImage}
			/>

			{#if $formData.images.length > 0}
				<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
					{#each $formData.images as image, index (image)}
						<div class="space-y-2 rounded-xl border p-3">
							<div
								class="relative aspect-video overflow-hidden rounded-lg"
							>
								<img
									src={image}
									alt={`Event image ${index + 1}`}
									class="h-full w-full object-cover"
								/>

								{#if index === 0}
									<span
										class="absolute left-2 top-2 rounded-full bg-primary px-2 py-1 text-xs text-primary-foreground"
									>
										Default
									</span>
								{/if}
							</div>

							<div
								class="flex items-center justify-between gap-2"
							>
								<p
									class="truncate text-xs text-muted-foreground"
								>
									Image {index + 1}
								</p>

								<Button
									type="button"
									variant="ghost"
									size="sm"
									onclick={() => removeImage(index)}
								>
									Remove
								</Button>
							</div>
						</div>
					{/each}
				</div>
			{:else}
				<p class="text-sm text-muted-foreground">
					No event images uploaded yet.
				</p>
			{/if}
		</fieldset>
	</Card>

	<Card class="space-y-4 p-5">
		<Form.Fieldset {form} name="prices">
			<Form.Legend>Ticket prices</Form.Legend>

			<Form.Control>
				{#snippet children({ props })}
					<div class="flex items-center gap-3">
						<Checkbox
							{...props}
							checked={hasTiers}
							onCheckedChange={(value) =>
								setHasTiers(value === true)}
						/>

						<Form.Label class="font-normal">
							This event has multiple ticket tiers
						</Form.Label>
					</div>
				{/snippet}
			</Form.Control>

			{#if $formData.prices.length === 0}
				<div
					class="rounded-2xl border border-dashed p-5 text-sm text-muted-foreground"
				>
					No ticket price has been added yet.
				</div>

				<Button type="button" variant="outline" onclick={addPrice}>
					Add ticket price
				</Button>
			{:else}
				{#each $formData.prices as _, index (index)}
					{#if hasTiers || index === 0}
						<Card class="space-y-5 border p-4 shadow-none">
							<div
								class="flex items-center justify-between gap-4"
							>
								<h3 class="font-medium">
									{hasTiers
										? `Ticket tier ${index + 1}`
										: "Ticket price"}
								</h3>

								{#if hasTiers}
									<Button
										type="button"
										variant="ghost"
										onclick={() => removePrice(index)}
									>
										Remove
									</Button>
								{/if}
							</div>

							<div class="grid gap-4 md:grid-cols-2">
								<Form.Control>
									{#snippet children({ props })}
										<Form.Label>Tier name</Form.Label>

										<Input
											{...props}
											bind:value={
												$formData.prices[index].nickname
											}
											placeholder="VIP Experience"
										/>
									{/snippet}
								</Form.Control>

								<Form.Control>
									{#snippet children({ props })}
										<Form.Label>Price</Form.Label>

										<div class="flex items-center gap-2">
											<span
												class="text-sm text-muted-foreground"
											>
												$
											</span>

											<Input
												{...props}
												type="number"
												inputmode="decimal"
												min="0"
												value={(
													$formData.prices[index]
														.unit_amount / 100
												).toFixed(2)}
												oninput={(event) =>
													setPriceAmount(
														index,
														event,
													)}
												placeholder="49.99"
											/>
										</div>
									{/snippet}
								</Form.Control>
							</div>

							<Form.Control>
								{#snippet children({ props })}
									<Form.Label>Description</Form.Label>

									<Textarea
										{...props}
										bind:value={
											$formData.prices[index].description
										}
										placeholder="What's included in this tier..."
										rows={3}
									/>
								{/snippet}
							</Form.Control>

							<div class="grid gap-4 md:grid-cols-2">
								<Form.Control>
									{#snippet children({ props })}
										<Form.Label>
											Max tickets per order
										</Form.Label>

										<Input
											{...props}
											type="number"
											min="1"
											value={$formData.prices[index]
												.quantity ?? ""}
											oninput={(event) =>
												setPriceQuantity(index, event)}
											placeholder="10"
										/>
									{/snippet}
								</Form.Control>

								<Form.Control>
									{#snippet children({ props })}
										<Form.Label>Currency</Form.Label>

										<Input
											{...props}
											value={$formData.prices[
												index
											].currency.toUpperCase()}
											readonly
										/>
									{/snippet}
								</Form.Control>
							</div>

							<fieldset class="space-y-3">
								<div
									class="flex items-center justify-between gap-4"
								>
									<legend>Features</legend>

									<Button
										type="button"
										variant="outline"
										size="sm"
										onclick={() => addPriceFeature(index)}
									>
										Add feature
									</Button>
								</div>

								{#if $formData.prices[index].features.length === 0}
									<p class="text-sm text-muted-foreground">
										No features added yet.
									</p>
								{:else}
									{#each $formData.prices[index].features as _, featureIndex (featureIndex)}
										<Form.Control>
											{#snippet children({ props })}
												<div class="flex gap-2">
													<Input
														{...props}
														bind:value={
															$formData.prices[
																index
															].features[
																featureIndex
															]
														}
														placeholder="Meet & Greet, Premium Parking"
													/>

													<Button
														type="button"
														variant="ghost"
														onclick={() =>
															removePriceFeature(
																index,
																featureIndex,
															)}
													>
														Remove
													</Button>
												</div>
											{/snippet}
										</Form.Control>
									{/each}
								{/if}
							</fieldset>

							<div class="flex flex-wrap gap-4">
								<Form.Control>
									{#snippet children({ props })}
										<div class="flex items-center gap-2">
											<Checkbox
												{...props}
												checked={$formData.prices[index]
													.requires_submission}
												onCheckedChange={(value) =>
													setRequiresSubmission(
														index,
														value === true,
													)}
											/>

											<Form.Label>
												Requires vehicle submission
											</Form.Label>
										</div>
									{/snippet}
								</Form.Control>

								{#if $formData.prices[index].requires_submission}
									<Form.Control>
										{#snippet children({ props })}
											<div
												class="flex items-center gap-2"
											>
												<Checkbox
													{...props}
													checked={$formData.prices[
														index
													].requires_approval}
													onCheckedChange={(value) =>
														($formData.prices[
															index
														].requires_approval =
															value === true)}
												/>

												<Form.Label>
													Requires approval
												</Form.Label>
											</div>
										{/snippet}
									</Form.Control>
								{/if}

								<Form.Control>
									{#snippet children({ props })}
										<div class="flex items-center gap-2">
											<Checkbox
												{...props}
												checked={$formData.prices[index]
													.most_popular}
												onCheckedChange={(value) =>
													($formData.prices[
														index
													].most_popular =
														value === true)}
											/>

											<Form.Label>
												Most popular tier
											</Form.Label>
										</div>
									{/snippet}
								</Form.Control>

								<Form.Control>
									{#snippet children({ props })}
										<div class="flex items-center gap-2">
											<Checkbox
												{...props}
												checked={$formData.prices[index]
													.sold_out ?? false}
												onCheckedChange={(value) =>
													($formData.prices[
														index
													].sold_out =
														value === true)}
											/>

											<Form.Label>Sold out</Form.Label>
										</div>
									{/snippet}
								</Form.Control>
							</div>
						</Card>
					{/if}
				{/each}

				{#if hasTiers}
					<Button type="button" variant="outline" onclick={addPrice}>
						Add tier
					</Button>
				{/if}
			{/if}

			<Form.FieldErrors />
		</Form.Fieldset>
	</Card>

	<div class="flex justify-end border-t pt-4">
		<Button type="submit" disabled={$submitting}>
			{$submitting ? "Saving…" : "Save event"}
		</Button>
	</div>
</form>
