<script lang="ts">
	type VideoSource = {
		src: string;
		type?: string; // e.g. "video/mp4", "video/webm"
		media?: string; // e.g. "(min-width: 768px)"
	};

	let {
		src,
		sources = [],
		poster,
		autoplay = true,
		loop = true,
		muted = true,
		playsinline = true,
		controls = false,
		preload = "metadata",
		class: className = "",
		width,
		height,
		...rest
	}: {
		/** Fallback video source URL. */
		src?: string;
		/** An array of source objects for multiple formats/resolutions. */
		sources?: VideoSource[];
		/** Poster image shown before the video loads/plays. */
		poster?: string;
		/** Whether the video should autoplay. Defaults to true. */
		autoplay?: boolean;
		/** Whether the video should loop. Defaults to true. */
		loop?: boolean;
		/** Whether the video should be muted. Defaults to true (required for autoplay). */
		muted?: boolean;
		/** Whether the video should play inline on mobile. Defaults to true. */
		playsinline?: boolean;
		/** Whether to show native video controls. Defaults to false. */
		controls?: boolean;
		/** Preload strategy. Defaults to "metadata". */
		preload?: "none" | "metadata" | "auto";
		/** CSS classes to apply to the <video> tag. */
		class?: string;
		/** The intrinsic width of the video. */
		width?: number | string;
		/** The intrinsic height of the video. */
		height?: number | string;
		/** Any additional attributes to spread onto the <video> tag. */
		[key: string]: any;
	} = $props();
</script>

<!--
	display: contents ensures this wrapper div doesn't interfere
	with parent Flexbox/Grid layouts, just like the Picture component.
-->
<div style="display: contents;">
	<!-- svelte-ignore a11y_media_has_caption -->
	<video
		{autoplay}
		{loop}
		{muted}
		{playsinline}
		{controls}
		{preload}
		{poster}
		{width}
		{height}
		class={className}
		{...rest}
	>
		{#each sources as source}
			<source src={source.src} type={source.type} media={source.media} />
		{/each}

		{#if src}
			<source {src} />
		{/if}
	</video>
</div>

<!-- Usage Examples -->
<!-- **Background hero video (autoplay, no controls):**
```svelte
<Video
	poster="/images/hero-poster.jpg"
	class="absolute inset-0 w-full h-full object-cover opacity-60"
	sources={[
		{ src: "/video/hero.webm", type: "video/webm" },
		{ src: "/video/hero.mp4", type: "video/mp4" },
	]}
/>
```

**Playable video with controls:**
```svelte
<Video
	src="/video/treatment-demo.mp4"
	poster="/images/treatment-poster.jpg"
	autoplay={false}
	muted={false}
	controls={true}
	class="w-full aspect-video rounded-lg"
/>
```

**Art-directed (different video for mobile vs desktop):**
```svelte
<Video
	poster="/images/poster.jpg"
	class="w-full h-full object-cover"
	sources={[
		{ src: "/video/desktop.mp4", type: "video/mp4", media: "(min-width: 768px)" },
		{ src: "/video/mobile-vertical.mp4", type: "video/mp4" },
	]}
/> -->

<style>
	source {
		display: none !important;
	}
</style>
