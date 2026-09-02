<script lang="ts">
	import { untrack } from "svelte";
	import type { SuperForm } from "sveltekit-superforms";

	import { Button } from "$lib/components/ui/button";
	import { Card } from "$lib/components/ui/card";
	import { Checkbox } from "$lib/components/ui/checkbox";
	import { Input } from "$lib/components/ui/input";
	import type { Product } from "$lib/schemas/product";
	import type { Price } from "$lib/schemas/price";

	interface Props {
		form: SuperForm<Product>;
	}

	let { form }: Props = $props();

	const { form: formData } = untrack(() => form);

	let hasVariants = $state(untrack(() => $formData.prices.length > 0));

	let isApparel = $derived($formData.category === "apparel");

	function emptyPrice(): Price {
		return {
			id: "",
			stripe_product_id: $formData.id,
			unit_amount: 0,
			currency: $formData.currency || "usd",
			nickname: "",
			description: "",
			active: true,
			features: [],
			default: $formData.prices.length === 0,
			most_popular: false,
			requires_approval: false,
			requires_submission: false,
			requirements: [],
			included_products: [],
			quantity: undefined,
			stock_quantity: undefined,
			sold_out: false,
			size: "",
			color: "",
		};
	}

	function addVariant() {
		hasVariants = true;

		$formData.prices = [...$formData.prices, emptyPrice()];
	}

	function removeVariant(index: number) {
		$formData.prices = $formData.prices.filter(
			(_, variantIndex) => variantIndex !== index,
		);
	}

	function moveVariant(index: number, direction: "up" | "down") {
		const nextIndex = direction === "up" ? index - 1 : index + 1;

		if (nextIndex < 0 || nextIndex >= $formData.prices.length) {
			return;
		}

		const variants = [...$formData.prices];

		[variants[index], variants[nextIndex]] = [
			variants[nextIndex],
			variants[index],
		];

		$formData.prices = variants;
	}

	function setVariantMode(value: boolean | "indeterminate") {
		hasVariants = value === true;

		if (!hasVariants) {
			$formData.prices = [];
		} else if ($formData.prices.length === 0) {
			addVariant();
		}
	}

	function displayDollars(cents: number): number {
		return cents / 100;
	}

	function setVariantPrice(index: number, event: Event) {
		const rawValue = (event.currentTarget as HTMLInputElement).value;
		const dollars = Number(rawValue);

		$formData.prices[index].unit_amount = Number.isFinite(dollars)
			? Math.round(Math.max(0, dollars) * 100)
			: 0;
	}
</script>

<section class="space-y-5">
	<div class="flex items-start gap-3">
		<Checkbox checked={hasVariants} onCheckedChange={setVariantMode} />

		<div>
			<p class="font-medium">This product has variants</p>

			<p class="text-sm text-muted-foreground">
				Use variants for sizes, colors, or other product options.
			</p>
		</div>
	</div>

	{#if hasVariants}
		<fieldset class="space-y-4">
			<legend class="text-lg font-semibold"> Product variants </legend>

			<p class="text-sm text-muted-foreground">
				These variants will be created when the product is saved.
			</p>

			<div class="flex justify-end">
				<Button type="button" variant="outline" onclick={addVariant}>
					Add variant
				</Button>
			</div>

			{#if $formData.prices.length === 0}
				<p class="py-8 text-center text-sm text-muted-foreground">
					No variants added yet.
				</p>
			{:else}
				<div class="space-y-4">
					{#each $formData.prices as price, index (price.id || index)}
						<Card class="space-y-4 p-4">
							<div
								class="flex items-center justify-between gap-4"
							>
								<h3 class="font-medium">
									Variant {index + 1}
								</h3>

								<div class="flex gap-1">
									<Button
										type="button"
										variant="ghost"
										size="sm"
										disabled={index === 0}
										onclick={() => moveVariant(index, "up")}
									>
										↑
									</Button>

									<Button
										type="button"
										variant="ghost"
										size="sm"
										disabled={index ===
											$formData.prices.length - 1}
										onclick={() =>
											moveVariant(index, "down")}
									>
										↓
									</Button>

									<Button
										type="button"
										variant="ghost"
										size="sm"
										onclick={() => removeVariant(index)}
									>
										Remove
									</Button>
								</div>
							</div>

							<div class="grid gap-4 md:grid-cols-2">
								<label class="space-y-2 text-sm font-medium">
									<span>Variant name</span>

									<Input
										bind:value={price.nickname}
										placeholder="Black shirt - Large"
									/>
								</label>

								<label class="space-y-2 text-sm font-medium">
									<span>Price</span>

									<Input
										type="number"
										min="0"
										step="0.01"
										value={displayDollars(
											price.unit_amount,
										)}
										oninput={(event) =>
											setVariantPrice(index, event)}
									/>
								</label>

								{#if isApparel}
									<label
										class="space-y-2 text-sm font-medium"
									>
										<span>Size</span>

										<Input
											bind:value={price.size}
											placeholder="S, M, L, XL"
										/>
									</label>

									<label
										class="space-y-2 text-sm font-medium"
									>
										<span>Color</span>

										<Input
											bind:value={price.color}
											placeholder="Black"
										/>
									</label>
								{/if}

								<label class="space-y-2 text-sm font-medium">
									<span>Stock quantity</span>

									<Input
										type="number"
										min="0"
										value={price.stock_quantity ?? ""}
										oninput={(event) => {
											const rawValue = (
												event.currentTarget as HTMLInputElement
											).value;
											price.stock_quantity =
												rawValue.trim()
													? Math.max(
															0,
															Math.floor(
																Number(
																	rawValue,
																),
															),
														)
													: undefined;
										}}
									/>

									<span class="text-xs text-muted-foreground">
										Leave empty for unlimited stock.
									</span>
								</label>
							</div>

							<label class="flex items-center gap-2 text-sm">
								<Checkbox bind:checked={price.active} />

								<span>In stock</span>
							</label>
						</Card>
					{/each}
				</div>
			{/if}
		</fieldset>
	{/if}
</section>
