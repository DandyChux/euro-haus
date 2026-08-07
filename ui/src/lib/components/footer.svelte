<script lang="ts">
	import { Mail } from "@lucide/svelte/icons";
	import Facebook from "$lib/assets/facebook.svg";
	import Youtube from "$lib/assets/youtube.svg";
	import Instagram from "$lib/assets/instagram.svg";
	import { buttonVariants } from "./ui/button";

	const socialLinks = [
		{
			name: "Instagram",
			url: "https://instagram.com/theeurohaus",
			icon: Instagram,
		},
		{
			name: "Facebook",
			url: "https://facebook.com",
			icon: Facebook,
		},
		{
			name: "YouTube",
			url: "https://youtube.com/@TheEuroHaus",
			icon: Youtube,
		},
	] as const;

	const shopLinks = [
		{ label: "Apparel", href: "/catalog" },
		{ label: "Event Tickets", href: "/events" },
	] as const;

	const communityLinks = [
		{ label: "Events", href: "/events" },
		{ label: "Car Gallery", href: "/gallery" },
	] as const;

	const legalLinks = [
		{ label: "Terms", href: "/terms" },
		{ label: "Privacy", href: "/privacy" },
		{ label: "Cookies", href: "/cookies" },
	] as const;

	const currentYear = new Date().getFullYear();
</script>

<footer class="border-t bg-background">
	<div class="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
		<div class="grid gap-8 md:grid-cols-2 lg:grid-cols-4">
			<div>
				<h2 class="text-lg font-semibold">Euro Haus</h2>
				<p class="mt-2 text-sm leading-6">
					Building a community around car enthusiasm since 2023.
				</p>

				<div class="mt-4 flex items-center gap-3">
					{#each socialLinks as social (social.name)}
						{@const Icon = social.icon}
						<a
							href={social.url}
							target="_blank"
							rel="noreferrer"
							class={buttonVariants({
								variant: "circle",
								size: "icon-sm",
							})}
							aria-label={social.name}
						>
							<img src={Icon} alt={social.name} class="size-5" />
						</a>
					{/each}
				</div>
			</div>

			<div>
				<h3 class="text-lg font-semibold">Shop</h3>
				<ul class="mt-3 space-y-2 text-sm">
					{#each shopLinks as link (link.href)}
						<li>
							<a
								href={link.href}
								data-sveltekit-preload-data="hover"
								class="transition hover:text-primary"
							>
								{link.label}
							</a>
						</li>
					{/each}
				</ul>
			</div>

			<div>
				<h3 class="text-lg font-semibold">Community</h3>
				<ul class="mt-3 space-y-2 text-sm">
					{#each communityLinks as link (link.href)}
						<li>
							<a
								href={link.href}
								data-sveltekit-preload-data="hover"
								class="transition hover:text-primary"
							>
								{link.label}
							</a>
						</li>
					{/each}
				</ul>
			</div>

			<div>
				<h3 class="text-lg font-semibold">Contact</h3>
				<ul class="mt-3 space-y-2 text-sm">
					<li class="flex items-start gap-2">
						<Mail class="mt-0.5 size-4 shrink-0" />
						<a
							href="mailto:info@theeurohaus.com"
							class="transition hover:text-primary"
						>
							info@theeurohaus.com
						</a>
					</li>
				</ul>
			</div>
		</div>

		<div class="mt-12 border-t border-border pt-6">
			<div
				class="flex flex-col items-center justify-between gap-4 md:flex-row"
			>
				<p class="text-sm">
					© {currentYear} Euro Haus. All rights reserved.
				</p>

				<ul
					class="flex flex-wrap items-center justify-center gap-4 text-sm"
				>
					{#each legalLinks as link (link.href)}
						<li>
							<a
								href={link.href}
								data-sveltekit-preload-data="hover"
								class="transition hover:text-primary"
							>
								{link.label}
							</a>
						</li>
					{/each}
				</ul>
			</div>
		</div>

		<address class="mt-6 text-center text-sm not-italic">
			Designed &amp; Developed by
			<a
				href="https://ceokoroji.dev"
				target="_blank"
				rel="noreferrer"
				class="underline transition hover:text-primary"
			>
				Chukwuma Okoroji
			</a>
		</address>
	</div>
</footer>
