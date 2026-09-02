<script lang="ts">
	import { onMount, untrack } from "svelte";
	import {
		superForm,
		type Infer,
		type SuperValidated,
	} from "sveltekit-superforms";
	import { zod4Client } from "sveltekit-superforms/adapters";
	import { toast } from "svelte-sonner";

	import * as Form from "$lib/components/ui/form";
	import { Input } from "$lib/components/ui/input";
	import { Button } from "$lib/components/ui/button";
	import { Checkbox } from "$lib/components/ui/checkbox";
	import { Card } from "$lib/components/ui/card";
	import { type Product, ProductSchema } from "$lib/schemas/product";
	import { apiClient } from "$lib/api";

	interface Props {
		data: {
			form: SuperValidated<Product>;
		};

		onsaved: (data: Product) => Promise<void>;
	}

	let { data, onsaved }: Props = $props();

	const form = superForm(
		untrack(() => data.form),
		{
			SPA: true,
			validators: zod4Client(ProductSchema),

			async onUpdate({ form }) {
				if (!form.valid) return;

				try {
					await onsaved(form.data);
					toast.success("Bundle saved.");
				} catch (error) {
					console.error("Saving bundle failed:", error);
					toast.error("Unable to save bundle.");
				}
			},
		},
	);

	const { form: formData, enhance, submitting } = form;

	let availableProducts = $state<Product[]>([]);
	let isLoadingProducts = $state(true);

	let selectedProductId = $state("");
	let selectedQuantity = $state(1);

	let totalValue = $derived(
		$formData.bundle_items.reduce(
			(total, item) => total + item.price * item.quantity,
			0,
		),
	);

	let bundlePrice = $derived.by(() => {
		const discount = $formData.discount_value || 0;

		if ($formData.discount_type === "percentage") {
			return Math.max(0, totalValue * (1 - discount / 100));
		}

		return Math.max(0, totalValue - discount);
	});

	let savings = $derived(Math.max(0, totalValue - bundlePrice));

	function money(value: number) {
		return `$${value.toFixed(2)}`;
	}

	function recalculatePrice() {
		$formData.price = Number(bundlePrice.toFixed(2));
	}

	async function loadProducts() {
		isLoadingProducts = true;

		try {
			const response = await apiClient.get<{ products: Product[] }>(
				"/products",
			);

			availableProducts = (response.products ?? []).filter(
				(product) =>
					product.type === "product" &&
					product.default_price !== null,
			);
		} catch (error) {
			console.error("Loading bundle products failed:", error);
			toast.error("Unable to load products.");
		} finally {
			isLoadingProducts = false;
		}
	}

	function addProduct() {
		const product = availableProducts.find(
			(entry) => entry.id === selectedProductId,
		);

		if (!product || !product.default_price) {
			toast.error("Select a valid product.");
			return;
		}

		if (
			$formData.bundle_items.some(
				(item) => item.product_id === product.id,
			)
		) {
			toast.error("That product is already in the bundle.");
			return;
		}

		$formData.bundle_items = [
			...$formData.bundle_items,
			{
				product_id: product.id,
				product_name: product.name,
				quantity: Math.max(1, selectedQuantity),
				price: product.price,
			},
		];

		selectedProductId = "";
		selectedQuantity = 1;
		recalculatePrice();
	}

	function removeProduct(productId: string) {
		$formData.bundle_items = $formData.bundle_items.filter(
			(item) => item.product_id !== productId,
		);

		recalculatePrice();
	}

	function updateQuantity(index: number, quantity: number) {
		if (quantity < 1) return;

		$formData.bundle_items[index].quantity = quantity;
		recalculatePrice();
	}

	onMount(() => {
		loadProducts();
	});
</script>

<form method="POST" use:enhance class="space-y-6">
	<Card>
		<div class="border-b p-4">
			<h2 class="text-lg font-semibold">Bundle contents</h2>

			<p class="text-sm text-muted-foreground">
				Select at least two products for this bundle.
			</p>
		</div>

		<div class="space-y-4 p-4">
			<div class="grid gap-3 md:grid-cols-[minmax(0,1fr)_7rem_auto]">
				<select
					class="rounded-2xl border bg-transparent px-3 py-2"
					bind:value={selectedProductId}
					disabled={isLoadingProducts}
				>
					<option value=""> Select a product </option>

					{#each availableProducts as product (product.id)}
						{#if product.price}
							<option value={product.id}>
								{product.name} ·
								{money(product.price)}
							</option>
						{/if}
					{/each}
				</select>

				<Input
					type="number"
					min="1"
					bind:value={selectedQuantity}
					aria-label="Product quantity"
				/>

				<Button
					type="button"
					disabled={!selectedProductId || isLoadingProducts}
					onclick={addProduct}
				>
					Add
				</Button>
			</div>

			{#if $formData.bundle_items.length === 0}
				<div
					class="rounded-2xl border border-dashed p-8 text-center text-sm text-muted-foreground"
				>
					No products added yet.
				</div>
			{:else}
				<div class="space-y-3">
					{#each $formData.bundle_items as item, index (item.product_id)}
						<div
							class="flex flex-col gap-3 rounded-xl border p-3 md:flex-row md:items-center md:justify-between"
						>
							<div>
								<p class="font-medium">
									{item.product_name}
								</p>

								<p class="text-sm text-muted-foreground">
									{money(item.price)} each ·
									{money(item.price * item.quantity)} total
								</p>
							</div>

							<div class="flex items-center gap-2">
								<Button
									type="button"
									variant="outline"
									size="sm"
									disabled={item.quantity <= 1}
									onclick={() =>
										updateQuantity(
											index,
											item.quantity - 1,
										)}
								>
									−
								</Button>

								<span class="w-8 text-center">
									{item.quantity}
								</span>

								<Button
									type="button"
									variant="outline"
									size="sm"
									onclick={() =>
										updateQuantity(
											index,
											item.quantity + 1,
										)}
								>
									+
								</Button>

								<Button
									type="button"
									variant="ghost"
									onclick={() =>
										removeProduct(item.product_id)}
								>
									Remove
								</Button>
							</div>
						</div>
					{/each}
				</div>
			{/if}

			<Form.Field {form} name="bundle_items">
				<Form.FieldErrors />
			</Form.Field>
		</div>
	</Card>

	<Card>
		<div class="border-b p-4">
			<h2 class="text-lg font-semibold">Bundle pricing</h2>

			<p class="text-sm text-muted-foreground">
				Choose a percentage or fixed discount.
			</p>
		</div>

		<div class="space-y-4 p-4">
			<div class="grid gap-4 md:grid-cols-2">
				<Form.Field {form} name="discount_type">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Discount type</Form.Label>

							<select
								{...props}
								class="w-full rounded-2xl border bg-transparent px-3 py-2"
								bind:value={$formData.discount_type}
								onchange={recalculatePrice}
							>
								<option value="percentage"> Percentage </option>

								<option value="fixed"> Fixed amount </option>
							</select>
						{/snippet}
					</Form.Control>

					<Form.FieldErrors />
				</Form.Field>

				<Form.Field {form} name="discount_value">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>
								Discount
								{$formData.discount_type === "percentage"
									? "(%)"
									: "($)"}
							</Form.Label>

							<Input
								{...props}
								type="number"
								min="0"
								step="0.01"
								bind:value={$formData.discount_value}
								oninput={recalculatePrice}
							/>
						{/snippet}
					</Form.Control>

					<Form.FieldErrors />
				</Form.Field>
			</div>

			<div class="space-y-2 rounded-2xl bg-muted p-4">
				<div class="flex justify-between text-sm">
					<span>Individual item value</span>
					<strong>{money(totalValue)}</strong>
				</div>

				<div class="flex justify-between text-sm text-emerald-600">
					<span>Savings</span>
					<strong>−{money(savings)}</strong>
				</div>

				<div
					class="flex justify-between border-t pt-2 text-lg font-semibold"
				>
					<span>Bundle price</span>
					<span>{money(bundlePrice)}</span>
				</div>
			</div>

			<Form.Field {form} name="price">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Final bundle price</Form.Label>

						<Input
							{...props}
							type="number"
							readonly
							bind:value={$formData.price}
						/>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>
		</div>
	</Card>

	<div class="grid gap-4 md:grid-cols-2">
		<Form.Field {form} name="in_stock">
			<Form.Control>
				{#snippet children({ props })}
					<div class="flex items-start gap-3 rounded-2xl border p-4">
						<Checkbox
							{...props}
							bind:checked={$formData.in_stock}
						/>

						<div>
							<Form.Label>In stock</Form.Label>

							<p class="text-sm text-muted-foreground">
								Make this bundle available for purchase.
							</p>
						</div>
					</div>
				{/snippet}
			</Form.Control>

			<Form.FieldErrors />
		</Form.Field>

		<Form.Field {form} name="max_quantity">
			<Form.Control>
				{#snippet children({ props })}
					<Form.Label>Maximum quantity per order</Form.Label>

					<Input
						{...props}
						type="number"
						min="1"
						bind:value={$formData.max_quantity}
					/>
				{/snippet}
			</Form.Control>

			<Form.FieldErrors />
		</Form.Field>
	</div>

	<div class="flex justify-end border-t pt-4">
		<Button type="submit" disabled={$submitting}>
			{$submitting ? "Saving…" : "Save bundle"}
		</Button>
	</div>
</form>

<!-- {#await loadProducts()}
	Product loading happens on mount.
{/await} -->
