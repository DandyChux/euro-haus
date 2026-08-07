<script lang="ts">
	import { untrack } from "svelte";
	import type { SuperForm } from "sveltekit-superforms";

	import { Button } from "$lib/components/ui/button";
	import { Card } from "$lib/components/ui/card";
	import { Checkbox } from "$lib/components/ui/checkbox";
	import { Input } from "$lib/components/ui/input";

	import type { ProductVariant, ProductVariants } from "$lib/schemas/product";

	interface Props {
		form: SuperForm<ProductVariants>;
	}

	let { form }: Props = $props();

	const { form: formData } = untrack(() => form);

	let hasVariants = $state(untrack(() => $formData.variants.length > 0));

	let isApparel = $derived($formData.category === "apparel");

	function emptyVariant(): ProductVariant {
		return {
			id: "",
			price_id: "",
			variant: "",
			price: 0,
			in_stock: true,
			stock_quantity: undefined,
			images: [],
		};
	}

	function addVariant() {
		hasVariants = true;

		$formData.variants = [...$formData.variants, emptyVariant()];
	}

	function removeVariant(index: number) {
		$formData.variants = $formData.variants.filter(
			(_, variantIndex) => variantIndex !== index,
		);
	}

	function moveVariant(index: number, direction: "up" | "down") {
		const nextIndex = direction === "up" ? index - 1 : index + 1;

		if (nextIndex < 0 || nextIndex >= $formData.variants.length) {
			return;
		}

		const variants = [...$formData.variants];

		[variants[index], variants[nextIndex]] = [
			variants[nextIndex],
			variants[index],
		];

		$formData.variants = variants;
	}

	function setVariantMode(value: boolean | "indeterminate") {
		hasVariants = value === true;

		if (!hasVariants) {
			$formData.variants = [];
		} else if ($formData.variants.length === 0) {
			addVariant();
		}
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

			{#if $formData.variants.length === 0}
				<p class="py-8 text-center text-sm text-muted-foreground">
					No variants added yet.
				</p>
			{:else}
				<div class="space-y-4">
					{#each $formData.variants as variant, index (variant.id || index)}
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
											$formData.variants.length - 1}
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
										bind:value={variant.variant}
										placeholder="Black shirt - Large"
									/>
								</label>

								<label class="space-y-2 text-sm font-medium">
									<span>Price</span>

									<Input
										type="number"
										min="0"
										step="0.01"
										bind:value={variant.price}
									/>
								</label>

								{#if isApparel}
									<label
										class="space-y-2 text-sm font-medium"
									>
										<span>Size</span>

										<Input
											bind:value={variant.size}
											placeholder="S, M, L, XL"
										/>
									</label>

									<label
										class="space-y-2 text-sm font-medium"
									>
										<span>Color</span>

										<Input
											bind:value={variant.color}
											placeholder="Black"
										/>
									</label>
								{/if}

								<label class="space-y-2 text-sm font-medium">
									<span>Stock quantity</span>

									<Input
										type="number"
										min="0"
										bind:value={variant.stock_quantity}
									/>

									<span class="text-xs text-muted-foreground">
										Leave empty for unlimited stock.
									</span>
								</label>
							</div>

							<label class="flex items-center gap-2 text-sm">
								<Checkbox bind:checked={variant.in_stock} />

								<span>In stock</span>
							</label>
						</Card>
					{/each}
				</div>
			{/if}
		</fieldset>
	{/if}
</section>
