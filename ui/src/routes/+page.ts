import type { PageLoad } from "./$types";
import { getAllEvents, getUpcomingEvents } from "$lib/services/event";

export const load: PageLoad = async ({ fetch }) => {
	const YOUTUBE_CHANNEL_ID = import.meta.env.VITE_YOUTUBE_CHANNEL_ID;
	const YOUTUBE_API_KEY = import.meta.env.VITE_YOUTUBE_API_KEY;
	const signatureSlug = `oktoberfest-${new Date().getFullYear()}`;

	const [events, allEvents, latestVideo] = await Promise.all([
		getUpcomingEvents(fetch, 3),
		getAllEvents(fetch),
		fetchLatestYouTubeVideo(fetch, YOUTUBE_CHANNEL_ID, YOUTUBE_API_KEY),
	]);

	const signatureEvent =
		allEvents.find((event) => event.slug === signatureSlug) ?? null;

	return {
		title: "Home",
		upcomingEvents: events,
		latestVideo,
		signatureEvent,
	};
};

interface YouTubeVideo {
	id: string;
	title: string;
	description: string;
	thumbnailUrl: string;
}

// Function to fetch latest YouTube video
async function fetchLatestYouTubeVideo(
	fetcher: typeof fetch,
	channelId: string,
	apiKey: string,
): Promise<YouTubeVideo | null> {
	try {
		// First, get the uploads playlist ID
		const channelResponse = await fetcher(
			`https://www.googleapis.com/youtube/v3/channels?part=contentDetails&id=${channelId}&key=${apiKey}`,
		);

		const channelData = await channelResponse.json();
		const uploadsPlaylistId =
			channelData.items?.[0]?.contentDetails?.relatedPlaylists?.uploads;

		if (!uploadsPlaylistId) {
			throw new Error("No uploads playlist found");
		}

		// Then, get the latest video from the uploads playlist
		const videosResponse = await fetcher(
			`https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&playlistId=${uploadsPlaylistId}&maxResults=1&key=${apiKey}`,
		);

		const videosData = await videosResponse.json();
		const latestVideo = videosData.items[0];

		if (!latestVideo) {
			return null;
		}

		return {
			id: latestVideo.snippet.resourceId.videoId,
			title: latestVideo.snippet.title,
			description: latestVideo.snippet.description,
			thumbnailUrl: latestVideo.snippet.thumbnails.high.url,
		};
	} catch (error) {
		console.error("Error fetching YouTube video:", error);
		return null;
	}
}
