<script lang="ts">
	import { untrack } from "svelte";
	import { resolve } from "$app/paths";
	import Newsletter from "$lib/components/newsletter.svelte";
	import { Button } from "$lib/components/ui/button";
	import { Label } from "$lib/components/ui/label";
	import * as RadioGroup from "$lib/components/ui/radio-group";
	import * as Select from "$lib/components/ui/select";
	import { formatDate } from "$lib/utils";
	import {
		getPriceAmount,
		getPriceName,
		priceIsSoldOut,
		type Price,
	} from "$lib/schemas/price";
	import apiClient from "$lib/api";
	import type { VehicleSubmission } from "$lib/schemas/submission";
	import VehicleSubmissionForm from "$lib/components/vehicle-submission-form.svelte";

	let { data } = $props();

	type CheckoutState = "idle" | "submission" | "merchandise" | "loading";

	let checkoutState = $state<CheckoutState>("idle");
	let pendingPrice = $state<Price | null>(null);
	let quantitiesByPrice = $state<Record<string, number>>({});

	let selectedAddOns = $state<Array<{ price_id: string; quantity: number }>>(
		[],
	);

	let selectedPriceID = $state(
		untrack(
			() =>
				data.event.prices.find(
					(price) => price.id && !priceIsSoldOut(price),
				)?.id ??
				data.event.prices.find((price) => price.id && price.default)
					?.id ??
				data.event.prices.find((price) => price.id)?.id ??
				"",
		),
	);

	let selectedPrice = $derived(
		data.event.prices.find((price) => price.id === selectedPriceID),
	);

	function getSelectedPrice(): Price | undefined {
		return data.event.prices.find((price) => price.id === selectedPriceID);
	}

	let currentPrice = $derived(getSelectedPrice());
	$inspect(currentPrice);

	let includedProducts = $derived(currentPrice?.included_products ?? []);

	let maximumQuantity = $derived(
		currentPrice ? getPriceMaximumQuantity(currentPrice) : 1,
	);

	let currentQuantity = $derived(
		currentPrice ? getPriceQuantity(currentPrice) : 1,
	);

	let selectedAddOnTotal = $derived(
		selectedAddOns.reduce((total, selectedAddOn) => {
			const product = includedProducts.find(
				(product) =>
					product.default_price?.id === selectedAddOn.price_id,
			);

			if (!product?.default_price) {
				return total;
			}

			return (
				total +
				product.default_price.unit_amount * selectedAddOn.quantity
			);
		}, 0),
	);

	let totalAmount = $derived(
		(currentPrice ? getPriceAmount(currentPrice) * currentQuantity : 0) +
			selectedAddOnTotal / 100,
	);

	$inspect("Current price: ", currentPrice);

	function priceRequiresSubmission(price: Price): boolean {
		return String(price.requires_submission) === "true";
	}

	function getPriceMaximumQuantity(price: Price): number {
		if (priceRequiresSubmission(price)) {
			return 1;
		}

		const configuredQuantity = Number(price.quantity);

		if (!Number.isFinite(configuredQuantity) || configuredQuantity < 1) {
			return 1;
		}

		return Math.floor(configuredQuantity);
	}

	function getPriceQuantity(price: Price): number {
		const maximumQuantity = getPriceMaximumQuantity(price);
		const selectedQuantity = quantitiesByPrice[price.id];

		if (!Number.isFinite(selectedQuantity) || selectedQuantity < 1) {
			return 1;
		}

		return Math.min(Math.floor(selectedQuantity), maximumQuantity);
	}

	function setPriceQuantity(price: Price, value: string): void {
		const maximumQuantity = getPriceMaximumQuantity(price);
		const parsedQuantity = Number(value);

		const nextQuantity =
			Number.isFinite(parsedQuantity) && parsedQuantity >= 1
				? Math.min(Math.floor(parsedQuantity), maximumQuantity)
				: 1;

		quantitiesByPrice = {
			...quantitiesByPrice,
			[price.id]: nextQuantity,
		};
	}

	function openCheckout(): void {
		const price = getSelectedPrice();

		if (!price?.id) {
			return;
		}

		if (priceRequiresSubmission(price)) {
			setPriceQuantity(price, "1");

			pendingPrice = price;
			checkoutState = "submission";
			return;
		}

		void createStripeCheckout(price);
	}

	async function createStripeCheckout(price: Price): Promise<void> {
		if (data.event.status === "sold_out") {
			return;
		}

		const selectedQuantity = getPriceQuantity(price);

		checkoutState = "loading";

		try {
			const response = await apiClient.post<{
				url?: string;
				session_id?: string;
			}>("/create-event-checkout-session", {
				event_id: data.event.id,
				price_id: price.id,
				quantity: selectedQuantity,
				addon_products: selectedAddOns,
			});

			if (!response.url) {
				throw new Error("Stripe Checkout URL was not returned");
			}

			window.location.assign(response.url);
		} catch (error) {
			console.error("Unable to create Stripe Checkout session", error);
			checkoutState = "idle";
		}
	}

	async function completeVehicleSubmission(
		submission: VehicleSubmission,
	): Promise<void> {
		if (!selectedPrice) {
			throw new Error("No ticket price selected");
		}

		checkoutState = "loading";

		const result = await apiClient.post<{
			session_id: string;
			session_url: string;
			requires_approval: boolean;
		}>("/create-participant-checkout", {
			submission_id: submission.id,
			price_id: selectedPrice.id,
			event_name: data.event.name,
			quantity: 1,
			addon_products: selectedAddOns,
		});

		if (!result.session_url) {
			throw new Error("Checkout URL was not returned");
		}

		window.location.assign(result.session_url);
	}
</script>

<svelte:head>
	<title>{data.event.name} | Euro Haus</title>
	<meta
		name="description"
		content={`${data.event.description} View event details and purchase tickets.`}
	/>
</svelte:head>

<div id="top">
	<main>
		<section class="detail-hero">
			<div class="wrap detail-hero-copy">
				<a class="back-link" href={resolve("/events")}>← All events</a>
				<p class="eyebrow light">
					Event · {formatDate(data.event.date, {
						dateStyle: "short",
					})}
				</p>
				<h1>{data.event.name}</h1>
				<div class="detail-meta">
					<p>{formatDate(data.event.date)}</p>
					<p>{data.event.venue}</p>
					<p>{data.event.location}</p>
				</div>
			</div>
			<img
				src={data.event.images[0]}
				alt={`European cars at ${data.event.name}`}
			/>
		</section>

		<section class="detail-intro wrap section-pad">
			<div>
				<p class="section-label"><span>01</span> About the event</p>
			</div>
			<div>
				<h2>{data.event.description}</h2>
				<p class="whitespace-pre-wrap">{data.event.long_description}</p>
			</div>
		</section>

		<section class="detail-grid wrap">
			<div class="detail-schedule">
				<p class="eyebrow">Schedule</p>
				{#each data.event.agenda as item (item.time)}
					<div>
						<time>{item.time}</time>
						<p>{item.activity}</p>
					</div>
				{/each}
			</div>
			<div class="detail-highlights">
				<p class="eyebrow">Included</p>
				<ul>
					{#each data.event.includes as highlight (highlight)}<li>
							{highlight}
						</li>{/each}
				</ul>
			</div>
		</section>

		{#if data.event.sponsors.length > 0}
			<section
				class="sponsors-section"
				aria-labelledby="sponsors-heading"
			>
				<div class="wrap sponsors-layout">
					<div class="sponsors-heading">
						<p class="eyebrow">Partners</p>
						<h2 id="sponsors-heading">
							Our<br /><em>sponsors.</em>
						</h2>
					</div>

					<div class="sponsors-grid">
						{#each data.event.sponsors as sponsor, index (`${sponsor.name}-${index}`)}
							<article class="sponsor-card">
								<div class="sponsor-logo">
									{#if sponsor.logo}
										<img
											src={sponsor.logo}
											alt={`${sponsor.name} logo`}
										/>
									{:else}
										<span aria-hidden="true">
											{sponsor.name.slice(0, 1)}
										</span>
									{/if}
								</div>

								<div class="sponsor-content">
									<p class="sponsor-tier">{sponsor.tier}</p>
									<h3>{sponsor.name}</h3>

									{#if sponsor.description}
										<p class="sponsor-description">
											{sponsor.description}
										</p>
									{/if}

									{#if sponsor.url}
										<a
											href={sponsor.url}
											target="_blank"
											rel="noreferrer"
										>
											Visit sponsor
											<span aria-hidden="true">↗</span>
										</a>
									{/if}
								</div>
							</article>
						{/each}
					</div>
				</div>
			</section>
		{/if}

		{#if checkoutState === "submission" && pendingPrice}
			<section class="submission-panel wrap" aria-live="polite">
				{#key pendingPrice.id}
					<VehicleSubmissionForm
						data={{ form: data.form }}
						eventId={data.event.id}
						priceId={pendingPrice.id}
						ticketTier={getPriceName(pendingPrice)}
						ticketPrice={getPriceAmount(pendingPrice)}
						ticketQuantity={1}
						requirements={pendingPrice.requirements}
						onsucceed={completeVehicleSubmission}
						oncancel={() => {
							pendingPrice = null;
							checkoutState = "idle";
						}}
					/>
				{/key}
			</section>
		{:else if checkoutState === "merchandise" && pendingPrice}
			<section class="checkout-panel wrap" aria-live="polite">
				<h2>Add event merchandise?</h2>
				<p>
					Choose any available merchandise before continuing to
					Stripe.
				</p>

				{#each data.linked_products as product (product.id)}
					<label class="merchandise-option">
						<input
							type="checkbox"
							onchange={(event) => {
								if (!product.default_price) {
									return;
								}

								if (event.currentTarget.checked) {
									selectedAddOns = [
										...selectedAddOns,
										{
											price_id: product.default_price.id,
											quantity: 1,
										},
									];
								} else {
									selectedAddOns = selectedAddOns.filter(
										(item) =>
											item.price_id !==
											product.default_price?.id,
									);
								}
							}}
						/>

						<span>
							<strong>{product.name}</strong>
							<small>{product.description}</small>
						</span>
					</label>
				{/each}

				<div class="checkout-actions">
					<button
						type="button"
						onclick={() => {
							selectedAddOns = [];
							checkoutState = "idle";
						}}
					>
						Skip
					</button>

					<button
						type="button"
						onclick={() => {
							if (!pendingPrice) {
								return;
							}

							void createStripeCheckout(pendingPrice);
						}}
					>
						Continue to Stripe
					</button>
				</div>
			</section>
		{/if}

		<section class="ticket-section" id="tickets">
			<div class="wrap ticket-layout">
				<div class="ticket-heading">
					<p class="eyebrow light">Secure your place</p>
					<h2>Choose your<br /><em>ticket.</em></h2>
					<p>
						Payments are processed securely by Stripe. Ticket
						availability is limited by event capacity.
					</p>
				</div>
				<div class="ticket-panel">
					<RadioGroup.Root
						class="ticket-options"
						name="ticket-choice"
						bind:value={selectedPriceID}
						onValueChange={(value) => {
							const price = data.event.prices.find(
								(item) => item.id === value,
							);

							if (price) {
								setPriceQuantity(
									price,
									String(getPriceQuantity(price)),
								);
							}
						}}
					>
						{#each data.event.prices as price (price.id)}
							<Label
								class={`${selectedPriceID === price.id ? "active" : ""} ${
									priceIsSoldOut(price) ? "unavailable" : ""
								}`}
								for={`price-${price.id}`}
							>
								<RadioGroup.Item
									id={`price-${price.id}`}
									value={price.id}
									disabled={priceIsSoldOut(price)}
								/>

								<span>
									<strong>{getPriceName(price)}</strong>

									{#if price.description}
										<small>{price.description}</small>
									{/if}
								</span>

								<b>${getPriceAmount(price).toFixed(2)}</b>
							</Label>
						{/each}
					</RadioGroup.Root>

					{#if includedProducts.length > 0}
						<section class="add-ons">
							<p class="eyebrow">Complete your experience</p>

							<h2>Add something extra.</h2>

							<p class="add-ons-description">
								These optional products are available with your
								selected ticket. Add anything you'd like before
								continuing to checkout.
							</p>

							<div class="add-on-list">
								{#each includedProducts as product}
									{@const price = product.default_price}
									{@const selected = selectedAddOns.some(
										(addOn) => addOn.price_id === price?.id,
									)}

									<button
										type="button"
										class:selected
										class="add-on-card"
										disabled={!price}
										onclick={() => {
											if (!price) return;

											const existingIndex =
												selectedAddOns.findIndex(
													(addOn) =>
														addOn.price_id ===
														price.id,
												);

											if (existingIndex >= 0) {
												selectedAddOns =
													selectedAddOns.filter(
														(_, index) =>
															index !==
															existingIndex,
													);
											} else {
												selectedAddOns = [
													...selectedAddOns,
													{
														price_id: price.id,
														quantity:
															product.quantity ??
															1,
													},
												];
											}
										}}
									>
										{#if product.images?.length ?? 0 > 0}
											<img
												src={product.images?.[0]}
												alt={product.name}
												class="add-on-image"
											/>
										{:else}
											<div
												class="add-on-image add-on-image-placeholder"
											>
												+
											</div>
										{/if}

										<div class="add-on-content">
											<strong>{product.name}</strong>

											{#if product.description}
												<p>{product.description}</p>
											{/if}

											{#if price}
												<span>
													{new Intl.NumberFormat(
														"en-US",
														{
															style: "currency",
															currency:
																price.currency.toUpperCase(),
														},
													).format(
														price.unit_amount / 100,
													)}
												</span>
											{:else}
												<span>Price unavailable</span>
											{/if}
										</div>

										<span
											class="add-on-checkmark"
											aria-hidden="true"
										>
											{selected ? "✓" : ""}
										</span>
									</button>
								{/each}
							</div>
						</section>
					{/if}

					<div>
						<input
							type="hidden"
							name="ticketId"
							value={selectedPriceID}
						/>

						<div class="quantity-field">
							<Label for="quantity">Quantity</Label>

							{#if currentPrice && priceRequiresSubmission(currentPrice)}
								<span
									id="quantity"
									class="quantity-value"
									aria-describedby="submission-quantity-help"
								>
									1
								</span>

								<small id="submission-quantity-help">
									Submission tickets are limited to one
									vehicle per submission.
								</small>
							{:else}
								<Select.Root
									type="single"
									name="quantity"
									value={String(currentQuantity)}
									onValueChange={(value) => {
										if (currentPrice) {
											setPriceQuantity(
												currentPrice,
												value,
											);
										}
									}}
								>
									<Select.Trigger id="quantity">
										{currentQuantity}
									</Select.Trigger>

									<Select.Content>
										<Select.Group>
											<Select.Label>Quantity</Select.Label
											>

											{#each Array.from({ length: maximumQuantity }, (_, index) => index + 1) as amount (amount)}
												<Select.Item
													value={String(amount)}
													label={String(amount)}
												>
													{amount}
												</Select.Item>
											{/each}
										</Select.Group>
									</Select.Content>
								</Select.Root>

								<small>
									Maximum quantity: {maximumQuantity}
								</small>
							{/if}
						</div>

						<div class="ticket-total">
							<span>Total</span>

							<strong>
								${totalAmount.toFixed(2)}
							</strong>
						</div>
						<Button
							type="button"
							onclick={openCheckout}
							disabled={!getSelectedPrice() ||
								checkoutState === "loading" ||
								data.event.status === "sold_out"}
						>
							{checkoutState === "loading"
								? "Opening Stripe…"
								: "Continue to Stripe"}
							<span aria-hidden="true">↗</span>
						</Button>
						<p class="secure-note">
							Secure, hosted checkout. Prices shown in USD.
						</p>
					</div>
				</div>
			</div>
		</section>
		<Newsletter />
	</main>
</div>

<style>
	.detail-hero {
		padding-top: 110px;
		background: var(--foreground);
		color: var(--background);
	}
	.detail-hero-copy {
		padding-bottom: 65px;
	}
	.back-link {
		display: inline-block;
		margin-bottom: 65px;
		color: color-mix(in srgb, var(--background) 65%, var(--primary));
		font-size: 11px;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.detail-hero h1 {
		max-width: 1200px;
		margin: 20px 0 55px;
		font-family: var(--font-display);
		font-size: clamp(100px, 12vw, 210px);
		font-weight: 800;
		letter-spacing: -0.06em;
		line-height: 0.72;
		text-transform: uppercase;
	}
	.detail-meta {
		display: flex;
		gap: 35px;
	}
	.detail-meta p {
		margin: 0;
		padding-left: 14px;
		border-left: 2px solid var(--accent);
		font-size: 12px;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}
	.detail-hero > img {
		height: 68vh;
		min-height: 600px;
		filter: saturate(0.72);
	}
	.detail-intro {
		display: grid;
		grid-template-columns: 320px 1fr;
		gap: 70px;
	}
	.detail-intro h2 {
		max-width: 950px;
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(50px, 5.5vw, 92px);
		letter-spacing: -0.04em;
		line-height: 0.95;
		text-transform: uppercase;
	}
	.detail-intro > div:last-child > p {
		max-width: 730px;
		margin: 40px 0 0;
		color: var(--muted);
		font-size: 17px;
	}
	.detail-grid {
		display: grid;
		grid-template-columns: 1.1fr 0.9fr;
		padding-bottom: 130px;
	}
	.detail-schedule,
	.detail-highlights {
		padding: 55px;
		border: 1px solid var(--border);
	}
	.detail-highlights {
		border-left: 0;
	}
	.detail-schedule > div {
		display: grid;
		grid-template-columns: 100px 1fr;
		gap: 30px;
		padding: 20px 0;
		border-bottom: 1px solid var(--border);
	}
	.detail-schedule time {
		color: var(--primary);
		font-size: 11px;
		font-weight: 700;
	}
	.detail-schedule p {
		margin: 0;
	}
	.detail-highlights ul {
		margin: 0;
		padding: 0;
		list-style: none;
	}
	.detail-highlights li {
		padding: 20px 0;
		border-bottom: 1px solid var(--border);
	}
	.sponsors-section {
		padding-block: 130px;
		background: var(--background);
	}

	.sponsors-layout {
		display: grid;
		grid-template-columns: 0.75fr 1.25fr;
		gap: 100px;
		align-items: start;
	}

	.sponsors-heading h2 {
		margin: 20px 0 0;
		font-family: var(--font-display);
		font-size: clamp(72px, 8vw, 145px);
		font-weight: 800;
		letter-spacing: -0.05em;
		line-height: 0.78;
		text-transform: uppercase;
	}

	.sponsors-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 16px;
	}

	.sponsor-card {
		display: flex;
		flex-direction: column;
		min-height: 320px;
		padding: 28px;
		border: 1px solid var(--border);
		transition:
			background-color 180ms ease,
			transform 180ms ease;
	}

	.sponsor-card:hover {
		background: color-mix(in srgb, var(--primary) 6%, var(--background));
		transform: translateY(-4px);
	}

	.sponsor-logo {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 100%;
		height: 130px;
		margin-bottom: 28px;
		background: var(--secondary);
		color: var(--background);
	}

	.sponsor-logo img {
		width: 100%;
		height: 100%;
		padding: 22px;
		object-fit: contain;
	}

	.sponsor-logo span {
		font-family: var(--font-display);
		font-size: 80px;
		font-weight: 800;
		line-height: 1;
		text-transform: uppercase;
	}

	.sponsor-content {
		display: flex;
		flex: 1;
		flex-direction: column;
		align-items: flex-start;
	}

	.sponsor-tier {
		margin: 0 0 8px;
		color: var(--primary);
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.14em;
		text-transform: uppercase;
	}

	.sponsor-card h3 {
		margin: 0;
		font-family: var(--font-display);
		font-size: 36px;
		font-weight: 700;
		letter-spacing: -0.03em;
		line-height: 0.9;
		text-transform: uppercase;
	}

	.sponsor-description {
		margin: 18px 0 0;
		color: var(--muted);
		font-size: 14px;
		line-height: 1.5;
	}

	.sponsor-card a {
		margin-top: auto;
		padding-top: 24px;
		color: var(--foreground);
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.sponsor-card a:hover {
		color: var(--primary);
	}

	.sponsor-card a span {
		margin-left: 6px;
		color: var(--primary);
	}
	.ticket-section {
		padding-block: 130px;
		background: var(--secondary);
		color: var(--background);
	}
	.ticket-layout {
		display: grid;
		grid-template-columns: 0.8fr 1.2fr;
		gap: 100px;
	}
	.ticket-heading h2 {
		margin: 25px 0;
		font-family: var(--font-display);
		font-size: clamp(80px, 8vw, 145px);
		letter-spacing: -0.05em;
		line-height: 0.78;
		text-transform: uppercase;
	}
	.ticket-panel {
		padding: 20px;
		background: var(--background);
		color: var(--foreground);
	}
	:global(.ticket-options [data-slot="label"]) {
		display: grid;
		grid-template-columns: 28px 1fr auto;
		gap: 18px;
		align-items: center;
		padding: 25px;
		border: 1px solid var(--border);
		border-bottom: 0;
		cursor: pointer;
	}
	:global(.ticket-options [data-slot="label"].active) {
		border-color: var(--primary);
		background: color-mix(in srgb, var(--primary) 8%, var(--background));
	}
	:global(.ticket-options [data-slot="label"].unavailable) {
		opacity: 0.45;
	}
	:global(.ticket-options [data-slot="label"] > span) {
		display: flex;
		flex-direction: column;
		gap: 5px;
	}
	:global(.ticket-options small),
	.secure-note {
		color: var(--muted);
	}
	:global(.ticket-options b) {
		font-family: var(--font-display);
		font-size: 28px;
	}
	.quantity-field,
	.ticket-total {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.quantity-field :global([data-slot="select-trigger"]) {
		width: 96px;
	}
	.quantity-value {
		display: inline-flex;
		width: 96px;
		align-items: center;
		justify-content: space-between;
		border: 1px solid var(--border);
		border-radius: 0.5rem;
		padding: 0.55rem 0.75rem;
		font-weight: 600;
	}
	.ticket-total {
		padding: 28px 0;
	}
	.ticket-total strong {
		font-family: var(--font-display);
		font-size: 48px;
	}
	.ticket-panel :global(.button) {
		width: 100%;
		border: 0;
		cursor: pointer;
	}
	.secure-note {
		font-size: 10px;
		text-align: center;
	}
	.submission-panel {
		padding-block: 100px;
		background:
			radial-gradient(
				circle at 10% 10%,
				color-mix(in srgb, var(--primary) 12%, transparent),
				transparent 35%
			),
			var(--secondary);
	}

	.add-ons {
		padding: 28px 24px;
		background: #edf3f8;
		border: 1px solid #d9e0e6;
	}

	.eyebrow {
		margin: 0 0 16px;
		font-size: 12px;
		font-weight: 700;
		letter-spacing: 0.18em;
		text-transform: uppercase;
	}

	.add-ons h2 {
		margin: 0;
		padding-bottom: 14px;
		border-bottom: 1px solid #d9e0e6;
		font-size: 36px;
	}

	.add-ons-description {
		max-width: 600px;
		margin: 16px 0 22px;
		color: #69717d;
		line-height: 1.5;
	}

	.add-on-list {
		display: grid;
		gap: 12px;
		max-width: 520px;
	}

	.add-on-card {
		display: flex;
		align-items: center;
		width: 100%;
		padding: 14px;
		border: 1px solid #d9e0e6;
		background: #f8f9fb;
		color: inherit;
		text-align: left;
		cursor: pointer;
		transition:
			border-color 0.15s ease,
			background 0.15s ease;
	}

	.add-on-card:hover,
	.add-on-card.selected {
		border-color: #1780c5;
		background: #fff;
	}

	.add-on-card:disabled {
		cursor: not-allowed;
		opacity: 0.65;
	}

	.add-on-image {
		width: 84px;
		height: 64px;
		flex: 0 0 84px;
		object-fit: cover;
		background: #5d3676;
	}

	.add-on-image-placeholder {
		display: grid;
		place-items: center;
		color: #1b91d0;
		font-size: 28px;
		font-weight: 700;
	}

	.add-on-content {
		display: grid;
		gap: 4px;
		padding: 0 16px;
	}

	.add-on-content strong {
		font-size: 15px;
	}

	.add-on-content p {
		margin: 0;
		color: #69717d;
		font-size: 13px;
	}

	.add-on-content span {
		color: #69717d;
		font-size: 13px;
	}

	.add-on-checkmark {
		display: grid;
		place-items: center;
		width: 24px;
		height: 24px;
		margin-left: auto;
		border: 1px solid #d9e0e6;
		color: white;
		font-weight: 700;
	}

	.add-on-card.selected .add-on-checkmark {
		border-color: #1780c5;
		background: #1780c5;
	}

	@media (max-width: 800px) {
		.detail-hero {
			padding-top: 70px;
		}
		.detail-hero h1 {
			font-size: 76px;
		}
		.detail-hero > img {
			height: 420px;
			min-height: 0;
		}
		.detail-meta {
			flex-direction: column;
			gap: 15px;
		}
		.detail-intro,
		.ticket-layout {
			display: block;
		}
		.detail-intro h2 {
			font-size: 52px;
		}
		.detail-grid {
			display: block;
			padding-bottom: 80px;
		}
		.detail-highlights {
			border-top: 0;
			border-left: 1px solid var(--border);
		}
		.sponsors-section {
			padding-block: 80px;
		}

		.sponsors-layout {
			display: block;
		}

		.sponsors-heading {
			margin-bottom: 50px;
		}

		.sponsors-heading h2 {
			font-size: 76px;
		}

		.sponsors-grid {
			grid-template-columns: 1fr;
		}

		.sponsor-card {
			min-height: 290px;
		}
		.ticket-section {
			padding-block: 80px;
		}
		.ticket-heading {
			margin-bottom: 50px;
		}
		.ticket-heading h2 {
			font-size: 72px;
		}
		.ticket-panel {
			padding: 10px;
		}
		:global(.ticket-options [data-slot="label"]) {
			padding: 20px 15px;
		}
		.submission-panel {
			padding-block: 50px;
		}
	}
</style>
