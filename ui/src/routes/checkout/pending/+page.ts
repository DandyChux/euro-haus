import { getSubmission } from "$lib/services/event";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ fetch, url }) => {
	const submissionId = url.searchParams.get("submission_id");

	if (!submissionId) {
		return {
			submission: null,
			title: "Submission pending",
		};
	}

	try {
		return {
			submission: await getSubmission(fetch, submissionId),
			title: "Submission pending",
		};
	} catch {
		return {
			submission: null,
			title: "Submission pending",
		};
	}
};
