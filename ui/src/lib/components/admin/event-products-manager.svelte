<script lang="ts">
	import { onMount } from "svelte";
	import { toast } from "svelte-sonner";

	import { apiClient } from "$lib/api";
	import { Button } from "$lib/components/ui/button";
	import { Card } from "$lib/components/ui/card";
	import { Checkbox } from "$lib/components/ui/checkbox";

	import type { IncludedProduct } from "$lib/schemas/event";
	import type { Price } from "$lib/schemas/price";
	import type { Product } from "$lib/schemas/product";

	type Tab = "linked" | "prices" | "add";

	interface Props {
		/**
		 * Database event ID.
		 *
		 * Used by admin mutation endpoints.
		 */
		eventId: string;
		eventName: string;
		prices?: Price[];
	}

	interface LinkedProduct {
		id: string;
		name: string;
		description?: string;
		images?: string[];
		active: boolean;
		sort_order: number;
		default_price?: {
			id: string;
			unit_amount: number;
			currency: string;
		};
	}

	interface TierProducts {
		tierId: string;
		tierName: string;
		amount: number;
		currency: string;
		included_products?: IncludedProduct[];
	}

	interface LinkedProductsResponse {
		event_id: string;
		event_name: string;
		linked_products?: LinkedProduct[];
		tier_products?: TierProducts[];
	}

	let { eventId, eventName, prices = [] }: Props = $props();

	let activeTab = $state<Tab>("linked");

	let allProducts = $state<Product[]>([]);
	let linkedProductIds = $state<string[]>([]);
	let tierProducts = $state<Record<string, IncludedProduct[]>>({});

	let selectedProducts = $state<string[]>([]);
	let isLoading = $state(true);
	let isSaving = $state(false);
	let errorMessage = $state("");

	let linkedProducts = $derived(
		allProducts.filter((product) => linkedProductIds.includes(product.id)),
	);

	let availableProducts = $derived(
		allProducts.filter((product) => !linkedProductIds.includes(product.id)),
	);

	const tabs: Array<{ value: Tab; label: string }> = [
		{ value: "linked", label: "Linked products" },
		{ value: "prices", label: "Tier bundles" },
		{ value: "add", label: "Add products" },
	];

	function normalizeProducts(products: Product[]): Product[] {
		return products.filter((product) => product.type !== "event");
	}

	async function loadCatalogProducts() {
		const response = await apiClient.get<{ products: Product[] }>(
			"/products",
		);

		allProducts = normalizeProducts(response.products ?? []);
	}

	async function loadEventProducts() {
		const response = await apiClient.get<LinkedProductsResponse>(
			`/events/${encodeURIComponent(eventId)}/linked-products`,
		);

		linkedProductIds = (response.linked_products ?? []).map(
			(product) => product.id,
		);

		const nextTierProducts: Record<string, IncludedProduct[]> = {};

		for (const tier of response.tier_products ?? []) {
			nextTierProducts[tier.tierId] = tier.included_products ?? [];
		}

		tierProducts = nextTierProducts;
	}

	async function reload() {
		isLoading = true;
		errorMessage = "";

		try {
			await loadCatalogProducts();
			await loadEventProducts();
		} catch (error) {
			console.error("Loading event products failed:", error);
			errorMessage = "Unable to load event products.";
		} finally {
			isLoading = false;
		}
	}

	function toggleProduct(productId: string, checked: boolean) {
		selectedProducts = checked
			? [...selectedProducts, productId]
			: selectedProducts.filter((id) => id !== productId);
	}

	async function linkSelectedProducts() {
		if (selectedProducts.length === 0) {
			toast.error("Select at least one product.");
			return;
		}

		isSaving = true;

		try {
			await apiClient.post(`/admin/events/${eventId}/link-products`, {
				productIds: selectedProducts,
			});

			toast.success("Products linked.");
			selectedProducts = [];

			await reload();
		} catch (error) {
			console.error("Linking products failed:", error);
			toast.error("Unable to link products.");
		} finally {
			isSaving = false;
		}
	}

	async function unlinkProduct(productId: string) {
		isSaving = true;

		try {
			await apiClient.delete(
				`/admin/events/${eventId}/products/${productId}`,
			);

			toast.success("Product unlinked.");
			await reload();
		} catch (error) {
			console.error("Unlinking product failed:", error);
			toast.error("Unable to unlink product.");
		} finally {
			isSaving = false;
		}
	}

	async function saveTierProducts(
		tierId: string,
		products: IncludedProduct[],
	) {
		isSaving = true;

		try {
			const response = await apiClient.put<{
				included_products?: IncludedProduct[];
			}>(`/admin/events/${eventId}/tiers/${tierId}/products`, {
				included_products: products.map((product) => ({
					product_id: product.id,
					quantity: product.quantity,
				})),
			});

			tierProducts = {
				...tierProducts,
				[tierId]: response.included_products ?? products,
			};
		} catch (error) {
			console.error("Updating tier products failed:", error);
			throw error;
		} finally {
			isSaving = false;
		}
	}

	async function addProductToTier(tierId: string, productId: string) {
		const currentProducts = tierProducts[tierId] ?? [];

		if (currentProducts.some((product) => product.id === productId)) {
			toast.error("Product is already included in this tier.");
			return;
		}

		const product = allProducts.find((item) => item.id === productId);

		if (!product) {
			toast.error("Product not found.");
			return;
		}

		try {
			await saveTierProducts(tierId, [
				...currentProducts,
				{
					id: product.id,
					name: product.name,
					description: product.description,
					images: product.images,
					quantity: 1,
				},
			]);

			toast.success("Product added to tier.");
		} catch {
			toast.error("Unable to add product to tier.");
		}
	}

	async function removeProductFromTier(tierId: string, productId: string) {
		const currentProducts = tierProducts[tierId] ?? [];

		try {
			await saveTierProducts(
				tierId,
				currentProducts.filter((product) => product.id !== productId),
			);

			toast.success("Product removed from tier.");
		} catch {
			toast.error("Unable to remove product from tier.");
		}
	}

	async function updateProductQuantity(
		tierId: string,
		productId: string,
		quantity: number,
	) {
		if (quantity < 1) return;

		const currentProducts = tierProducts[tierId] ?? [];

		const updatedProducts = currentProducts.map((product) =>
			product.id === productId ? { ...product, quantity } : product,
		);

		try {
			await saveTierProducts(tierId, updatedProducts);
			toast.success("Quantity updated.");
		} catch {
			toast.error("Unable to update quantity.");
		}
	}

	function formatProductPrice(product: Product) {
		return `$${(product.price / 100).toFixed(2)}`;
	}

	function formatTierPrice(price: Price) {
		return `$${(price.unit_amount / 100).toFixed(2)} ${price.currency.toUpperCase()}`;
	}

	onMount(reload);
</script>

<Card class="space-y-6 p-5">
	<div>
		<h2 class="text-lg font-semibold">Event products and add-ons</h2>

		<p class="text-sm text-muted-foreground">
			Manage products linked to {eventName}.
		</p>
	</div>

	<div class="grid grid-cols-3 rounded-xl border p-1">
		{#each tabs as tab (tab.value)}
			<button
				type="button"
				class={[
					"rounded-lg px-3 py-2 text-sm transition-colors",
					activeTab === tab.value
						? "bg-foreground text-background"
						: "text-muted-foreground hover:text-foreground",
				]}
				onclick={() => (activeTab = tab.value)}
			>
				{tab.label}
			</button>
		{/each}
	</div>

	{#if errorMessage}
		<p
			role="alert"
			class="rounded-xl border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive"
		>
			{errorMessage}
		</p>
	{/if}

	{#if isLoading}
		<p class="text-sm text-muted-foreground">Loading event products…</p>
	{:else if activeTab === "linked"}
		<section class="space-y-4">
			<p class="text-sm text-muted-foreground">
				These products are available as add-ons during event checkout.
			</p>

			{#if linkedProducts.length === 0}
				<div class="rounded-xl border border-dashed p-8 text-center">
					<p class="text-sm text-muted-foreground">
						No products are linked to this event.
					</p>
				</div>
			{:else}
				<div class="space-y-2">
					{#each linkedProducts as product (product.id)}
						<div
							class="flex items-center justify-between gap-4 rounded-xl border p-3"
						>
							<div class="flex min-w-0 items-center gap-3">
								{#if product.images[0]}
									<img
										src={product.images[0]}
										alt={product.name}
										class="h-12 w-12 rounded object-cover"
									/>
								{:else}
									<div
										class="flex h-12 w-12 items-center justify-center rounded bg-muted text-xs"
										aria-hidden="true"
									>
										—
									</div>
								{/if}

								<div class="min-w-0">
									<p class="truncate font-medium">
										{product.name}
									</p>

									<p class="text-sm text-muted-foreground">
										{formatProductPrice(product)}
									</p>
								</div>
							</div>

							<Button
								type="button"
								variant="ghost"
								disabled={isSaving}
								onclick={() => unlinkProduct(product.id)}
							>
								Unlink
							</Button>
						</div>
					{/each}
				</div>
			{/if}
		</section>
	{:else if activeTab === "prices"}
		<section class="space-y-4">
			<p class="text-sm text-muted-foreground">
				Include products with specific ticket prices. Customers buying
				those prices will receive the included products.
			</p>

			{#if prices.length === 0}
				<div class="rounded-xl border border-dashed p-8 text-center">
					<p class="text-sm text-muted-foreground">
						This event does not have any ticket prices.
					</p>
				</div>
			{:else}
				{#each prices as price (price.id)}
					<Card class="space-y-4 border p-4 shadow-none">
						<div>
							<h3 class="font-medium">
								{price.nickname || "Standard"}
							</h3>

							<p class="text-sm text-muted-foreground">
								{formatTierPrice(price)}
							</p>
						</div>

						{#if (tierProducts[price.id] ?? []).length === 0}
							<p
								class="rounded-xl border border-dashed p-4 text-center text-sm text-muted-foreground"
							>
								No products included in this tier.
							</p>
						{:else}
							<div class="space-y-2">
								{#each tierProducts[price.id] ?? [] as product (product.id)}
									<div
										class="flex items-center justify-between gap-3 rounded-lg bg-muted p-2"
									>
										<p
											class="min-w-0 flex-1 truncate text-sm font-medium"
										>
											{product.name}
										</p>

										<div class="flex items-center gap-1">
											<Button
												type="button"
												variant="ghost"
												size="sm"
												disabled={isSaving ||
													product.quantity <= 1}
												onclick={() =>
													updateProductQuantity(
														price.id,
														product.id,
														product.quantity - 1,
													)}
											>
												−
											</Button>

											<span
												class="min-w-10 text-center text-sm"
											>
												{product.quantity}×
											</span>

											<Button
												type="button"
												variant="ghost"
												size="sm"
												disabled={isSaving}
												onclick={() =>
													updateProductQuantity(
														price.id,
														product.id,
														product.quantity + 1,
													)}
											>
												+
											</Button>

											<Button
												type="button"
												variant="ghost"
												size="sm"
												disabled={isSaving}
												onclick={() =>
													removeProductFromTier(
														price.id,
														product.id,
													)}
											>
												Remove
											</Button>
										</div>
									</div>
								{/each}
							</div>
						{/if}

						<select
							class="w-full rounded-xl border bg-background px-3 py-2 text-sm"
							disabled={isSaving}
							onchange={(event) => {
								const select =
									event.currentTarget as HTMLSelectElement;
								const productId = select.value;

								if (productId) {
									void addProductToTier(price.id, productId);
									select.value = "";
								}
							}}
						>
							<option value=""> Add product to tier </option>

							{#each allProducts as product (product.id)}
								<option value={product.id}>
									{product.name} ·
									{formatProductPrice(product)}
								</option>
							{/each}
						</select>
					</Card>
				{/each}
			{/if}
		</section>
	{:else}
		<section class="space-y-4">
			<p class="text-sm text-muted-foreground">
				Select products to link as add-ons for this event.
			</p>

			{#if availableProducts.length === 0}
				<div class="rounded-xl border border-dashed p-8 text-center">
					<p class="text-sm text-muted-foreground">
						No available products to link.
					</p>
				</div>
			{:else}
				<div class="max-h-96 space-y-2 overflow-y-auto">
					{#each availableProducts as product (product.id)}
						<label
							class="flex cursor-pointer items-center gap-3 rounded-xl border p-3 hover:bg-muted/50"
						>
							<Checkbox
								checked={selectedProducts.includes(product.id)}
								disabled={isSaving}
								onCheckedChange={(value) =>
									toggleProduct(product.id, value === true)}
							/>

							<div class="min-w-0 flex-1">
								<p class="truncate font-medium">
									{product.name}
								</p>

								<p class="text-sm text-muted-foreground">
									{formatProductPrice(product)}
								</p>
							</div>
						</label>
					{/each}
				</div>

				<Button
					type="button"
					class="w-full"
					disabled={isSaving || selectedProducts.length === 0}
					onclick={linkSelectedProducts}
				>
					Link selected products ({selectedProducts.length})
				</Button>
			{/if}
		</section>
	{/if}
</Card>
