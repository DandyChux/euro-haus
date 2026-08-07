import type { PageLoad } from "./$types";
import { getEventByID, getSubmission } from "$lib/services/event";
import { getDefaultPrice } from "$lib/schemas/price";

function normalize(value?: string) {
	return (value ?? "").trim().toLowerCase();
}

export const load: PageLoad = async ({ fetch, url }) => {
	const sessionId = url.searchParams.get("session");
	const submissionId = url.searchParams.get("submission");

	if (!submissionId && !sessionId) {
		return {
			type: null,
			sessionId: null,
			submission: null,
			event: null,
			matchedPriceId: null,
			error: "No recovery details were provided.",
		};
	}

	if (submissionId) {
		try {
			const submission = await getSubmission(fetch, submissionId);
			const event = await getEventByID(fetch, submission.event_id);

			const matchedPriceId =
				submission.price_id ??
				getDefaultPrice(event?.prices ?? [])?.id ??
				null;

			return {
				type: "submission",
				sessionId: null,
				submission,
				event,
				matchedPriceId,
				error: null,
			};
		} catch {
			return {
				type: "submission",
				sessionId: null,
				submission: null,
				event: null,
				matchedPriceId: null,
				error: "We could not find that submission.",
			};
		}
	}

	return {
		type: "session",
		sessionId,
		submission: null,
		event: null,
		matchedPriceId: null,
		error: null,
	};
};
