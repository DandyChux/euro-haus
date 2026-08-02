<script lang="ts">
	import { onMount, untrack } from "svelte";
	import {
		superForm,
		type Infer,
		type SuperValidated,
	} from "sveltekit-superforms";
	import { zod4Client } from "sveltekit-superforms/adapters";

	import * as Form from "$lib/components/ui/form";
	import { Input } from "$lib/components/ui/input";
	import { Textarea } from "$lib/components/ui/textarea";
	import { Button } from "$lib/components/ui/button";
	import { Checkbox } from "$lib/components/ui/checkbox";

	import { apiClient } from "$lib/api";
	import { priceEditSchema, type Price } from "$lib/schemas/price";

	type ProductType = "product" | "event";

	interface Props {
		productId: string;
		productType: ProductType;
		form: SuperValidated<Infer<typeof priceEditSchema>>;
		onchanged?: () => void;
	}

	let { productId, productType, form: formData, onchanged }: Props = $props();

	const form = superForm(
		untrack(() => formData),
		{
			SPA: true,
			validators: zod4Client(priceEditSchema),
			async onUpdate({ form }) {
				if (form.valid) {
					await saveEdit();
				}
			},
		},
	);

	const {
		form: formDataStore,
		enhance,
		errors,
		constraints,
		submitting,
	} = form;

	let prices = $state<Price[]>([]);
	let isLoading = $state(true);
	let isSaving = $state(false);

	let editingPriceId = $state<string | null>(null);
	let errorMessage = $state("");
	let statusMessage = $state("");

	function populateForm(price: Price) {
		$formDataStore.id = price.id;
		$formDataStore.nickname = price.nickname ?? "";
		$formDataStore.description = price.description ?? "";
		$formDataStore.features = [...(price.features ?? [])];
		$formDataStore.most_popular = price.most_popular;
		$formDataStore.requires_approval = price.requires_approval;
		$formDataStore.requires_submission = price.requires_submission;
	}

	async function loadPrices() {
		isLoading = true;
		errorMessage = "";

		try {
			const response = await apiClient.get<{
				prices?: Price[];
			}>(`/products/${productId}/prices`);

			prices = (response.prices ?? []).map((price) => ({
				...price,
				features: price.features ?? [],
				included_products: price.included_products ?? [],
			}));
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to load prices.";
		} finally {
			isLoading = false;
		}
	}

	function beginEdit(price: Price) {
		editingPriceId = price.id;
		populateForm(price);

		errorMessage = "";
		statusMessage = "";
	}

	function cancelEdit() {
		editingPriceId = null;
	}

	function addFeature() {
		$formDataStore.features = [...$formDataStore.features, ""];
	}

	function removeFeature(index: number) {
		$formDataStore.features = $formDataStore.features.filter(
			(_, featureIndex) => featureIndex !== index,
		);
	}

	async function saveEdit() {
		const price = prices.find((entry) => entry.id === editingPriceId);

		if (!price || isSaving) return;

		isSaving = true;
		errorMessage = "";

		try {
			await apiClient.put(`/admin/update-price/${price.id}`, {
				id: $formDataStore.id,
				nickname: $formDataStore.nickname?.trim(),
				description: $formDataStore.description?.trim(),
				features: $formDataStore.features.filter((feature) =>
					feature.trim(),
				),
				most_popular: $formDataStore.most_popular,
				requires_approval: $formDataStore.requires_approval,
				requires_submission: $formDataStore.requires_submission,
			});

			statusMessage = "Price updated.";
			editingPriceId = null;

			await loadPrices();
			onchanged?.();
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to update price.";
		} finally {
			isSaving = false;
		}
	}

	async function setDefault(price: Price) {
		try {
			await apiClient.post("/admin/set-default-price", {
				productId,
				priceId: price.id,
			});

			statusMessage = "Default price updated.";

			await loadPrices();
			onchanged?.();
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to set default price.";
		}
	}

	async function archivePrice(price: Price) {
		if (price.default) {
			errorMessage =
				"Set another price as default before archiving this price.";
			return;
		}

		if (!window.confirm(`Archive ${price.nickname || "this price"}?`)) {
			return;
		}

		try {
			await apiClient.put(`/admin/archive-price/${price.id}`, {
				active: false,
			});

			statusMessage = "Price archived.";

			await loadPrices();
			onchanged?.();
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to archive price.";
		}
	}

	onMount(loadPrices);
</script>

<section class="space-y-4" aria-labelledby="existing-prices-heading">
	<div>
		<h2 id="existing-prices-heading" class="text-lg font-semibold">
			Existing {productType === "event" ? "ticket tiers" : "prices"}
		</h2>

		<p class="text-sm text-muted-foreground">
			Edit price details without changing the Stripe amount.
		</p>
	</div>

	{#if errorMessage}
		<p
			role="alert"
			class="rounded-xl border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive"
		>
			{errorMessage}
		</p>
	{/if}

	{#if statusMessage}
		<p
			role="status"
			class="rounded-xl border border-emerald-500/40 bg-emerald-500/10 p-3 text-sm"
		>
			{statusMessage}
		</p>
	{/if}

	{#if isLoading}
		<div class="rounded-2xl border p-5 text-sm text-muted-foreground">
			Loading prices…
		</div>
	{:else if prices.length === 0}
		<div
			class="rounded-2xl border border-dashed p-5 text-sm text-muted-foreground"
		>
			No prices found.
		</div>
	{:else}
		<div class="space-y-3">
			{#each prices as price (price.id)}
				<article class="rounded-2xl border p-4">
					{#if editingPriceId === price.id}
						<form method="POST" use:enhance class="space-y-4">
							<Form.Field {form} name="nickname">
								<Form.Control>
									{#snippet children({ props })}
										<Form.Label>Price name</Form.Label>

										<Input
											{...props}
											bind:value={$formDataStore.nickname}
											{...$constraints.nickname}
										/>
									{/snippet}
								</Form.Control>

								<Form.FieldErrors />
							</Form.Field>

							{#if productType === "event"}
								<Form.Field {form} name="description">
									<Form.Control>
										{#snippet children({ props })}
											<Form.Label>Description</Form.Label>

											<Textarea
												{...props}
												bind:value={
													$formDataStore.description
												}
											/>
										{/snippet}
									</Form.Control>

									<Form.FieldErrors />
								</Form.Field>

								<div class="space-y-2">
									<div
										class="flex items-center justify-between"
									>
										<Form.Label>Features</Form.Label>

										<Button
											type="button"
											variant="outline"
											size="sm"
											onclick={addFeature}
										>
											Add feature
										</Button>
									</div>

									<Form.Fieldset {form} name={"features"}>
										<Form.Legend>Features</Form.Legend>
										{#each $formDataStore.features as _, index (index)}
											<Form.Control>
												{#snippet children({ props })}
													<div class="flex gap-2">
														<Input
															{...props}
															bind:value={
																$formDataStore
																	.features[
																	index
																]
															}
															placeholder={`Feature ${index + 1}`}
														/>

														<Button
															type="button"
															variant="ghost"
															aria-label="Remove feature"
															onclick={() =>
																removeFeature(
																	index,
																)}
														>
															×
														</Button>
													</div>
												{/snippet}
											</Form.Control>

											<Form.FieldErrors />
										{/each}
									</Form.Fieldset>
								</div>

								<Form.Field {form} name="requires_submission">
									<Form.Control>
										{#snippet children({ props })}
											<div
												class="flex items-center gap-2"
											>
												<Checkbox
													{...props}
													bind:checked={
														$formDataStore.requires_submission
													}
												/>

												<Form.Label>
													Requires vehicle submission
												</Form.Label>
											</div>
										{/snippet}
									</Form.Control>

									<Form.FieldErrors />
								</Form.Field>

								{#if $formDataStore.requires_submission}
									<Form.Field {form} name="requires_approval">
										<Form.Control>
											{#snippet children({ props })}
												<div
													class="flex items-center gap-2"
												>
													<Checkbox
														{...props}
														bind:checked={
															$formDataStore.requires_approval
														}
													/>

													<Form.Label>
														Requires approval
													</Form.Label>
												</div>
											{/snippet}
										</Form.Control>

										<Form.FieldErrors />
									</Form.Field>
								{/if}

								<Form.Field {form} name="most_popular">
									<Form.Control>
										{#snippet children({ props })}
											<div
												class="flex items-center gap-2"
											>
												<Checkbox
													{...props}
													bind:checked={
														$formDataStore.most_popular
													}
												/>

												<Form.Label>
													Most popular tier
												</Form.Label>
											</div>
										{/snippet}
									</Form.Control>

									<Form.FieldErrors />
								</Form.Field>
							{/if}

							<div class="flex gap-2">
								<Button
									type="submit"
									disabled={$submitting || isSaving}
								>
									{#if $submitting || isSaving}
										Saving…
									{:else}
										Save
									{/if}
								</Button>

								<Button
									type="button"
									variant="outline"
									onclick={cancelEdit}
								>
									Cancel
								</Button>
							</div>
						</form>
					{:else}
						<div
							class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between"
						>
							<div>
								<div class="flex flex-wrap items-center gap-2">
									<h3 class="font-medium">
										{price.nickname ||
											(productType === "event"
												? "Ticket"
												: "Price")}
									</h3>

									{#if price.default}
										<span
											class="rounded-full bg-primary px-2 py-1 text-xs"
										>
											Default
										</span>
									{/if}

									{#if price.most_popular}
										<span
											class="rounded-full border px-2 py-1 text-xs"
										>
											Most popular
										</span>
									{/if}

									{#if price.sold_out}
										<span
											class="rounded-full border border-destructive/40 px-2 py-1 text-xs text-destructive"
										>
											Sold out
										</span>
									{/if}
								</div>

								<p class="mt-1 text-sm text-muted-foreground">
									{(price.unit_amount / 100).toFixed(2)}
									{price.currency.toUpperCase()}
								</p>

								{#if price.description}
									<p
										class="mt-2 whitespace-pre-wrap text-sm text-muted-foreground"
									>
										{price.description}
									</p>
								{/if}

								{#if price.features.length > 0}
									<ul
										class="mt-2 list-inside list-disc text-sm text-muted-foreground"
									>
										{#each price.features as feature (feature)}
											<li>{feature}</li>
										{/each}
									</ul>
								{/if}
							</div>

							<div class="flex shrink-0 flex-wrap gap-2">
								<Button
									variant="outline"
									size="sm"
									disabled={!price.active}
									onclick={() => beginEdit(price)}
								>
									Edit
								</Button>

								{#if price.active && !price.default}
									<Button
										variant="outline"
										size="sm"
										onclick={() => void setDefault(price)}
									>
										Set default
									</Button>
								{/if}

								<Button
									variant="destructive"
									size="sm"
									disabled={!price.active || price.default}
									onclick={() => void archivePrice(price)}
								>
									Archive
								</Button>
							</div>
						</div>
					{/if}
				</article>
			{/each}
		</div>
	{/if}
</section>
