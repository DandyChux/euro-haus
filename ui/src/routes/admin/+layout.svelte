<script lang="ts">
	import { afterNavigate, goto } from "$app/navigation";
	import { page } from "$app/state";
	import { adminAuth, restoreAdminSession } from "$lib/stores/auth.svelte";
	import { validateAdminSession, logoutAdmin } from "$lib/auth";
	import { resolve } from "$app/paths";

	let { children } = $props();

	let ready = $state(false);
	let lastGuardedPath = $state("");

	const links = [
		{ href: "/admin", label: "Dashboard" },
		{ href: "/admin/products", label: "Products" },
		{ href: "/admin/events", label: "Events" },
		{ href: "/admin/coupons", label: "Coupons" },
		{ href: "/admin/media", label: "Media" },
		{ href: "/admin/submissions", label: "Submissions" },
		{ href: "/admin/submission-issues", label: "Submission issues" },
	];

	function isActive(href: string) {
		const pathname = page.url.pathname;

		if (href === "/admin") {
			return pathname === href;
		}

		if (href === "/admin/events" && pathname === "/admin/event-details") {
			return true;
		}

		return pathname === href || pathname.startsWith(`${href}/`);
	}

	async function guardAdminRoute() {
		restoreAdminSession();

		const valid = await validateAdminSession();
		ready = true;

		if (!valid) {
			await goto("/admin/login", { replaceState: true });
		}
	}

	async function handleLogout() {
		await logoutAdmin();
		await goto("/admin/login", { replaceState: true });
	}

	afterNavigate(() => {
		const pathname = page.url.pathname;

		// The login page must remain publicly accessible.
		if (pathname === "/admin/login") {
			ready = true;
			return;
		}

		if (!pathname.startsWith("/admin") || lastGuardedPath === pathname) {
			return;
		}

		lastGuardedPath = pathname;
		ready = false;

		void guardAdminRoute();
	});
</script>

{#if page.url.pathname === "/admin/login"}
	{@render children()}
{:else if !ready || adminAuth.checking}
	<section class="flex min-h-screen items-center justify-center px-4 py-12">
		<div
			class="rounded-3xl border border-white/10 bg-white/5 px-8 py-6 text-center text-sm"
		>
			Checking admin session…
		</div>
	</section>
{:else if adminAuth.isAuthenticated}
	<div class="min-h-screen">
		<div
			class="mx-auto grid min-h-screen max-w-7xl gap-8 px-4 py-6 sm:px-6 lg:grid-cols-[15rem_minmax(0,1fr)] lg:px-8"
		>
			<aside class="rounded-3xl border border-white/10 bg-white/5 p-5">
				<a
					href="/admin"
					class="text-sm font-semibold uppercase tracking-[0.3em]"
				>
					Euro Haus Admin
				</a>

				<nav class="mt-6 space-y-2" aria-label="Admin navigation">
					{#each links as link (link.href)}
						<a
							href={link.href}
							class={[
								"block rounded-2xl px-4 py-3 text-sm transition-colors",
								isActive(link.href)
									? "bg-white"
									: "hover:bg-white/10 hover:text-primary",
							]}
							aria-current={isActive(link.href)
								? "page"
								: undefined}
						>
							{link.label}
						</a>
					{/each}
					<div class="site-link-wrap">
						<a
							href={resolve("/")}
							class="site-link"
							aria-label="Return to the Euro Haus website"
						>
							<svg
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="1.8"
								stroke-linecap="round"
								stroke-linejoin="round"
								aria-hidden="true"
							>
								<path d="M19 12H5" />
								<path d="m12 19-7-7 7-7" />
							</svg>

							<span>
								<strong>Back to site</strong>
								<small>View the public website</small>
							</span>
						</a>
					</div>
				</nav>

				<button
					type="button"
					class="mt-8 w-full rounded-2xl border border-white/10 px-4 py-3 text-sm hover:border-white/30 hover:text-primary"
					onclick={handleLogout}
				>
					Sign out
				</button>
			</aside>

			<main class="min-w-0 space-y-6">
				{@render children()}
			</main>
		</div>
	</div>
{/if}

<style>
	.site-link-wrap {
		margin-top: 18px;
		padding-top: 18px;
		border-top: 1px solid
			color-mix(in srgb, var(--foreground) 12%, transparent);
	}

	.site-link {
		display: flex;
		align-items: center;
		gap: 12px;
		border-radius: 18px;
		padding: 12px 14px;
		color: var(--muted);
		transition:
			background-color 180ms ease,
			color 180ms ease,
			transform 180ms ease;
	}

	.site-link:hover {
		background: color-mix(in srgb, var(--primary) 10%, transparent);
		color: var(--foreground);
		transform: translateX(2px);
	}

	.site-link svg {
		width: 18px;
		height: 18px;
		flex: 0 0 auto;
	}

	.site-link span {
		display: grid;
		gap: 3px;
	}

	.site-link strong {
		font-size: 0.875rem;
		font-weight: 500;
	}

	.site-link small {
		color: var(--muted);
		font-size: 0.6875rem;
	}
</style>
